package miniapp

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/hijri"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
	"github.com/escalopa/prayer-bot/global/internal/location"
	"github.com/escalopa/prayer-bot/global/internal/prayertime"
	"github.com/escalopa/prayer-bot/global/internal/store"
)

const initDataMaxAge = 24 * time.Hour

//go:embed static/*
var embeddedStatic embed.FS

type Storage interface {
	UpsertChat(context.Context, domain.Chat) error
	Chat(context.Context, int64) (domain.Chat, error)
	SetLanguage(context.Context, int64, string) error
	Profile(context.Context, int64) (domain.PrayerProfile, error)
	UpsertProfile(context.Context, domain.PrayerProfile) (domain.PrayerProfile, error)
	EnabledRules(context.Context, int64) ([]domain.ReminderRule, error)
	EnableDefaultRules(context.Context, int64) error
	DisableRules(context.Context, int64) error
	SetWeeklyRule(context.Context, int64, domain.ReminderKind, bool) error
}

type ReminderPlanner interface {
	RebuildChat(context.Context, int64, time.Time) error
}

type Handler struct {
	botToken   string
	store      Storage
	resolver   location.Resolver
	calculator prayertime.Calculator
	planner    ReminderPlanner
	logger     *slog.Logger
	now        func() time.Time
}

func NewHandler(botToken string, storage Storage, resolver location.Resolver, calculator prayertime.Calculator, planner ReminderPlanner, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Handler{
		botToken: botToken, store: storage, resolver: resolver,
		calculator: calculator, planner: planner, logger: logger, now: time.Now,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	static, err := fs.Sub(embeddedStatic, "static")
	if err != nil {
		panic(err)
	}
	files := http.StripPrefix("/app/", http.FileServer(http.FS(static)))
	mux.Handle("GET /app/", h.staticHeaders(files))
	mux.HandleFunc("GET /app", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/app/", http.StatusPermanentRedirect)
	})
	mux.HandleFunc("POST /api/miniapp/bootstrap", h.api(h.bootstrap))
	mux.HandleFunc("PUT /api/miniapp/location", h.api(h.updateLocation))
	mux.HandleFunc("PUT /api/miniapp/preferences", h.api(h.updatePreferences))
	mux.HandleFunc("PUT /api/miniapp/settings", h.api(h.updateSettings))
	mux.HandleFunc("PUT /api/miniapp/reminders", h.api(h.updateReminders))
}

type endpoint func(http.ResponseWriter, *http.Request, Identity) error

func (h *Handler) api(next endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		identity, err := ValidateInitData(r.Header.Get("X-Telegram-Init-Data"), h.botToken, h.now(), initDataMaxAge)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if err := next(w, r, identity); err != nil {
			var apiErr *requestError
			if errors.As(err, &apiErr) {
				writeError(w, apiErr.status, apiErr.code)
				return
			}
			h.logger.Error("Mini App request failed", "path", r.URL.Path, "user_id", identity.UserID, "error", err)
			writeError(w, http.StatusInternalServerError, "temporary_failure")
		}
	}
}

func (h *Handler) staticHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' https://telegram.org; style-src 'self'; img-src 'self' data: https:; connect-src 'self'; frame-ancestors https://web.telegram.org https://*.telegram.org")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) bootstrap(w http.ResponseWriter, r *http.Request, identity Identity) error {
	data, err := h.build(r.Context(), identity)
	if err != nil {
		return err
	}
	return writeJSON(w, data)
}

type locationRequest struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

func (h *Handler) updateLocation(w http.ResponseWriter, r *http.Request, identity Identity) error {
	var request locationRequest
	if err := decodeJSON(w, r, &request); err != nil {
		return badRequest("invalid_request")
	}
	if request.Latitude == nil || request.Longitude == nil ||
		*request.Latitude < -90 || *request.Latitude > 90 || *request.Longitude < -180 || *request.Longitude > 180 {
		return badRequest("invalid_location")
	}
	if err := h.ensureChat(r.Context(), identity); err != nil {
		return err
	}
	resolved, err := h.resolver.Resolve(r.Context(), *request.Latitude, *request.Longitude)
	if err != nil {
		return fmt.Errorf("resolve location: %w", err)
	}
	latitude, longitude := domain.RoundedCoordinates(*request.Latitude, *request.Longitude)
	profile := domain.PrayerProfile{
		ChatID: identity.UserID, Latitude: latitude, Longitude: longitude,
		Timezone: resolved.Timezone, PlaceID: resolved.PlaceID,
		Method: location.RecommendedMethod(resolved.CountryCode), Madhab: domain.MadhabShafii,
		HighLatitudeRule: domain.HighLatitudeAngleBased,
	}
	current, err := h.store.Profile(r.Context(), identity.UserID)
	if err == nil {
		profile.Method = current.Method
		profile.Madhab = current.Madhab
		profile.HighLatitudeRule = current.HighLatitudeRule
		profile.Adjustments = current.Adjustments
		profile.HijriAdjustment = current.HijriAdjustment
	} else if !store.IsNotFound(err) {
		return fmt.Errorf("load profile: %w", err)
	}
	if _, err := h.store.UpsertProfile(r.Context(), profile); err != nil {
		return fmt.Errorf("save profile: %w", err)
	}
	if err := h.planner.RebuildChat(r.Context(), identity.UserID, h.now()); err != nil {
		return fmt.Errorf("rebuild reminders: %w", err)
	}
	data, err := h.build(r.Context(), identity)
	if err != nil {
		return err
	}
	data.LocationName = resolved.City
	return writeJSON(w, data)
}

type settingsRequest struct {
	Language         string         `json:"language"`
	Method           domain.Method  `json:"method"`
	Madhab           domain.Madhab  `json:"madhab"`
	HighLatitudeRule string         `json:"high_latitude_rule"`
	HijriAdjustment  int            `json:"hijri_adjustment"`
	Adjustments      map[string]int `json:"adjustments"`
}

func (h *Handler) updateSettings(w http.ResponseWriter, r *http.Request, identity Identity) error {
	var request settingsRequest
	if err := decodeJSON(w, r, &request); err != nil {
		return badRequest("invalid_request")
	}
	validated, err := validateSettings(request)
	if err != nil {
		return err
	}
	profile, err := h.store.Profile(r.Context(), identity.UserID)
	if store.IsNotFound(err) {
		return conflict("location_required")
	}
	if err != nil {
		return fmt.Errorf("load profile: %w", err)
	}
	profile.Method = request.Method
	profile.Madhab = request.Madhab
	profile.HighLatitudeRule = validated.highLatitude
	profile.HijriAdjustment = request.HijriAdjustment
	profile.Adjustments = validated.adjustments
	if err := h.store.SetLanguage(r.Context(), identity.UserID, validated.locale.Code); err != nil {
		return fmt.Errorf("save language: %w", err)
	}
	if _, err := h.store.UpsertProfile(r.Context(), profile); err != nil {
		return fmt.Errorf("save profile: %w", err)
	}
	if err := h.planner.RebuildChat(r.Context(), identity.UserID, h.now()); err != nil {
		return fmt.Errorf("rebuild reminders: %w", err)
	}
	data, err := h.build(r.Context(), Identity{UserID: identity.UserID, FirstName: identity.FirstName, LanguageCode: validated.locale.Code})
	if err != nil {
		return err
	}
	return writeJSON(w, data)
}

type remindersRequest struct {
	Prayer  *bool `json:"prayer"`
	Fasting *bool `json:"fasting"`
	Kahf    *bool `json:"kahf"`
}

type preferencesRequest struct {
	Settings  settingsRequest  `json:"settings"`
	Reminders remindersRequest `json:"reminders"`
}

type validatedSettings struct {
	locale       i18n.Locale
	highLatitude domain.HighLatitudeRule
	adjustments  domain.Adjustments
}

func validateSettings(request settingsRequest) (validatedSettings, error) {
	locale, ok := supportedLocale(request.Language)
	highLatitude := domain.HighLatitudeRule(request.HighLatitudeRule)
	if !ok || !request.Method.Valid() || !request.Madhab.Valid() || !highLatitude.Valid() ||
		request.HijriAdjustment < -2 || request.HijriAdjustment > 2 {
		return validatedSettings{}, badRequest("invalid_settings")
	}
	adjustments, err := parseAdjustments(request.Adjustments)
	if err != nil {
		return validatedSettings{}, badRequest("invalid_adjustments")
	}
	return validatedSettings{locale: locale, highLatitude: highLatitude, adjustments: adjustments}, nil
}

func validateReminders(request remindersRequest) error {
	if request.Prayer == nil || request.Fasting == nil || request.Kahf == nil {
		return badRequest("invalid_request")
	}
	return nil
}

func (h *Handler) updatePreferences(w http.ResponseWriter, r *http.Request, identity Identity) error {
	var request preferencesRequest
	if err := decodeJSON(w, r, &request); err != nil {
		return badRequest("invalid_request")
	}
	validated, err := validateSettings(request.Settings)
	if err != nil {
		return err
	}
	if err := validateReminders(request.Reminders); err != nil {
		return err
	}
	profile, err := h.store.Profile(r.Context(), identity.UserID)
	if store.IsNotFound(err) {
		return conflict("location_required")
	}
	if err != nil {
		return fmt.Errorf("load profile: %w", err)
	}
	profile.Method = request.Settings.Method
	profile.Madhab = request.Settings.Madhab
	profile.HighLatitudeRule = validated.highLatitude
	profile.HijriAdjustment = request.Settings.HijriAdjustment
	profile.Adjustments = validated.adjustments
	if err := h.store.SetLanguage(r.Context(), identity.UserID, validated.locale.Code); err != nil {
		return fmt.Errorf("save language: %w", err)
	}
	if _, err := h.store.UpsertProfile(r.Context(), profile); err != nil {
		return fmt.Errorf("save profile: %w", err)
	}
	if _, _, err := h.applyReminders(r.Context(), identity.UserID, request.Reminders); err != nil {
		return err
	}
	if err := h.planner.RebuildChat(r.Context(), identity.UserID, h.now()); err != nil {
		return fmt.Errorf("rebuild reminders: %w", err)
	}
	data, err := h.build(r.Context(), Identity{
		UserID: identity.UserID, FirstName: identity.FirstName, LanguageCode: validated.locale.Code,
	})
	if err != nil {
		return err
	}
	return writeJSON(w, data)
}

func (h *Handler) updateReminders(w http.ResponseWriter, r *http.Request, identity Identity) error {
	var request remindersRequest
	if err := decodeJSON(w, r, &request); err != nil {
		return badRequest("invalid_request")
	}
	if err := validateReminders(request); err != nil {
		return err
	}
	if _, err := h.store.Profile(r.Context(), identity.UserID); store.IsNotFound(err) {
		return conflict("location_required")
	} else if err != nil {
		return fmt.Errorf("load profile: %w", err)
	}
	changed, desired, err := h.applyReminders(r.Context(), identity.UserID, request)
	if err != nil {
		return err
	}
	if changed && (desired.Prayer || desired.Fasting || desired.Kahf) {
		if err := h.planner.RebuildChat(r.Context(), identity.UserID, h.now()); err != nil {
			return fmt.Errorf("rebuild reminders: %w", err)
		}
	}
	data, err := h.build(r.Context(), identity)
	if err != nil {
		return err
	}
	return writeJSON(w, data)
}

func (h *Handler) applyReminders(ctx context.Context, chatID int64, request remindersRequest) (bool, reminderResponse, error) {
	current, err := h.reminderState(ctx, chatID)
	if err != nil {
		return false, reminderResponse{}, err
	}
	desired := reminderResponse{Prayer: *request.Prayer, Fasting: *request.Fasting, Kahf: *request.Kahf}
	if current.Prayer != desired.Prayer {
		if *request.Prayer {
			err = h.store.EnableDefaultRules(ctx, chatID)
		} else {
			err = h.store.DisableRules(ctx, chatID)
		}
		if err != nil {
			return false, reminderResponse{}, fmt.Errorf("update prayer reminders: %w", err)
		}
	}
	if current.Fasting != desired.Fasting {
		if err := h.store.SetWeeklyRule(ctx, chatID, domain.ReminderWeeklyFasting, desired.Fasting); err != nil {
			return false, reminderResponse{}, fmt.Errorf("update fasting reminders: %w", err)
		}
	}
	if current.Kahf != desired.Kahf {
		if err := h.store.SetWeeklyRule(ctx, chatID, domain.ReminderWeeklyKahf, desired.Kahf); err != nil {
			return false, reminderResponse{}, fmt.Errorf("update Al-Kahf reminders: %w", err)
		}
	}
	return current != desired, desired, nil
}

type bootstrapResponse struct {
	User          userResponse      `json:"user"`
	Locale        string            `json:"locale"`
	NeedsLocation bool              `json:"needs_location"`
	LocationName  string            `json:"location_name,omitempty"`
	Profile       *profileResponse  `json:"profile,omitempty"`
	Today         *scheduleResponse `json:"today,omitempty"`
	Tomorrow      *scheduleResponse `json:"tomorrow,omitempty"`
	Reminders     reminderResponse  `json:"reminders"`
	Options       optionsResponse   `json:"options"`
	Labels        map[string]string `json:"labels"`
}

type userResponse struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
}

type profileResponse struct {
	Timezone         string         `json:"timezone"`
	Method           domain.Method  `json:"method"`
	Madhab           domain.Madhab  `json:"madhab"`
	HighLatitudeRule string         `json:"high_latitude_rule"`
	HijriAdjustment  int            `json:"hijri_adjustment"`
	Adjustments      map[string]int `json:"adjustments"`
}

type scheduleResponse struct {
	Gregorian string           `json:"gregorian"`
	Hijri     string           `json:"hijri"`
	Timezone  string           `json:"timezone"`
	Prayers   []prayerResponse `json:"prayers"`
}

type prayerResponse struct {
	ID    domain.Prayer `json:"id"`
	Name  string        `json:"name"`
	Emoji string        `json:"emoji"`
	Time  string        `json:"time"`
}

type reminderResponse struct {
	Prayer  bool `json:"prayer"`
	Fasting bool `json:"fasting"`
	Kahf    bool `json:"kahf"`
}

type option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type optionsResponse struct {
	Languages    []option `json:"languages"`
	Methods      []option `json:"methods"`
	Madhabs      []option `json:"madhabs"`
	HighLatitude []option `json:"high_latitude"`
}

func (h *Handler) build(ctx context.Context, identity Identity) (bootstrapResponse, error) {
	if err := h.ensureChat(ctx, identity); err != nil {
		return bootstrapResponse{}, err
	}
	chat, err := h.store.Chat(ctx, identity.UserID)
	if err != nil {
		return bootstrapResponse{}, fmt.Errorf("load chat: %w", err)
	}
	locale := i18n.Resolve(chat.LanguageCode)
	response := bootstrapResponse{
		User:   userResponse{ID: identity.UserID, FirstName: identity.FirstName},
		Locale: locale.Code, Labels: labels(locale), Options: options(locale),
	}
	response.Reminders, err = h.reminderState(ctx, identity.UserID)
	if err != nil {
		return bootstrapResponse{}, err
	}
	profile, err := h.store.Profile(ctx, identity.UserID)
	if store.IsNotFound(err) {
		response.NeedsLocation = true
		return response, nil
	}
	if err != nil {
		return bootstrapResponse{}, fmt.Errorf("load profile: %w", err)
	}
	response.Profile = &profileResponse{
		Timezone: profile.Timezone, Method: profile.Method, Madhab: profile.Madhab,
		HighLatitudeRule: string(profile.HighLatitudeRule), HijriAdjustment: profile.HijriAdjustment,
		Adjustments: adjustmentMap(profile.Adjustments),
	}
	response.LocationName = profile.Timezone
	now := h.now()
	today, err := h.calculator.Day(ctx, now, profile)
	if err != nil {
		return bootstrapResponse{}, fmt.Errorf("calculate today: %w", err)
	}
	tomorrow, err := h.calculator.Day(ctx, now.In(today.Date.Location()).AddDate(0, 0, 1), profile)
	if err != nil {
		return bootstrapResponse{}, fmt.Errorf("calculate tomorrow: %w", err)
	}
	formattedToday := formatSchedule(today, profile, locale)
	formattedTomorrow := formatSchedule(tomorrow, profile, locale)
	response.Today = &formattedToday
	response.Tomorrow = &formattedTomorrow
	return response, nil
}

func (h *Handler) ensureChat(ctx context.Context, identity Identity) error {
	language := i18n.Resolve(identity.LanguageCode).Code
	if err := h.store.UpsertChat(ctx, domain.Chat{
		TelegramChatID: identity.UserID, Type: "private", LanguageCode: language,
	}); err != nil {
		return fmt.Errorf("save chat: %w", err)
	}
	return nil
}

func (h *Handler) reminderState(ctx context.Context, chatID int64) (reminderResponse, error) {
	rules, err := h.store.EnabledRules(ctx, chatID)
	if err != nil {
		return reminderResponse{}, fmt.Errorf("load reminders: %w", err)
	}
	var state reminderResponse
	for _, rule := range rules {
		switch rule.Kind {
		case domain.ReminderWeeklyFasting:
			state.Fasting = true
		case domain.ReminderWeeklyKahf:
			state.Kahf = true
		default:
			state.Prayer = true
		}
	}
	return state, nil
}

func formatSchedule(schedule domain.DaySchedule, profile domain.PrayerProfile, locale i18n.Locale) scheduleResponse {
	result := scheduleResponse{
		Gregorian: fmt.Sprintf("%d %s %d", schedule.Date.Day(), locale.Month(int(schedule.Date.Month())), schedule.Date.Year()),
		Timezone:  profile.Timezone,
	}
	if date, err := hijri.FromGregorian(schedule.Date, profile.HijriAdjustment); err == nil {
		result.Hijri = fmt.Sprintf("%d %s %d %s", date.Day, locale.HijriMonth(date.Month), date.Year, locale.Message("hijri_era"))
	}
	for _, prayer := range prayers() {
		if at, ok := schedule.At(prayer); ok {
			result.Prayers = append(result.Prayers, prayerResponse{
				ID: prayer, Name: locale.Prayer(prayer), Emoji: prayerEmoji(prayer), Time: at.Format("15:04"),
			})
		}
	}
	return result
}

func options(locale i18n.Locale) optionsResponse {
	result := optionsResponse{}
	for _, language := range i18n.Supported() {
		result.Languages = append(result.Languages, option{Value: language.Code, Label: language.NativeName})
	}
	for _, method := range domain.SupportedMethods() {
		result.Methods = append(result.Methods, option{Value: string(method), Label: locale.Method(method)})
	}
	for _, madhab := range []domain.Madhab{domain.MadhabShafii, domain.MadhabHanafi} {
		result.Madhabs = append(result.Madhabs, option{Value: string(madhab), Label: locale.Madhab(madhab)})
	}
	for _, rule := range []domain.HighLatitudeRule{domain.HighLatitudeAngleBased, domain.HighLatitudeMiddleNight, domain.HighLatitudeSeventhNight} {
		result.HighLatitude = append(result.HighLatitude, option{Value: string(rule), Label: locale.HighLatitudeRule(rule)})
	}
	return result
}

func supportedLocale(code string) (i18n.Locale, bool) {
	for _, locale := range i18n.Supported() {
		if locale.Code == code {
			return locale, true
		}
	}
	return i18n.Locale{}, false
}

func parseAdjustments(values map[string]int) (domain.Adjustments, error) {
	if len(values) != len(prayers()) {
		return domain.Adjustments{}, fmt.Errorf("all prayer adjustments are required")
	}
	for _, prayer := range prayers() {
		value, ok := values[string(prayer)]
		if !ok || value < -30 || value > 30 {
			return domain.Adjustments{}, fmt.Errorf("invalid prayer adjustment")
		}
	}
	return domain.Adjustments{
		Fajr: values["fajr"], Sunrise: values["sunrise"], Dhuhr: values["dhuhr"],
		Asr: values["asr"], Maghrib: values["maghrib"], Isha: values["isha"],
	}, nil
}

func adjustmentMap(value domain.Adjustments) map[string]int {
	return map[string]int{
		"fajr": value.Fajr, "sunrise": value.Sunrise, "dhuhr": value.Dhuhr,
		"asr": value.Asr, "maghrib": value.Maghrib, "isha": value.Isha,
	}
}

func prayers() []domain.Prayer {
	return []domain.Prayer{domain.PrayerFajr, domain.PrayerSunrise, domain.PrayerDhuhr, domain.PrayerAsr, domain.PrayerMaghrib, domain.PrayerIsha}
}

func prayerEmoji(prayer domain.Prayer) string {
	switch prayer {
	case domain.PrayerFajr:
		return "🌙"
	case domain.PrayerSunrise:
		return "🌅"
	case domain.PrayerDhuhr:
		return "☀️"
	case domain.PrayerAsr:
		return "🌤"
	case domain.PrayerMaghrib:
		return "🌇"
	default:
		return "🌌"
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("multiple JSON values")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, value any) error {
	return json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}

type requestError struct {
	status int
	code   string
}

func (e *requestError) Error() string { return e.code }
func badRequest(code string) error    { return &requestError{status: http.StatusBadRequest, code: code} }
func conflict(code string) error      { return &requestError{status: http.StatusConflict, code: code} }

func labels(locale i18n.Locale) map[string]string {
	copy := miniCopy[locale.Code]
	return map[string]string{
		"app_title": locale.BotName, "today": locale.Button(i18n.ActionToday),
		"tomorrow": locale.Button(i18n.ActionTomorrow), "settings": locale.Button(i18n.ActionSettings),
		"reminders": locale.Button(i18n.ActionReminders), "location": locale.Button(i18n.ActionLocation),
		"share_location": locale.Button("share_location"), "language": locale.Button(i18n.ActionLanguage),
		"method": locale.Message("method"), "madhab": locale.Message("madhab"),
		"highlat": locale.Message("highlat"), "adjustments": locale.Message("adjustments"),
		"hijri": locale.Message("hijri_date"), "prayer_reminders": locale.Button("prayer_reminders"),
		"fasting_reminders": locale.Button("fasting_reminders"), "kahf_reminders": locale.Button("kahf_reminders"),
		"fasting_schedule": locale.Message("fasting_schedule"), "kahf_schedule": locale.Message("kahf_schedule"),
		"save": copy.Save, "saved": copy.Saved, "loading": copy.Loading,
		"location_help": copy.LocationHelp, "location_error": copy.LocationError,
		"open_in_telegram": copy.OpenInTelegram, "temporary_failure": copy.TemporaryFailure,
		"calculated_locally": copy.CalculatedLocally, "companion": copy.Companion,
		"update_location": copy.UpdateLocation,
	}
}

type miniAppCopy struct {
	Save, Saved, Loading, LocationHelp, LocationError   string
	OpenInTelegram, TemporaryFailure, CalculatedLocally string
	Companion, UpdateLocation                           string
}

var miniCopy = map[string]miniAppCopy{
	"en": {"Save changes", "Saved", "Loading prayer times…", "Share your location to calculate accurate local prayer times.", "Location access failed. Check Telegram's location permission and try again.", "Open this page from the bot inside Telegram.", "Something went wrong. Please try again.", "Calculated locally for your saved timezone", "Prayer companion", "Update location"},
	"ar": {"حفظ التغييرات", "تم الحفظ", "جارٍ تحميل مواقيت الصلاة…", "شارك موقعك لحساب مواقيت الصلاة المحلية بدقة.", "تعذر الوصول إلى الموقع. تحقق من إذن الموقع في تيليجرام وحاول مجددًا.", "افتح هذه الصفحة من داخل البوت في تيليجرام.", "حدث خطأ. حاول مرة أخرى.", "محسوبة محليًا حسب منطقتك الزمنية", "رفيق الصلاة", "تحديث الموقع"},
	"es": {"Guardar cambios", "Guardado", "Cargando horarios de oración…", "Comparte tu ubicación para calcular horarios locales precisos.", "No se pudo obtener la ubicación. Revisa el permiso de Telegram e inténtalo de nuevo.", "Abre esta página desde el bot en Telegram.", "Algo salió mal. Inténtalo de nuevo.", "Calculado localmente para tu zona horaria", "Compañero de oración", "Actualizar ubicación"},
	"fr": {"Enregistrer", "Enregistré", "Chargement des horaires de prière…", "Partagez votre position pour calculer des horaires locaux précis.", "Impossible d'accéder à la position. Vérifiez l'autorisation Telegram et réessayez.", "Ouvrez cette page depuis le bot dans Telegram.", "Une erreur est survenue. Réessayez.", "Calculé localement pour votre fuseau horaire", "Compagnon de prière", "Actualiser le lieu"},
	"ru": {"Сохранить", "Сохранено", "Загружаем время намаза…", "Поделитесь геопозицией для точного расчёта местного времени намаза.", "Не удалось получить геопозицию. Проверьте разрешение Telegram и повторите попытку.", "Откройте эту страницу из бота в Telegram.", "Произошла ошибка. Попробуйте ещё раз.", "Рассчитано локально для вашего часового пояса", "Помощник для намаза", "Обновить геопозицию"},
	"tr": {"Değişiklikleri kaydet", "Kaydedildi", "Namaz vakitleri yükleniyor…", "Doğru yerel namaz vakitleri için konumunuzu paylaşın.", "Konum alınamadı. Telegram konum iznini kontrol edip tekrar deneyin.", "Bu sayfayı Telegram içindeki bottan açın.", "Bir hata oluştu. Lütfen tekrar deneyin.", "Kayıtlı saat diliminiz için yerel olarak hesaplandı", "Namaz yardımcısı", "Konumu güncelle"},
	"uz": {"O‘zgarishlarni saqlash", "Saqlandi", "Namoz vaqtlari yuklanmoqda…", "Aniq mahalliy namoz vaqtlari uchun joylashuvingizni ulashing.", "Joylashuv olinmadi. Telegram ruxsatini tekshirib, qayta urinib ko‘ring.", "Bu sahifani Telegram ichidagi botdan oching.", "Xatolik yuz berdi. Qayta urinib ko‘ring.", "Saqlangan vaqt mintaqangiz uchun mahalliy hisoblandi", "Namoz yordamchisi", "Joylashuvni yangilash"},
	"tt": {"Үзгәрешләрне саклау", "Сакланды", "Намаз вакытлары йөкләнә…", "Төгәл җирле намаз вакытлары өчен урыныгызны бүлешегез.", "Урынны алып булмады. Telegram рөхсәтен тикшереп, кабатлап карагыз.", "Бу битне Telegram эчендәге боттан ачыгыз.", "Хата чыкты. Кабатлап карагыз.", "Сакланган сәгать поясы өчен җирле исәпләнде", "Намаз ярдәмчесе", "Урынны яңарту"},
}
