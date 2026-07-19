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
	if !strings.Contains(html, "qibla-compass") || !strings.Contains(html, "connect-calendar") ||
		!strings.Contains(string(script), "DeviceOrientation") ||
		!strings.Contains(string(script), "/api/miniapp/calendar-subscription") {
		t.Fatal("Mini App is missing Qibla compass or rolling calendar subscription support")
	}
	if !strings.Contains(html, "add-home-screen") || !strings.Contains(html, "share-prayer-card") ||
		!strings.Contains(string(script), "DeviceStorage") ||
		!strings.Contains(string(script), "addToHomeScreen") ||
		!strings.Contains(string(script), "navigator.share") ||
		!strings.Contains(string(script), "canvas.toDataURL") {
		t.Fatal("Mini App is missing offline, home-screen, or prayer-card sharing support")
	}
	serviceWorker, err := embeddedStatic.ReadFile("static/sw.js")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(script), "serviceWorker.register") ||
		!strings.Contains(string(serviceWorker), "global-prayer-miniapp-shell") ||
		!strings.Contains(string(script), "delete snapshot.calendar.path") {
		t.Fatal("Mini App is missing its safe offline application shell")
	}
	if !strings.Contains(response.Header().Get("Content-Security-Policy"), "telegram.org") {
		t.Fatal("Mini App response is missing its CSP")
	}
}

func TestOfflineAndSharingLabelsExistForEveryLocale(t *testing.T) {
	keys := []string{
		"offline_updating", "offline_updating_help", "offline_title", "offline_help",
		"home_title", "home_help", "home_add", "home_added",
		"share_title", "share_help", "share_action", "share_preparing",
		"share_downloaded", "share_failed", "share_card_heading",
		"share_card_footer", "share_message",
	}
	for _, locale := range i18n.Supported() {
		localized := labels(locale)
		for _, key := range keys {
			if localized[key] == "" {
				t.Errorf("locale %q has no %q label", locale.Code, key)
			}
		}
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

func TestCalendarSubscriptionProducesRollingThirtyDayFeed(t *testing.T) {
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

	linkRequest := httptest.NewRequest(http.MethodPost, "/api/miniapp/calendar-subscription", nil)
	linkRequest.Header.Set("X-Telegram-Init-Data", signedInitData(t, "test-token", now, initDataUser{
		ID: 42, FirstName: "Amina", LanguageCode: "en",
	}))
	linkResponse := httptest.NewRecorder()
	mux.ServeHTTP(linkResponse, linkRequest)
	if linkResponse.Code != http.StatusOK {
		t.Fatalf("link status = %d, body = %s", linkResponse.Code, linkResponse.Body.String())
	}
	var link calendarSubscriptionResponse
	if err := json.Unmarshal(linkResponse.Body.Bytes(), &link); err != nil {
		t.Fatal(err)
	}
	if !link.Enabled || !strings.HasPrefix(link.Path, "/api/miniapp/calendar.ics?token=") {
		t.Fatalf("unexpected calendar link: %+v", link)
	}
	subscription := storage.subscriptions[42]
	if !subscription.Enabled || len(subscription.FeedToken) != 64 || len(subscription.UIDNamespace) != 32 {
		t.Fatalf("unexpected stored subscription: %+v", subscription)
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
	if disposition := downloadResponse.Header().Get("Content-Disposition"); disposition != `inline; filename="prayer-times.ics"` {
		t.Fatalf("content disposition = %q", disposition)
	}
	if !strings.Contains(downloadResponse.Body.String(), "REFRESH-INTERVAL;VALUE=DURATION:PT12H") {
		t.Fatal("calendar response is missing its refresh hint")
	}
	if events := strings.Count(downloadResponse.Body.String(), "BEGIN:VEVENT\r\n"); events != 180 {
		t.Fatalf("calendar event count = %d, want 180", events)
	}
	if !strings.Contains(downloadResponse.Body.String(), subscription.UIDNamespace+"-20260717-fajr@global-prayer-bot") {
		t.Fatal("calendar response is missing its stable event UID")
	}
	etag := downloadResponse.Header().Get("ETag")
	if etag == "" {
		t.Fatal("calendar response is missing its ETag")
	}
	cachedRequest := httptest.NewRequest(http.MethodGet, link.Path, nil)
	cachedRequest.Header.Set("If-None-Match", etag)
	cachedResponse := httptest.NewRecorder()
	mux.ServeHTTP(cachedResponse, cachedRequest)
	if cachedResponse.Code != http.StatusNotModified {
		t.Fatalf("cached status = %d, want 304", cachedResponse.Code)
	}
	tamperedResponse := httptest.NewRecorder()
	mux.ServeHTTP(tamperedResponse, httptest.NewRequest(http.MethodGet, link.Path+"0", nil))
	if tamperedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("tampered calendar status = %d, want 401", tamperedResponse.Code)
	}

	now = now.AddDate(0, 0, 1)
	rolledRequest := httptest.NewRequest(http.MethodGet, link.Path, nil)
	rolledResponse := httptest.NewRecorder()
	mux.ServeHTTP(rolledResponse, rolledRequest)
	if rolledResponse.Code != http.StatusOK {
		t.Fatalf("rolled feed status = %d, body = %s", rolledResponse.Code, rolledResponse.Body.String())
	}
	if strings.Contains(rolledResponse.Body.String(), subscription.UIDNamespace+"-20260717-fajr@global-prayer-bot") ||
		!strings.Contains(rolledResponse.Body.String(), subscription.UIDNamespace+"-20260718-fajr@global-prayer-bot") {
		t.Fatal("calendar feed did not roll forward by one local day")
	}
}

func TestCalendarSubscriptionCanBeRevoked(t *testing.T) {
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
	initData := signedInitData(t, "test-token", now, initDataUser{
		ID: 42, FirstName: "Amina", LanguageCode: "en",
	})

	createRequest := httptest.NewRequest(http.MethodPost, "/api/miniapp/calendar-subscription", nil)
	createRequest.Header.Set("X-Telegram-Init-Data", initData)
	createResponse := httptest.NewRecorder()
	mux.ServeHTTP(createResponse, createRequest)
	var created calendarSubscriptionResponse
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/miniapp/calendar-subscription", nil)
	deleteRequest.Header.Set("X-Telegram-Init-Data", initData)
	deleteResponse := httptest.NewRecorder()
	mux.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusOK || storage.subscriptions[42].Enabled {
		t.Fatalf("calendar subscription was not disabled: %d %+v", deleteResponse.Code, storage.subscriptions[42])
	}

	downloadResponse := httptest.NewRecorder()
	mux.ServeHTTP(downloadResponse, httptest.NewRequest(http.MethodGet, created.Path, nil))
	if downloadResponse.Code != http.StatusUnauthorized {
		t.Fatalf("revoked calendar status = %d, want 401", downloadResponse.Code)
	}

	previous := storage.subscriptions[42]
	reconnectRequest := httptest.NewRequest(http.MethodPost, "/api/miniapp/calendar-subscription", nil)
	reconnectRequest.Header.Set("X-Telegram-Init-Data", initData)
	reconnectResponse := httptest.NewRecorder()
	mux.ServeHTTP(reconnectResponse, reconnectRequest)
	var reconnected calendarSubscriptionResponse
	if err := json.Unmarshal(reconnectResponse.Body.Bytes(), &reconnected); err != nil {
		t.Fatal(err)
	}
	current := storage.subscriptions[42]
	if reconnected.Path == created.Path || current.FeedToken == previous.FeedToken {
		t.Fatal("reconnecting did not replace the revoked private feed token")
	}
	if current.UIDNamespace != previous.UIDNamespace {
		t.Fatal("reconnecting changed the stable event UID namespace")
	}
	newFeedResponse := httptest.NewRecorder()
	mux.ServeHTTP(newFeedResponse, httptest.NewRequest(http.MethodGet, reconnected.Path, nil))
	if newFeedResponse.Code != http.StatusOK {
		t.Fatalf("reconnected calendar status = %d, want 200", newFeedResponse.Code)
	}
}

func TestCalendarCredentialsAreOpaqueAndIndependent(t *testing.T) {
	firstToken, firstNamespace, err := newCalendarCredentials()
	if err != nil {
		t.Fatal(err)
	}
	secondToken, secondNamespace, err := newCalendarCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if len(firstToken) != 64 || len(firstNamespace) != 32 {
		t.Fatalf("unexpected credential sizes: token=%d namespace=%d", len(firstToken), len(firstNamespace))
	}
	if firstToken == secondToken || firstNamespace == secondNamespace {
		t.Fatal("calendar credentials were unexpectedly reused")
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
	chats         map[int64]domain.Chat
	profiles      map[int64]domain.PrayerProfile
	rules         map[int64][]domain.ReminderRule
	subscriptions map[int64]domain.CalendarSubscription
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{
		chats: make(map[int64]domain.Chat), profiles: make(map[int64]domain.PrayerProfile),
		rules:         make(map[int64][]domain.ReminderRule),
		subscriptions: make(map[int64]domain.CalendarSubscription),
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

func (s *fakeStorage) CalendarSubscription(_ context.Context, chatID int64) (domain.CalendarSubscription, error) {
	subscription, ok := s.subscriptions[chatID]
	if !ok {
		return domain.CalendarSubscription{}, pgx.ErrNoRows
	}
	return subscription, nil
}

func (s *fakeStorage) CalendarSubscriptionByToken(
	_ context.Context,
	feedToken string,
) (domain.CalendarSubscription, error) {
	for _, subscription := range s.subscriptions {
		if subscription.FeedToken == feedToken {
			return subscription, nil
		}
	}
	return domain.CalendarSubscription{}, pgx.ErrNoRows
}

func (s *fakeStorage) EnableCalendarSubscription(
	_ context.Context,
	chatID int64,
	feedToken string,
	uidNamespace string,
) (domain.CalendarSubscription, error) {
	subscription, ok := s.subscriptions[chatID]
	if !ok {
		subscription = domain.CalendarSubscription{
			ChatID: chatID, FeedToken: feedToken, UIDNamespace: uidNamespace,
		}
	} else if !subscription.Enabled {
		subscription.FeedToken = feedToken
	}
	subscription.Enabled = true
	s.subscriptions[chatID] = subscription
	return subscription, nil
}

func (s *fakeStorage) DisableCalendarSubscription(_ context.Context, chatID int64) error {
	subscription, ok := s.subscriptions[chatID]
	if ok && subscription.Enabled {
		subscription.Enabled = false
		s.subscriptions[chatID] = subscription
	}
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
