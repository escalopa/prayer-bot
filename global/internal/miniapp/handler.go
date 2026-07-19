package miniapp

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"net/http"
	"time"

	botapi "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/hijri"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
	"github.com/escalopa/prayer-bot/global/internal/location"
	"github.com/escalopa/prayer-bot/global/internal/occasions"
	"github.com/escalopa/prayer-bot/global/internal/prayertime"
	"github.com/escalopa/prayer-bot/global/internal/qibla"
	"github.com/escalopa/prayer-bot/global/internal/store"
)

const initDataMaxAge = 24 * time.Hour
const maxPrayerCardBytes = 5 << 20

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
	SetOccasionRule(context.Context, int64, domain.ReminderKind, bool) error
	CalendarSubscription(context.Context, int64) (domain.CalendarSubscription, error)
	CalendarSubscriptionByToken(context.Context, string) (domain.CalendarSubscription, error)
	EnableCalendarSubscription(context.Context, int64, string, string) (domain.CalendarSubscription, error)
	DisableCalendarSubscription(context.Context, int64) error
}

type ReminderPlanner interface {
	RebuildChat(context.Context, int64, time.Time) error
}

type PhotoSender interface {
	SendPhoto(context.Context, *botapi.SendPhotoParams) (*models.Message, error)
}

type Handler struct {
	botToken    string
	store       Storage
	resolver    location.Resolver
	calculator  prayertime.Calculator
	planner     ReminderPlanner
	photoSender PhotoSender
	logger      *slog.Logger
	now         func() time.Time
}

func NewHandler(
	botToken string,
	storage Storage,
	resolver location.Resolver,
	calculator prayertime.Calculator,
	planner ReminderPlanner,
	logger *slog.Logger,
	photoSenders ...PhotoSender,
) *Handler {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	handler := &Handler{
		botToken: botToken, store: storage, resolver: resolver,
		calculator: calculator, planner: planner, logger: logger, now: time.Now,
	}
	if len(photoSenders) > 0 {
		handler.photoSender = photoSenders[0]
	}
	return handler
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
	mux.HandleFunc("POST /api/miniapp/prayer-card", h.api(h.sendPrayerCard))
	mux.HandleFunc("POST /api/miniapp/calendar-subscription", h.api(h.createCalendarSubscription))
	mux.HandleFunc("DELETE /api/miniapp/calendar-subscription", h.api(h.disableCalendarSubscription))
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

func (h *Handler) sendPrayerCard(w http.ResponseWriter, r *http.Request, identity Identity) error {
	if h.photoSender == nil {
		return fmt.Errorf("prayer card sender is unavailable")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxPrayerCardBytes+(1<<20))
	if err := r.ParseMultipartForm(maxPrayerCardBytes); err != nil {
		return badRequest("invalid_prayer_card")
	}
	file, _, err := r.FormFile("card")
	if err != nil {
		return badRequest("invalid_prayer_card")
	}
	defer file.Close() //nolint:errcheck
	content, err := io.ReadAll(io.LimitReader(file, maxPrayerCardBytes+1))
	if err != nil || len(content) == 0 || len(content) > maxPrayerCardBytes {
		return badRequest("invalid_prayer_card")
	}
	config, err := png.DecodeConfig(bytes.NewReader(content))
	if err != nil || config.Width != 1080 || config.Height != 1350 {
		return badRequest("invalid_prayer_card")
	}
	if _, err := h.photoSender.SendPhoto(r.Context(), &botapi.SendPhotoParams{
		ChatID: identity.UserID,
		Photo: &models.InputFileUpload{
			Filename: "prayer-times.png",
			Data:     bytes.NewReader(content),
		},
	}); err != nil {
		return fmt.Errorf("send prayer card to Telegram chat: %w", err)
	}
	return writeJSON(w, map[string]string{"status": "sent"})
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
	OccasionMajor    *bool `json:"occasion_major"`
	OccasionFasting  *bool `json:"occasion_fasting"`
	OccasionObserved *bool `json:"occasion_observed"`
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
		request.OccasionMajor == nil || request.OccasionFasting == nil || request.OccasionObserved == nil ||
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
	if changed && (desired.Prayer || desired.Fasting || desired.Kahf ||
		desired.OccasionMajor || desired.OccasionFasting || desired.OccasionObserved) {
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
		OccasionMajor: *request.OccasionMajor, OccasionFasting: *request.OccasionFasting,
		OccasionObserved: *request.OccasionObserved,
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
	for _, change := range []struct {
		current bool
		desired bool
		kind    domain.ReminderKind
		name    string
	}{
		{current.OccasionMajor, desired.OccasionMajor, domain.ReminderOccasionMajor, "major occasion"},
		{current.OccasionFasting, desired.OccasionFasting, domain.ReminderOccasionFasting, "occasion fasting"},
		{current.OccasionObserved, desired.OccasionObserved, domain.ReminderOccasionObserved, "commonly observed occasion"},
	} {
		if change.current == change.desired {
			continue
		}
		if err := h.store.SetOccasionRule(ctx, chatID, change.kind, change.desired); err != nil {
			return false, reminderResponse{}, fmt.Errorf("update %s reminders: %w", change.name, err)
		}
	}
	return current != desired, desired, nil
}

type bootstrapResponse struct {
	User          userResponse                 `json:"user"`
	Locale        string                       `json:"locale"`
	NeedsLocation bool                         `json:"needs_location"`
	LocationName  string                       `json:"location_name,omitempty"`
	Profile       *profileResponse             `json:"profile,omitempty"`
	Today         *scheduleResponse            `json:"today,omitempty"`
	Tomorrow      *scheduleResponse            `json:"tomorrow,omitempty"`
	Qibla         *qiblaResponse               `json:"qibla,omitempty"`
	Calendar      calendarSubscriptionResponse `json:"calendar"`
	Occasions     []occasionResponse           `json:"occasions,omitempty"`
	Reminders     reminderResponse             `json:"reminders"`
	Options       optionsResponse              `json:"options"`
	Labels        map[string]string            `json:"labels"`
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
	OccasionMajor    bool `json:"occasion_major"`
	OccasionFasting  bool `json:"occasion_fasting"`
	OccasionObserved bool `json:"occasion_observed"`
}

type occasionSourceResponse struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type occasionResponse struct {
	ID            string                   `json:"id"`
	Emoji         string                   `json:"emoji"`
	Category      string                   `json:"category"`
	CategoryLabel string                   `json:"category_label"`
	Title         string                   `json:"title"`
	Summary       string                   `json:"summary"`
	Action        string                   `json:"action"`
	Gregorian     string                   `json:"gregorian"`
	Hijri         string                   `json:"hijri"`
	Sources       []occasionSourceResponse `json:"sources"`
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
	subscription, err := h.store.CalendarSubscription(ctx, identity.UserID)
	if err == nil {
		response.Calendar.Enabled = subscription.Enabled
	} else if !store.IsNotFound(err) {
		return bootstrapResponse{}, fmt.Errorf("load calendar subscription: %w", err)
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
	upcoming, err := occasions.Between(now.In(today.Date.Location()), 400, profile.HijriAdjustment)
	if err != nil {
		return bootstrapResponse{}, fmt.Errorf("calculate upcoming Islamic occasions: %w", err)
	}
	for _, occurrence := range upcoming {
		copy := locale.Occasion(occurrence.Definition.ID)
		item := occasionResponse{
			ID: occurrence.Definition.ID, Emoji: occurrence.Definition.Emoji,
			Category:      string(occurrence.Definition.Category),
			CategoryLabel: locale.OccasionCategory(string(occurrence.Definition.Category)),
			Title:         copy.Title, Summary: copy.Summary, Action: copy.Action,
			Gregorian: fmt.Sprintf("%d %s %d", occurrence.Date.Day(), locale.Month(int(occurrence.Date.Month())), occurrence.Date.Year()),
			Hijri:     fmt.Sprintf("%d %s %d", occurrence.Hijri.Day, locale.HijriMonth(occurrence.Hijri.Month), occurrence.Hijri.Year),
		}
		for _, source := range occurrence.Definition.Sources {
			item.Sources = append(item.Sources, occasionSourceResponse{Label: source.Label, URL: source.URL})
		}
		response.Occasions = append(response.Occasions, item)
		if len(response.Occasions) == 3 {
			break
		}
	}
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
		case domain.ReminderOccasionMajor:
			state.OccasionMajor = true
		case domain.ReminderOccasionFasting:
			state.OccasionFasting = true
		case domain.ReminderOccasionObserved:
			state.OccasionObserved = true
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
		"occasions_title": locale.OccasionUI("title"), "occasions_help": locale.OccasionUI("help"),
		"occasions_disclaimer": locale.OccasionUI("disclaimer"),
		"occasion_recommended": locale.OccasionUI("recommended"), "occasion_sources": locale.OccasionUI("sources"),
		"occasion_major_reminders":    locale.OccasionUI("major_reminders"),
		"occasion_fasting_reminders":  locale.OccasionUI("fasting_reminders"),
		"occasion_observed_reminders": locale.OccasionUI("observed_reminders"),
		"occasion_schedule":           locale.OccasionUI("schedule"),
		"save":                        copy.Save, "saved": copy.Saved, "loading": copy.Loading,
		"location_help": copy.LocationHelp, "location_error": copy.LocationError,
		"open_in_telegram": copy.OpenInTelegram, "temporary_failure": copy.TemporaryFailure,
		"calculated_locally": copy.CalculatedLocally, "companion": copy.Companion,
		"update_location": copy.UpdateLocation,
		"tools":           copy.Tools, "qibla_title": copy.QiblaTitle, "qibla_help": copy.QiblaHelp,
		"qibla_bearing": copy.QiblaBearing, "qibla_distance": copy.QiblaDistance,
		"compass_start": copy.CompassStart, "compass_active": copy.CompassActive,
		"compass_unavailable": copy.CompassUnavailable,
		"calendar_title":      copy.CalendarTitle, "calendar_help": copy.CalendarHelp,
		"calendar_connect": copy.CalendarConnect, "calendar_copy": copy.CalendarCopy,
		"calendar_disconnect": copy.CalendarDisconnect, "calendar_opening": copy.CalendarOpening,
		"calendar_copied": copy.CalendarCopied, "calendar_disconnected": copy.CalendarDisconnected,
		"calendar_private": copy.CalendarPrivate,
		"offline_updating": copy.OfflineUpdating, "offline_updating_help": copy.OfflineUpdatingHelp,
		"offline_title": copy.OfflineTitle, "offline_help": copy.OfflineHelp,
		"home_title": copy.HomeTitle, "home_help": copy.HomeHelp,
		"home_add": copy.HomeAdd, "home_added": copy.HomeAdded,
		"share_title": copy.ShareTitle, "share_help": copy.ShareHelp,
		"share_action": copy.ShareAction, "share_preparing": copy.SharePreparing,
		"share_sent": copy.ShareSent, "share_failed": copy.ShareFailed,
		"share_card_heading": copy.ShareCardHeading, "share_card_footer": copy.ShareCardFooter,
		"share_message": copy.ShareMessage,
	}
}

type miniAppCopy struct {
	Save, Saved, Loading, LocationHelp, LocationError     string
	OpenInTelegram, TemporaryFailure, CalculatedLocally   string
	Companion, UpdateLocation                             string
	Tools, QiblaTitle, QiblaHelp, QiblaBearing            string
	QiblaDistance, CompassStart, CompassActive            string
	CompassUnavailable, CalendarTitle, CalendarHelp       string
	CalendarConnect, CalendarCopy, CalendarDisconnect     string
	CalendarOpening, CalendarCopied, CalendarDisconnected string
	CalendarPrivate                                       string
	OfflineUpdating, OfflineUpdatingHelp                  string
	OfflineTitle, OfflineHelp                             string
	HomeTitle, HomeHelp, HomeAdd, HomeAdded               string
	ShareTitle, ShareHelp, ShareAction, SharePreparing    string
	ShareSent, ShareFailed                                string
	ShareCardHeading, ShareCardFooter, ShareMessage       string
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
		CalendarTitle:      "Prayer calendar", CalendarHelp: "Connect a rolling 30-day prayer calendar that updates automatically.",
		CalendarConnect: "Connect Google Calendar", CalendarCopy: "Copy private link", CalendarDisconnect: "Disconnect calendar",
		CalendarOpening: "Opening Google Calendar…", CalendarCopied: "Private calendar link copied",
		CalendarDisconnected: "Calendar feed disconnected",
		CalendarPrivate:      "Keep the link private. Google controls when subscribed calendars refresh.",
		OfflineUpdating:      "Updating prayer times…", OfflineUpdatingHelp: "Showing the saved schedule while the latest data loads.",
		OfflineTitle: "Offline schedule", OfflineHelp: "Showing the schedule saved at {time}. Changes will be available after reconnecting.",
		HomeTitle: "Quick access", HomeHelp: "Add prayer times to your Telegram home screen.",
		HomeAdd: "Add to home screen", HomeAdded: "Added to home screen",
		ShareTitle: "Share prayer card", ShareHelp: "Create a beautiful image with the selected day’s prayer times.",
		ShareAction: "Create & share", SharePreparing: "Creating card…",
		ShareSent: "Card sent to this bot chat. Open the chat to save or forward it.", ShareFailed: "The prayer card could not be created.",
		ShareCardHeading: "Daily prayer times", ShareCardFooter: "Global Prayer Times",
		ShareMessage: "Prayer times",
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
		CalendarTitle:      "تقويم الصلاة", CalendarHelp: "اربط تقويمًا متجددًا لمواقيت الصلاة خلال 30 يومًا.",
		CalendarConnect: "الربط مع تقويم Google", CalendarCopy: "نسخ الرابط الخاص", CalendarDisconnect: "قطع اتصال التقويم",
		CalendarOpening: "جارٍ فتح تقويم Google…", CalendarCopied: "تم نسخ رابط التقويم الخاص",
		CalendarDisconnected: "تم قطع اتصال التقويم",
		CalendarPrivate:      "احتفظ بالرابط سريًا. يحدد Google وقت تحديث التقويمات المشتركة.",
		OfflineUpdating:      "جارٍ تحديث المواقيت…", OfflineUpdatingHelp: "نعرض الجدول المحفوظ أثناء تحميل أحدث البيانات.",
		OfflineTitle: "الجدول دون اتصال", OfflineHelp: "نعرض الجدول المحفوظ الساعة {time}. ستتوفر التغييرات بعد عودة الاتصال.",
		HomeTitle: "وصول سريع", HomeHelp: "أضف مواقيت الصلاة إلى شاشة تيليجرام الرئيسية.",
		HomeAdd: "إضافة إلى الشاشة الرئيسية", HomeAdded: "تمت الإضافة إلى الشاشة الرئيسية",
		ShareTitle: "مشاركة بطاقة الصلاة", ShareHelp: "أنشئ صورة جميلة بمواقيت اليوم المحدد.",
		ShareAction: "إنشاء ومشاركة", SharePreparing: "جارٍ إنشاء البطاقة…",
		ShareSent: "تم إرسال البطاقة إلى محادثة البوت. افتح المحادثة لحفظها أو إعادة توجيهها.", ShareFailed: "تعذر إنشاء بطاقة الصلاة.",
		ShareCardHeading: "مواقيت الصلاة اليومية", ShareCardFooter: "مواقيت الصلاة العالمية",
		ShareMessage: "مواقيت الصلاة",
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
		CalendarTitle:      "Calendario de oración", CalendarHelp: "Conecta un calendario móvil de 30 días que se actualiza automáticamente.",
		CalendarConnect: "Conectar Google Calendar", CalendarCopy: "Copiar enlace privado", CalendarDisconnect: "Desconectar calendario",
		CalendarOpening: "Abriendo Google Calendar…", CalendarCopied: "Enlace privado copiado",
		CalendarDisconnected: "Calendario desconectado",
		CalendarPrivate:      "Mantén el enlace privado. Google decide cuándo actualizar los calendarios suscritos.",
		OfflineUpdating:      "Actualizando horarios…", OfflineUpdatingHelp: "Mostramos el horario guardado mientras se cargan los datos nuevos.",
		OfflineTitle: "Horario sin conexión", OfflineHelp: "Mostrando el horario guardado a las {time}. Los cambios estarán disponibles al reconectar.",
		HomeTitle: "Acceso rápido", HomeHelp: "Añade los horarios a la pantalla de inicio de Telegram.",
		HomeAdd: "Añadir a inicio", HomeAdded: "Añadido a inicio",
		ShareTitle: "Compartir tarjeta", ShareHelp: "Crea una imagen con los horarios del día seleccionado.",
		ShareAction: "Crear y compartir", SharePreparing: "Creando tarjeta…",
		ShareSent: "La tarjeta se envió al chat del bot. Abre el chat para guardarla o reenviarla.", ShareFailed: "No se pudo crear la tarjeta.",
		ShareCardHeading: "Horarios diarios de oración", ShareCardFooter: "Horarios de oración globales",
		ShareMessage: "Horarios de oración",
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
		CalendarTitle:      "Calendrier des prières", CalendarHelp: "Connectez un calendrier glissant de 30 jours mis à jour automatiquement.",
		CalendarConnect: "Connecter Google Agenda", CalendarCopy: "Copier le lien privé", CalendarDisconnect: "Déconnecter le calendrier",
		CalendarOpening: "Ouverture de Google Agenda…", CalendarCopied: "Lien privé copié",
		CalendarDisconnected: "Calendrier déconnecté",
		CalendarPrivate:      "Gardez ce lien privé. Google décide quand actualiser les calendriers abonnés.",
		OfflineUpdating:      "Actualisation des horaires…", OfflineUpdatingHelp: "L’horaire enregistré reste affiché pendant le chargement.",
		OfflineTitle: "Horaire hors ligne", OfflineHelp: "Horaire enregistré à {time}. Les modifications seront disponibles après reconnexion.",
		HomeTitle: "Accès rapide", HomeHelp: "Ajoutez les horaires à l’écran d’accueil Telegram.",
		HomeAdd: "Ajouter à l’accueil", HomeAdded: "Ajouté à l’accueil",
		ShareTitle: "Partager une carte", ShareHelp: "Créez une image avec les horaires du jour sélectionné.",
		ShareAction: "Créer et partager", SharePreparing: "Création de la carte…",
		ShareSent: "La carte a été envoyée dans le chat du bot. Ouvrez le chat pour l’enregistrer ou la transférer.", ShareFailed: "Impossible de créer la carte.",
		ShareCardHeading: "Horaires de prière du jour", ShareCardFooter: "Horaires de prière mondiaux",
		ShareMessage: "Horaires de prière",
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
		CalendarTitle:      "Календарь намаза", CalendarHelp: "Подключите автоматически обновляемый календарь намаза на 30 дней.",
		CalendarConnect: "Подключить Google Календарь", CalendarCopy: "Копировать закрытую ссылку", CalendarDisconnect: "Отключить календарь",
		CalendarOpening: "Открываем Google Календарь…", CalendarCopied: "Закрытая ссылка скопирована",
		CalendarDisconnected: "Календарь отключён",
		CalendarPrivate:      "Не передавайте ссылку другим. Частоту обновления подписки определяет Google.",
		OfflineUpdating:      "Обновляем время намаза…", OfflineUpdatingHelp: "Пока новые данные загружаются, показываем сохранённое расписание.",
		OfflineTitle: "Расписание офлайн", OfflineHelp: "Показано расписание, сохранённое в {time}. Изменения станут доступны после подключения.",
		HomeTitle: "Быстрый доступ", HomeHelp: "Добавьте время намаза на главный экран Telegram.",
		HomeAdd: "Добавить на главный экран", HomeAdded: "Добавлено на главный экран",
		ShareTitle: "Поделиться карточкой", ShareHelp: "Создайте красивую карточку с расписанием выбранного дня.",
		ShareAction: "Создать и поделиться", SharePreparing: "Создаём карточку…",
		ShareSent: "Карточка отправлена в чат с ботом. Откройте чат, чтобы сохранить или переслать её.", ShareFailed: "Не удалось создать карточку.",
		ShareCardHeading: "Время намаза на день", ShareCardFooter: "Global Prayer Times",
		ShareMessage: "Время намаза",
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
		CalendarTitle:      "Namaz takvimi", CalendarHelp: "Otomatik güncellenen 30 günlük namaz takvimini bağlayın.",
		CalendarConnect: "Google Takvim’e bağlan", CalendarCopy: "Özel bağlantıyı kopyala", CalendarDisconnect: "Takvim bağlantısını kes",
		CalendarOpening: "Google Takvim açılıyor…", CalendarCopied: "Özel takvim bağlantısı kopyalandı",
		CalendarDisconnected: "Takvim bağlantısı kesildi",
		CalendarPrivate:      "Bağlantıyı gizli tutun. Abone takvimlerin yenilenme zamanını Google belirler.",
		OfflineUpdating:      "Namaz vakitleri güncelleniyor…", OfflineUpdatingHelp: "Yeni veriler yüklenirken kayıtlı takvim gösteriliyor.",
		OfflineTitle: "Çevrimdışı takvim", OfflineHelp: "{time} saatinde kaydedilen takvim gösteriliyor. Değişiklikler bağlantı gelince açılır.",
		HomeTitle: "Hızlı erişim", HomeHelp: "Namaz vakitlerini Telegram ana ekranına ekleyin.",
		HomeAdd: "Ana ekrana ekle", HomeAdded: "Ana ekrana eklendi",
		ShareTitle: "Namaz kartını paylaş", ShareHelp: "Seçilen günün vakitleriyle güzel bir görsel oluşturun.",
		ShareAction: "Oluştur ve paylaş", SharePreparing: "Kart oluşturuluyor…",
		ShareSent: "Kart bot sohbetine gönderildi. Kaydetmek veya iletmek için sohbeti açın.", ShareFailed: "Namaz kartı oluşturulamadı.",
		ShareCardHeading: "Günlük namaz vakitleri", ShareCardFooter: "Global Namaz Vakitleri",
		ShareMessage: "Namaz vakitleri",
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
		CalendarTitle:      "Namoz taqvimi", CalendarHelp: "Avtomatik yangilanadigan 30 kunlik namoz taqvimini ulang.",
		CalendarConnect: "Google Taqvimga ulash", CalendarCopy: "Maxfiy havolani nusxalash", CalendarDisconnect: "Taqvimni uzish",
		CalendarOpening: "Google Taqvim ochilmoqda…", CalendarCopied: "Maxfiy taqvim havolasi nusxalandi",
		CalendarDisconnected: "Taqvim uzildi",
		CalendarPrivate:      "Havolani maxfiy saqlang. Obuna taqvimini qachon yangilashni Google belgilaydi.",
		OfflineUpdating:      "Namoz vaqtlari yangilanmoqda…", OfflineUpdatingHelp: "Yangi ma’lumot yuklanayotganda saqlangan jadval ko‘rsatiladi.",
		OfflineTitle: "Oflayn jadval", OfflineHelp: "{time} da saqlangan jadval ko‘rsatilmoqda. O‘zgarishlar ulanish qaytgach ishlaydi.",
		HomeTitle: "Tezkor kirish", HomeHelp: "Namoz vaqtlarini Telegram bosh ekraniga qo‘shing.",
		HomeAdd: "Bosh ekranga qo‘shish", HomeAdded: "Bosh ekranga qo‘shildi",
		ShareTitle: "Namoz kartasini ulashish", ShareHelp: "Tanlangan kun vaqtlari bilan chiroyli rasm yarating.",
		ShareAction: "Yaratish va ulashish", SharePreparing: "Karta yaratilmoqda…",
		ShareSent: "Karta bot chatiga yuborildi. Saqlash yoki ulashish uchun chatni oching.", ShareFailed: "Namoz kartasini yaratib bo‘lmadi.",
		ShareCardHeading: "Kunlik namoz vaqtlari", ShareCardFooter: "Global Namoz Vaqtlari",
		ShareMessage: "Namoz vaqtlari",
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
		CalendarTitle:      "Намаз календаре", CalendarHelp: "Автоматик яңартыла торган 30 көнлек намаз календарен тоташтырыгыз.",
		CalendarConnect: "Google Календарьга тоташтыру", CalendarCopy: "Яшерен сылтаманы күчерү", CalendarDisconnect: "Календарьны өзү",
		CalendarOpening: "Google Календарь ачыла…", CalendarCopied: "Яшерен календарь сылтамасы күчерелде",
		CalendarDisconnected: "Календарь өзелде",
		CalendarPrivate:      "Сылтаманы яшерен саклагыз. Яңарту вакытын Google билгели.",
		OfflineUpdating:      "Намаз вакытлары яңартыла…", OfflineUpdatingHelp: "Яңа мәгълүмат йөкләнгәндә сакланган җәдвәл күрсәтелә.",
		OfflineTitle: "Офлайн җәдвәл", OfflineHelp: "{time} сәгатьтә сакланган җәдвәл күрсәтелә. Үзгәрешләр элемтә кайткач эшләячәк.",
		HomeTitle: "Тиз керү", HomeHelp: "Намаз вакытларын Telegram төп экранына өстәгез.",
		HomeAdd: "Төп экранга өстәү", HomeAdded: "Төп экранга өстәлде",
		ShareTitle: "Намаз карточкасын бүлешү", ShareHelp: "Сайланган көн вакытлары белән матур рәсем ясагыз.",
		ShareAction: "Ясау һәм бүлешү", SharePreparing: "Карточка ясала…",
		ShareSent: "Карточка бот чатына җибәрелде. Саклау яки җибәрү өчен чатны ачыгыз.", ShareFailed: "Намаз карточкасын ясап булмады.",
		ShareCardHeading: "Көнлек намаз вакытлары", ShareCardFooter: "Глобаль намаз вакытлары",
		ShareMessage: "Намаз вакытлары",
	},
}
