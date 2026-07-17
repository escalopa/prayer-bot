package miniapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
	"github.com/escalopa/prayer-bot/global/internal/location"
	"github.com/escalopa/prayer-bot/global/internal/prayertime"
	"github.com/jackc/pgx/v5"
)

func TestStaticMiniAppIsEmbeddedWithSecurityHeaders(t *testing.T) {
	handler := NewHandler("token", nil, nil, nil, nil, nil)
	mux := http.NewServeMux()
	handler.Register(mux)

	request := httptest.NewRequest(http.MethodGet, "/app/", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	html := response.Body.String()
	if !strings.Contains(html, "telegram-web-app.js?63") {
		t.Fatal("embedded Mini App HTML is missing Telegram SDK")
	}
	if strings.Contains(html, "section-kicker") {
		t.Fatal("Mini App HTML contains a duplicate section icon")
	}
	script, err := embeddedStatic.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(script), "tgWebAppData") || !strings.Contains(string(script), "/api/miniapp/preferences") {
		t.Fatal("Mini App script is missing resilient Telegram launch data or unified preference saving")
	}
	if !strings.Contains(html, "qibla-compass") || !strings.Contains(html, "export-month") ||
		!strings.Contains(string(script), "DeviceOrientation") || !strings.Contains(string(script), "downloadFile") {
		t.Fatal("Mini App is missing Qibla compass or native calendar export support")
	}
	if !strings.Contains(response.Header().Get("Content-Security-Policy"), "telegram.org") {
		t.Fatal("Mini App response is missing its CSP")
	}
}

func TestMiniAppAPIRejectsUnsignedRequests(t *testing.T) {
	handler := NewHandler("token", nil, nil, nil, nil, nil)
	mux := http.NewServeMux()
	handler.Register(mux)

	request := httptest.NewRequest(http.MethodPost, "/api/miniapp/bootstrap", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), "unauthorized") {
		t.Fatalf("unexpected response: %d %s", response.Code, response.Body.String())
	}
}

func TestFormatScheduleIncludesLocalizedGregorianAndHijriDates(t *testing.T) {
	location := time.FixedZone("test", 2*60*60)
	date := time.Date(2026, time.July, 17, 12, 0, 0, 0, location)
	schedule := domain.DaySchedule{
		Date: date, Timezone: "Africa/Cairo",
		Times: map[domain.Prayer]time.Time{
			domain.PrayerFajr: time.Date(2026, time.July, 17, 4, 12, 0, 0, location),
		},
	}
	profile := domain.PrayerProfile{Timezone: "Africa/Cairo", Method: domain.MethodEgyptian}

	result := formatSchedule(schedule, profile, i18n.Resolve("ar"))
	if !strings.Contains(result.Gregorian, "يوليو") || !strings.Contains(result.Hijri, "هـ") {
		t.Fatalf("unexpected localized dates: %+v", result)
	}
	if len(result.Prayers) != 1 || result.Prayers[0].Time != "04:12" || result.Prayers[0].Name != "الفجر" {
		t.Fatalf("unexpected prayers: %+v", result.Prayers)
	}
}

func TestBootstrapUsesSignedTelegramIdentityAndReturnsLocationGate(t *testing.T) {
	now := time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC)
	storage := newFakeStorage()
	handler := NewHandler("test-token", storage, nil, nil, nil, nil)
	handler.now = func() time.Time { return now }
	mux := http.NewServeMux()
	handler.Register(mux)

	request := httptest.NewRequest(http.MethodPost, "/api/miniapp/bootstrap", nil)
	request.Header.Set("X-Telegram-Init-Data", signedInitData(t, "test-token", now, initDataUser{
		ID: 42, FirstName: "Amina", LanguageCode: "ar",
	}))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var data bootstrapResponse
	if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if data.User.ID != 42 || data.User.FirstName != "Amina" || data.Locale != "ar" || !data.NeedsLocation {
		t.Fatalf("unexpected bootstrap response: %+v", data)
	}
	if chat := storage.chats[42]; chat.TelegramChatID != 42 || chat.LanguageCode != "ar" || chat.Type != "private" {
		t.Fatalf("unexpected stored chat: %+v", chat)
	}
}

func TestLocationUpdatePreservesCalculationSettingsAndRoundsCoordinates(t *testing.T) {
	now := time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC)
	storage := newFakeStorage()
	storage.chats[42] = domain.Chat{TelegramChatID: 42, Type: "private", LanguageCode: "en"}
	storage.profiles[42] = domain.PrayerProfile{
		ChatID: 42, Latitude: 1, Longitude: 2, Timezone: "UTC", PlaceID: "old",
		Method: domain.MethodISNA, Madhab: domain.MadhabHanafi,
		HighLatitudeRule: domain.HighLatitudeMiddleNight, HijriAdjustment: 1,
		Adjustments: domain.Adjustments{Fajr: 2},
	}
	resolver := &fakeResolver{resolved: location.Resolved{
		Timezone: "Africa/Cairo", PlaceID: "cairo", City: "Cairo", CountryCode: "EG",
	}}
	planner := &fakePlanner{}
	handler := NewHandler("test-token", storage, resolver, prayertime.New(), planner, nil)
	handler.now = func() time.Time { return now }
	mux := http.NewServeMux()
	handler.Register(mux)

	request := httptest.NewRequest(http.MethodPut, "/api/miniapp/location", strings.NewReader(`{"latitude":30.04442,"longitude":31.23571}`))
	request.Header.Set("X-Telegram-Init-Data", signedInitData(t, "test-token", now, initDataUser{ID: 42, FirstName: "Amina", LanguageCode: "en"}))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	profile := storage.profiles[42]
	if profile.Latitude != 30.044 || profile.Longitude != 31.236 || profile.Timezone != "Africa/Cairo" {
		t.Fatalf("unexpected saved location: %+v", profile)
	}
	if profile.Method != domain.MethodISNA || profile.Madhab != domain.MadhabHanafi || profile.HijriAdjustment != 1 || profile.Adjustments.Fajr != 2 {
		t.Fatalf("existing calculation settings were not preserved: %+v", profile)
	}
	if resolver.latitude != 30.04442 || resolver.longitude != 31.23571 || planner.rebuilds != 1 {
		t.Fatalf("resolver/planner calls = %+v / %d", resolver, planner.rebuilds)
	}
	var data bootstrapResponse
	if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if data.LocationName != "Cairo" || data.Today == nil || data.Tomorrow == nil || data.Qibla == nil {
		t.Fatalf("unexpected response: %+v", data)
	}
	if data.Qibla.BearingDegrees < 135 || data.Qibla.BearingDegrees > 137 || data.Qibla.DistanceKilometres < 1250 {
		t.Fatalf("unexpected Qibla result: %+v", data.Qibla)
	}
}

func TestCalendarExportUsesShortLivedSignedDownload(t *testing.T) {
	now := time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC)
	storage := newFakeStorage()
	storage.chats[42] = domain.Chat{TelegramChatID: 42, Type: "private", LanguageCode: "en"}
	storage.profiles[42] = domain.PrayerProfile{
		ChatID: 42, Latitude: 30.044, Longitude: 31.236, Timezone: "Africa/Cairo",
		Method: domain.MethodEgyptian, Madhab: domain.MadhabShafii,
		HighLatitudeRule: domain.HighLatitudeAngleBased,
	}
	handler := NewHandler("test-token", storage, nil, prayertime.New(), nil, nil)
	handler.now = func() time.Time { return now }
	mux := http.NewServeMux()
	handler.Register(mux)

	linkRequest := httptest.NewRequest(http.MethodPost, "/api/miniapp/calendar-link", strings.NewReader(`{"days":7}`))
	linkRequest.Header.Set("X-Telegram-Init-Data", signedInitData(t, "test-token", now, initDataUser{
		ID: 42, FirstName: "Amina", LanguageCode: "en",
	}))
	linkResponse := httptest.NewRecorder()
	mux.ServeHTTP(linkResponse, linkRequest)
	if linkResponse.Code != http.StatusOK {
		t.Fatalf("link status = %d, body = %s", linkResponse.Code, linkResponse.Body.String())
	}
	var link calendarLinkResponse
	if err := json.Unmarshal(linkResponse.Body.Bytes(), &link); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(link.Path, "/api/miniapp/calendar.ics?token=") ||
		link.Filename != "prayer-times-2026-07-17-7-days.ics" {
		t.Fatalf("unexpected calendar link: %+v", link)
	}

	downloadRequest := httptest.NewRequest(http.MethodGet, link.Path, nil)
	downloadResponse := httptest.NewRecorder()
	mux.ServeHTTP(downloadResponse, downloadRequest)
	if downloadResponse.Code != http.StatusOK {
		t.Fatalf("download status = %d, body = %s", downloadResponse.Code, downloadResponse.Body.String())
	}
	if contentType := downloadResponse.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/calendar") {
		t.Fatalf("content type = %q", contentType)
	}
	if disposition := downloadResponse.Header().Get("Content-Disposition"); !strings.Contains(disposition, link.Filename) {
		t.Fatalf("content disposition = %q", disposition)
	}
	if downloadResponse.Header().Get("Access-Control-Allow-Origin") != "https://web.telegram.org" {
		t.Fatal("calendar response is missing Telegram Web download CORS header")
	}
	if events := strings.Count(downloadResponse.Body.String(), "BEGIN:VEVENT\r\n"); events != 42 {
		t.Fatalf("calendar event count = %d, want 42", events)
	}
}

func TestCalendarTokensRejectTamperingAndExpiry(t *testing.T) {
	now := time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC)
	token, err := signCalendarToken("secret", calendarTokenPayload{
		ChatID: 42, Days: 30, ExpiresAt: now.Add(calendarTokenLifetime).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if payload, err := verifyCalendarToken("secret", token, now); err != nil || payload.ChatID != 42 || payload.Days != 30 {
		t.Fatalf("valid token failed: payload=%+v err=%v", payload, err)
	}
	if _, err := verifyCalendarToken("secret", token+"x", now); err == nil {
		t.Fatal("tampered token was accepted")
	}
	if _, err := verifyCalendarToken("secret", token, now.Add(calendarTokenLifetime)); err == nil {
		t.Fatal("expired token was accepted")
	}
}

func TestPreferencesUpdateSavesSettingsAndRemindersTogether(t *testing.T) {
	now := time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC)
	storage := newFakeStorage()
	storage.chats[42] = domain.Chat{TelegramChatID: 42, Type: "private", LanguageCode: "en"}
	storage.profiles[42] = domain.PrayerProfile{
		ChatID: 42, Latitude: 30.044, Longitude: 31.236, Timezone: "UTC",
		Method: domain.MethodEgyptian, Madhab: domain.MadhabShafii,
		HighLatitudeRule: domain.HighLatitudeAngleBased,
	}
	planner := &fakePlanner{}
	handler := NewHandler("test-token", storage, nil, prayertime.New(), planner, nil)
	handler.now = func() time.Time { return now }
	mux := http.NewServeMux()
	handler.Register(mux)

	body := `{
		"settings":{"language":"ar","method":"isna","madhab":"hanafi","high_latitude_rule":"middle_of_night","hijri_adjustment":1,
		"adjustments":{"fajr":2,"sunrise":0,"dhuhr":0,"asr":1,"maghrib":0,"isha":-1}},
		"reminders":{"prayer":true,"pre_prayer_minutes":20,"fasting":true,"kahf":false}
	}`
	request := httptest.NewRequest(http.MethodPut, "/api/miniapp/preferences", strings.NewReader(body))
	request.Header.Set("X-Telegram-Init-Data", signedInitData(t, "test-token", now, initDataUser{ID: 42, FirstName: "Amina", LanguageCode: "en"}))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	profile := storage.profiles[42]
	if storage.chats[42].LanguageCode != "ar" || profile.Method != domain.MethodISNA || profile.Madhab != domain.MadhabHanafi ||
		profile.HighLatitudeRule != domain.HighLatitudeMiddleNight || profile.HijriAdjustment != 1 || profile.Adjustments.Fajr != 2 {
		t.Fatalf("preferences were not saved together: chat=%+v profile=%+v", storage.chats[42], profile)
	}
	var data bootstrapResponse
	if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if data.Locale != "ar" || !data.Reminders.Prayer || data.Reminders.PrePrayerMinutes != 20 ||
		!data.Reminders.Fasting || data.Reminders.Kahf || planner.rebuilds != 1 {
		t.Fatalf("unexpected updated state: response=%+v rebuilds=%d", data.Reminders, planner.rebuilds)
	}
}

func TestParseAdjustmentsRequiresCompleteSnapshot(t *testing.T) {
	if _, err := parseAdjustments(map[string]int{"fajr": 1}); err == nil {
		t.Fatal("expected an incomplete adjustment snapshot to fail")
	}
	values := map[string]int{"fajr": 1, "sunrise": 0, "dhuhr": 0, "asr": 0, "maghrib": 0, "isha": -1}
	adjustments, err := parseAdjustments(values)
	if err != nil || adjustments.Fajr != 1 || adjustments.Isha != -1 {
		t.Fatalf("unexpected adjustments: %+v, %v", adjustments, err)
	}
}

func TestValidateRemindersRejectsUnsupportedLeadTime(t *testing.T) {
	enabled, disabled := true, false
	if err := validateReminders(remindersRequest{
		Prayer: &enabled, PrePrayerMinutes: 20, Fasting: &disabled, Kahf: &disabled,
	}); err != nil {
		t.Fatalf("20-minute pre-reminder was rejected: %v", err)
	}
	if err := validateReminders(remindersRequest{
		Prayer: &enabled, PrePrayerMinutes: 17, Fasting: &disabled, Kahf: &disabled,
	}); err == nil {
		t.Fatal("unsupported pre-reminder lead time was accepted")
	}
}

type fakeStorage struct {
	chats    map[int64]domain.Chat
	profiles map[int64]domain.PrayerProfile
	rules    map[int64][]domain.ReminderRule
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{
		chats: make(map[int64]domain.Chat), profiles: make(map[int64]domain.PrayerProfile),
		rules: make(map[int64][]domain.ReminderRule),
	}
}

func (s *fakeStorage) UpsertChat(_ context.Context, chat domain.Chat) error {
	if current, ok := s.chats[chat.TelegramChatID]; ok {
		chat.LanguageCode = current.LanguageCode
	}
	s.chats[chat.TelegramChatID] = chat
	return nil
}

func (s *fakeStorage) Chat(_ context.Context, chatID int64) (domain.Chat, error) {
	chat, ok := s.chats[chatID]
	if !ok {
		return domain.Chat{}, pgx.ErrNoRows
	}
	return chat, nil
}

func (s *fakeStorage) SetLanguage(_ context.Context, chatID int64, language string) error {
	chat := s.chats[chatID]
	chat.LanguageCode = language
	s.chats[chatID] = chat
	return nil
}

func (s *fakeStorage) Profile(_ context.Context, chatID int64) (domain.PrayerProfile, error) {
	profile, ok := s.profiles[chatID]
	if !ok {
		return domain.PrayerProfile{}, pgx.ErrNoRows
	}
	return profile, nil
}

func (s *fakeStorage) UpsertProfile(_ context.Context, profile domain.PrayerProfile) (domain.PrayerProfile, error) {
	s.profiles[profile.ChatID] = profile
	return profile, nil
}

func (s *fakeStorage) EnabledRules(_ context.Context, chatID int64) ([]domain.ReminderRule, error) {
	var enabled []domain.ReminderRule
	for _, rule := range s.rules[chatID] {
		if rule.Enabled {
			enabled = append(enabled, rule)
		}
	}
	return enabled, nil
}

func (s *fakeStorage) EnableDefaultRules(_ context.Context, chatID int64) error {
	s.rules[chatID] = append(s.rules[chatID], domain.ReminderRule{ChatID: chatID, Kind: domain.ReminderAt, Enabled: true})
	return nil
}

func (s *fakeStorage) DisableRules(_ context.Context, chatID int64) error {
	for index := range s.rules[chatID] {
		if !s.rules[chatID][index].Kind.Weekly() {
			s.rules[chatID][index].Enabled = false
		}
	}
	return nil
}

func (s *fakeStorage) ConfigurePrayerRules(_ context.Context, chatID int64, enabled bool, beforeMinutes int) error {
	for index := range s.rules[chatID] {
		if !s.rules[chatID][index].Kind.Weekly() {
			s.rules[chatID][index].Enabled = false
		}
	}
	if enabled {
		s.rules[chatID] = append(s.rules[chatID], domain.ReminderRule{
			ChatID: chatID, Kind: domain.ReminderAt, Enabled: true,
		})
		if beforeMinutes > 0 {
			s.rules[chatID] = append(s.rules[chatID], domain.ReminderRule{
				ChatID: chatID, Kind: domain.ReminderBefore, OffsetMinutes: beforeMinutes, Enabled: true,
			})
		}
	}
	return nil
}

func (s *fakeStorage) SetWeeklyRule(_ context.Context, chatID int64, kind domain.ReminderKind, enabled bool) error {
	for index := range s.rules[chatID] {
		if s.rules[chatID][index].Kind == kind {
			s.rules[chatID][index].Enabled = enabled
			return nil
		}
	}
	s.rules[chatID] = append(s.rules[chatID], domain.ReminderRule{ChatID: chatID, Kind: kind, Enabled: enabled})
	return nil
}

type fakeResolver struct {
	resolved            location.Resolved
	latitude, longitude float64
}

func (r *fakeResolver) Resolve(_ context.Context, latitude, longitude float64) (location.Resolved, error) {
	r.latitude, r.longitude = latitude, longitude
	return r.resolved, nil
}

type fakePlanner struct{ rebuilds int }

func (p *fakePlanner) RebuildChat(context.Context, int64, time.Time) error {
	p.rebuilds++
	return nil
}
