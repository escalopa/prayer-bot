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
	"math"
	"net/http"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/hijri"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
	"github.com/escalopa/prayer-bot/global/internal/location"
	"github.com/escalopa/prayer-bot/global/internal/prayertime"
	"github.com/escalopa/prayer-bot/global/internal/qibla"
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
	ConfigurePrayerRules(context.Context, int64, bool, int) error
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
	mux.HandleFunc("POST /api/miniapp/calendar-link", h.api(h.calendarLink))
	mux.HandleFunc("GET /api/miniapp/calendar.ics", h.calendarDownload)
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
	Prayer           *bool `json:"prayer"`
	PrePrayerMinutes int   `json:"pre_prayer_minutes"`
	Fasting          *bool `json:"fasting"`
	Kahf             *bool `json:"kahf"`
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
	if request.Prayer == nil || request.Fasting == nil || request.Kahf == nil ||
		!domain.ValidPreReminderMinutes(request.PrePrayerMinutes) {
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
	desired := reminderResponse{
		Prayer: *request.Prayer, PrePrayerMinutes: request.PrePrayerMinutes,
		Fasting: *request.Fasting, Kahf: *request.Kahf,
	}
	if !desired.Prayer {
		desired.PrePrayerMinutes = 0
	}
	if current.Prayer != desired.Prayer || current.PrePrayerMinutes != desired.PrePrayerMinutes {
		err = h.store.ConfigurePrayerRules(ctx, chatID, desired.Prayer, desired.PrePrayerMinutes)
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
	Qibla         *qiblaResponse    `json:"qibla,omitempty"`
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

type qiblaResponse struct {
	BearingDegrees     float64 `json:"bearing_degrees"`
	DistanceKilometres int     `json:"distance_kilometres"`
}

type reminderResponse struct {
	Prayer           bool `json:"prayer"`
	PrePrayerMinutes int  `json:"pre_prayer_minutes"`
	Fasting          bool `json:"fasting"`
	Kahf             bool `json:"kahf"`
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
	PreReminders []option `json:"pre_reminders"`
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
	direction, err := qibla.Calculate(profile.Latitude, profile.Longitude)
	if err != nil {
		return bootstrapResponse{}, fmt.Errorf("calculate Qibla direction: %w", err)
	}
	response.Qibla = &qiblaResponse{
		BearingDegrees:     math.Round(direction.BearingDegrees*10) / 10,
		DistanceKilometres: int(math.Round(direction.DistanceKilometres)),
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
		case domain.ReminderAt:
			state.Prayer = true
		case domain.ReminderBefore:
			state.PrePrayerMinutes = rule.OffsetMinutes
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
	for _, minutes := range domain.SupportedPreReminderMinutes() {
		label := locale.Message("pre_reminder_off")
		if minutes > 0 {
			label = fmt.Sprintf(locale.Message("minutes_before"), minutes)
		}
		result.PreReminders = append(result.PreReminders, option{Value: fmt.Sprint(minutes), Label: label})
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
		"pre_prayer_reminder": locale.Message("pre_prayer_reminder"),
		"fasting_reminders":   locale.Button("fasting_reminders"), "kahf_reminders": locale.Button("kahf_reminders"),
		"fasting_schedule": locale.Message("fasting_schedule"), "kahf_schedule": locale.Message("kahf_schedule"),
		"save": copy.Save, "saved": copy.Saved, "loading": copy.Loading,
		"location_help": copy.LocationHelp, "location_error": copy.LocationError,
		"open_in_telegram": copy.OpenInTelegram, "temporary_failure": copy.TemporaryFailure,
		"calculated_locally": copy.CalculatedLocally, "companion": copy.Companion,
		"update_location": copy.UpdateLocation,
		"tools":           copy.Tools, "qibla_title": copy.QiblaTitle, "qibla_help": copy.QiblaHelp,
		"qibla_bearing": copy.QiblaBearing, "qibla_distance": copy.QiblaDistance,
		"compass_start": copy.CompassStart, "compass_active": copy.CompassActive,
		"compass_unavailable": copy.CompassUnavailable,
		"calendar_title":      copy.CalendarTitle, "calendar_help": copy.CalendarHelp,
		"export_week": copy.ExportWeek, "export_month": copy.ExportMonth,
		"export_preparing": copy.ExportPreparing, "export_ready": copy.ExportReady,
	}
}

type miniAppCopy struct {
	Save, Saved, Loading, LocationHelp, LocationError     string
	OpenInTelegram, TemporaryFailure, CalculatedLocally   string
	Companion, UpdateLocation                             string
	Tools, QiblaTitle, QiblaHelp, QiblaBearing            string
	QiblaDistance, CompassStart, CompassActive            string
	CompassUnavailable, CalendarTitle, CalendarHelp       string
	ExportWeek, ExportMonth, ExportPreparing, ExportReady string
}

var miniCopy = map[string]miniAppCopy{
	"en": {
		Save: "Save changes", Saved: "Saved", Loading: "Loading prayer times…",
		LocationHelp:   "Share your location to calculate accurate local prayer times.",
		LocationError:  "Location access failed. Check Telegram's location permission and try again.",
		OpenInTelegram: "Open this page from the bot inside Telegram.", TemporaryFailure: "Something went wrong. Please try again.",
		CalculatedLocally: "Calculated locally for your saved timezone", Companion: "Prayer companion", UpdateLocation: "Update location",
		Tools: "Tools", QiblaTitle: "Qibla direction", QiblaHelp: "The arrow points from north toward the Kaaba.",
		QiblaBearing: "{bearing}° from north", QiblaDistance: "Approximately {distance} km to the Kaaba",
		CompassStart: "Use live compass", CompassActive: "Live compass active",
		CompassUnavailable: "An absolute compass is unavailable on this device. Use the bearing above.",
		CalendarTitle:      "Prayer calendar", CalendarHelp: "Export prayer times to Apple, Google, Outlook, or another calendar.",
		ExportWeek: "Export 7 days", ExportMonth: "Export 30 days", ExportPreparing: "Preparing calendar…", ExportReady: "Calendar ready",
	},
	"ar": {
		Save: "حفظ التغييرات", Saved: "تم الحفظ", Loading: "جارٍ تحميل مواقيت الصلاة…",
		LocationHelp:   "شارك موقعك لحساب مواقيت الصلاة المحلية بدقة.",
		LocationError:  "تعذر الوصول إلى الموقع. تحقق من إذن الموقع في تيليجرام وحاول مجددًا.",
		OpenInTelegram: "افتح هذه الصفحة من داخل البوت في تيليجرام.", TemporaryFailure: "حدث خطأ. حاول مرة أخرى.",
		CalculatedLocally: "محسوبة محليًا حسب منطقتك الزمنية", Companion: "رفيق الصلاة", UpdateLocation: "تحديث الموقع",
		Tools: "الأدوات", QiblaTitle: "اتجاه القبلة", QiblaHelp: "يشير السهم من الشمال نحو الكعبة.",
		QiblaBearing: "{bearing}° من الشمال", QiblaDistance: "حوالي {distance} كم إلى الكعبة",
		CompassStart: "استخدام البوصلة المباشرة", CompassActive: "البوصلة المباشرة مفعّلة",
		CompassUnavailable: "البوصلة المطلقة غير متاحة على هذا الجهاز. استخدم الزاوية الموضحة أعلاه.",
		CalendarTitle:      "تقويم الصلاة", CalendarHelp: "صدّر مواقيت الصلاة إلى تقويم Apple أو Google أو Outlook أو أي تقويم آخر.",
		ExportWeek: "تصدير 7 أيام", ExportMonth: "تصدير 30 يومًا", ExportPreparing: "جارٍ إعداد التقويم…", ExportReady: "التقويم جاهز",
	},
	"es": {
		Save: "Guardar cambios", Saved: "Guardado", Loading: "Cargando horarios de oración…",
		LocationHelp:   "Comparte tu ubicación para calcular horarios locales precisos.",
		LocationError:  "No se pudo obtener la ubicación. Revisa el permiso de Telegram e inténtalo de nuevo.",
		OpenInTelegram: "Abre esta página desde el bot en Telegram.", TemporaryFailure: "Algo salió mal. Inténtalo de nuevo.",
		CalculatedLocally: "Calculado localmente para tu zona horaria", Companion: "Compañero de oración", UpdateLocation: "Actualizar ubicación",
		Tools: "Herramientas", QiblaTitle: "Dirección de la alquibla", QiblaHelp: "La flecha apunta desde el norte hacia la Kaaba.",
		QiblaBearing: "{bearing}° desde el norte", QiblaDistance: "Aproximadamente {distance} km hasta la Kaaba",
		CompassStart: "Usar brújula en vivo", CompassActive: "Brújula en vivo activa",
		CompassUnavailable: "Este dispositivo no ofrece una brújula absoluta. Usa el ángulo indicado arriba.",
		CalendarTitle:      "Calendario de oración", CalendarHelp: "Exporta los horarios a Apple, Google, Outlook u otro calendario.",
		ExportWeek: "Exportar 7 días", ExportMonth: "Exportar 30 días", ExportPreparing: "Preparando calendario…", ExportReady: "Calendario listo",
	},
	"fr": {
		Save: "Enregistrer", Saved: "Enregistré", Loading: "Chargement des horaires de prière…",
		LocationHelp:   "Partagez votre position pour calculer des horaires locaux précis.",
		LocationError:  "Impossible d'accéder à la position. Vérifiez l'autorisation Telegram et réessayez.",
		OpenInTelegram: "Ouvrez cette page depuis le bot dans Telegram.", TemporaryFailure: "Une erreur est survenue. Réessayez.",
		CalculatedLocally: "Calculé localement pour votre fuseau horaire", Companion: "Compagnon de prière", UpdateLocation: "Actualiser le lieu",
		Tools: "Outils", QiblaTitle: "Direction de la Qibla", QiblaHelp: "La flèche indique la direction de la Kaaba depuis le nord.",
		QiblaBearing: "{bearing}° depuis le nord", QiblaDistance: "Environ {distance} km jusqu’à la Kaaba",
		CompassStart: "Utiliser la boussole", CompassActive: "Boussole active",
		CompassUnavailable: "La boussole absolue n’est pas disponible sur cet appareil. Utilisez l’angle ci-dessus.",
		CalendarTitle:      "Calendrier des prières", CalendarHelp: "Exportez les horaires vers Apple, Google, Outlook ou un autre calendrier.",
		ExportWeek: "Exporter 7 jours", ExportMonth: "Exporter 30 jours", ExportPreparing: "Préparation du calendrier…", ExportReady: "Calendrier prêt",
	},
	"ru": {
		Save: "Сохранить", Saved: "Сохранено", Loading: "Загружаем время намаза…",
		LocationHelp:   "Поделитесь геопозицией для точного расчёта местного времени намаза.",
		LocationError:  "Не удалось получить геопозицию. Проверьте разрешение Telegram и повторите попытку.",
		OpenInTelegram: "Откройте эту страницу из бота в Telegram.", TemporaryFailure: "Произошла ошибка. Попробуйте ещё раз.",
		CalculatedLocally: "Рассчитано локально для вашего часового пояса", Companion: "Помощник для намаза", UpdateLocation: "Обновить геопозицию",
		Tools: "Инструменты", QiblaTitle: "Направление Кыблы", QiblaHelp: "Стрелка показывает направление от севера к Каабе.",
		QiblaBearing: "{bearing}° от севера", QiblaDistance: "Примерно {distance} км до Каабы",
		CompassStart: "Включить компас", CompassActive: "Компас включён",
		CompassUnavailable: "Абсолютный компас недоступен на этом устройстве. Используйте угол выше.",
		CalendarTitle:      "Календарь намаза", CalendarHelp: "Экспортируйте время намаза в Apple, Google, Outlook или другой календарь.",
		ExportWeek: "Экспорт на 7 дней", ExportMonth: "Экспорт на 30 дней", ExportPreparing: "Готовим календарь…", ExportReady: "Календарь готов",
	},
	"tr": {
		Save: "Değişiklikleri kaydet", Saved: "Kaydedildi", Loading: "Namaz vakitleri yükleniyor…",
		LocationHelp:   "Doğru yerel namaz vakitleri için konumunuzu paylaşın.",
		LocationError:  "Konum alınamadı. Telegram konum iznini kontrol edip tekrar deneyin.",
		OpenInTelegram: "Bu sayfayı Telegram içindeki bottan açın.", TemporaryFailure: "Bir hata oluştu. Lütfen tekrar deneyin.",
		CalculatedLocally: "Kayıtlı saat diliminiz için yerel olarak hesaplandı", Companion: "Namaz yardımcısı", UpdateLocation: "Konumu güncelle",
		Tools: "Araçlar", QiblaTitle: "Kıble yönü", QiblaHelp: "Ok, kuzeyden Kâbe’ye doğru yönü gösterir.",
		QiblaBearing: "Kuzeyden {bearing}°", QiblaDistance: "Kâbe’ye yaklaşık {distance} km",
		CompassStart: "Canlı pusulayı kullan", CompassActive: "Canlı pusula etkin",
		CompassUnavailable: "Bu cihazda mutlak pusula kullanılamıyor. Yukarıdaki açıyı kullanın.",
		CalendarTitle:      "Namaz takvimi", CalendarHelp: "Namaz vakitlerini Apple, Google, Outlook veya başka bir takvime aktarın.",
		ExportWeek: "7 günü aktar", ExportMonth: "30 günü aktar", ExportPreparing: "Takvim hazırlanıyor…", ExportReady: "Takvim hazır",
	},
	"uz": {
		Save: "O‘zgarishlarni saqlash", Saved: "Saqlandi", Loading: "Namoz vaqtlari yuklanmoqda…",
		LocationHelp:   "Aniq mahalliy namoz vaqtlari uchun joylashuvingizni ulashing.",
		LocationError:  "Joylashuv olinmadi. Telegram ruxsatini tekshirib, qayta urinib ko‘ring.",
		OpenInTelegram: "Bu sahifani Telegram ichidagi botdan oching.", TemporaryFailure: "Xatolik yuz berdi. Qayta urinib ko‘ring.",
		CalculatedLocally: "Saqlangan vaqt mintaqangiz uchun mahalliy hisoblandi", Companion: "Namoz yordamchisi", UpdateLocation: "Joylashuvni yangilash",
		Tools: "Vositalar", QiblaTitle: "Qibla yo‘nalishi", QiblaHelp: "Ko‘rsatkich shimoldan Ka’ba tomon yo‘naladi.",
		QiblaBearing: "Shimoldan {bearing}°", QiblaDistance: "Ka’bagacha taxminan {distance} km",
		CompassStart: "Jonli kompasni yoqish", CompassActive: "Jonli kompas faol",
		CompassUnavailable: "Bu qurilmada mutlaq kompas mavjud emas. Yuqoridagi burchakdan foydalaning.",
		CalendarTitle:      "Namoz taqvimi", CalendarHelp: "Namoz vaqtlarini Apple, Google, Outlook yoki boshqa taqvimga eksport qiling.",
		ExportWeek: "7 kunni eksport qilish", ExportMonth: "30 kunni eksport qilish", ExportPreparing: "Taqvim tayyorlanmoqda…", ExportReady: "Taqvim tayyor",
	},
	"tt": {
		Save: "Үзгәрешләрне саклау", Saved: "Сакланды", Loading: "Намаз вакытлары йөкләнә…",
		LocationHelp:   "Төгәл җирле намаз вакытлары өчен урыныгызны бүлешегез.",
		LocationError:  "Урынны алып булмады. Telegram рөхсәтен тикшереп, кабатлап карагыз.",
		OpenInTelegram: "Бу битне Telegram эчендәге боттан ачыгыз.", TemporaryFailure: "Хата чыкты. Кабатлап карагыз.",
		CalculatedLocally: "Сакланган сәгать поясы өчен җирле исәпләнде", Companion: "Намаз ярдәмчесе", UpdateLocation: "Урынны яңарту",
		Tools: "Кораллар", QiblaTitle: "Кыйбла юнәлеше", QiblaHelp: "Ук төньяктан Кәгъбә ягына юнәлешне күрсәтә.",
		QiblaBearing: "Төньяктан {bearing}°", QiblaDistance: "Кәгъбәгә якынча {distance} км",
		CompassStart: "Тере компасны кабызу", CompassActive: "Тере компас эшли",
		CompassUnavailable: "Бу җайланмада абсолют компас юк. Өстәге почмакны кулланыгыз.",
		CalendarTitle:      "Намаз календаре", CalendarHelp: "Намаз вакытларын Apple, Google, Outlook яки башка календарьга чыгарыгыз.",
		ExportWeek: "7 көнне чыгару", ExportMonth: "30 көнне чыгару", ExportPreparing: "Календарь әзерләнә…", ExportReady: "Календарь әзер",
	},
}
