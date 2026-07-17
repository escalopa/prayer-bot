package i18n

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/escalopa/prayer-bot/global/internal/domain"
)

func TestResolveNormalizesTelegramLanguageTags(t *testing.T) {
	if got := Resolve("ar-EG").Code; got != "ar" {
		t.Fatalf("Resolve(ar-EG) = %q, want ar", got)
	}
	if got := Resolve("uz_UZ").Code; got != "uz" {
		t.Fatalf("Resolve(uz_UZ) = %q, want uz", got)
	}
	if got := Resolve("unknown").Code; got != "en" {
		t.Fatalf("Resolve(unknown) = %q, want en", got)
	}
}

func TestLocalizedFormatStringsAcceptExpectedArguments(t *testing.T) {
	samples := map[string][]any{
		"location_set":      {"Cairo", "Africa/Cairo", "Egyptian"},
		"next_prayer":       {"Fajr", "04:15", "Africa/Cairo"},
		"adjust_prayer":     {"Fajr", 2},
		"method_saved":      {"Egyptian"},
		"madhab_saved":      {"Hanafi"},
		"highlat_saved":     {"Angle based"},
		"adjust_saved":      {"Fajr", 2},
		"reminder_at":       {"Fajr"},
		"reminder_before":   {"Fajr", 10, "04:15"},
		"reminder_tomorrow": {"Fajr", "04:15"},
	}
	for _, locale := range Supported() {
		for key, arguments := range samples {
			formatted := fmt.Sprintf(locale.Text[key], arguments...)
			if strings.Contains(formatted, "%!") {
				t.Errorf("%s text %q has incompatible formatting: %s", locale.Code, key, formatted)
			}
		}
	}
}

func TestLocalesAreCompleteAndWithinTelegramLimits(t *testing.T) {
	buttonKeys := append(append([]string{}, mainActions...),
		"share_location", "method", "madhab", "highlat", "adjustments", "back", "close", "enable", "disable", "main_menu")
	textKeys := []string{
		"welcome", "location_prompt", "location_group", "location_set", "invalid_location", "need_location",
		"today_title", "tomorrow_title", "next_prayer", "settings_title", "timezone", "method", "madhab",
		"highlat", "adjustments", "choose_method", "choose_madhab", "choose_highlat", "choose_adjustment",
		"adjust_prayer", "method_saved", "madhab_saved", "highlat_saved", "adjust_saved", "reminders_title",
		"reminders_on", "reminders_off", "reminders_enabled", "reminders_disabled", "choose_language",
		"language_saved", "admin_only", "unknown", "deleted", "help", "privacy", "reminder_at",
		"reminder_before", "reminder_tomorrow",
	}
	commandKeys := []string{"location", "today", "tomorrow", "next", "settings", "remind", "language", "privacy", "help"}
	prayers := []domain.Prayer{domain.PrayerFajr, domain.PrayerSunrise, domain.PrayerDhuhr, domain.PrayerAsr, domain.PrayerMaghrib, domain.PrayerIsha}

	seen := make(map[string]bool)
	for _, locale := range Supported() {
		if seen[locale.Code] {
			t.Fatalf("duplicate locale %q", locale.Code)
		}
		seen[locale.Code] = true
		if utf8.RuneCountInString(locale.BotName) == 0 || utf8.RuneCountInString(locale.BotName) > 64 {
			t.Errorf("%s bot name has invalid length %d", locale.Code, utf8.RuneCountInString(locale.BotName))
		}
		if utf8.RuneCountInString(locale.ShortDescription) == 0 || utf8.RuneCountInString(locale.ShortDescription) > 120 {
			t.Errorf("%s short description has invalid length %d", locale.Code, utf8.RuneCountInString(locale.ShortDescription))
		}
		if utf8.RuneCountInString(locale.Description) == 0 || utf8.RuneCountInString(locale.Description) > 512 {
			t.Errorf("%s description has invalid length %d", locale.Code, utf8.RuneCountInString(locale.Description))
		}
		if len(locale.Months) != 12 {
			t.Errorf("%s has %d months", locale.Code, len(locale.Months))
		}
		for _, key := range buttonKeys {
			if locale.Button(key) == "" {
				t.Errorf("%s missing button %q", locale.Code, key)
			}
		}
		for _, key := range textKeys {
			if locale.Text[key] == "" {
				t.Errorf("%s missing text %q", locale.Code, key)
			}
		}
		for _, key := range commandKeys {
			if locale.Commands[key] == "" {
				t.Errorf("%s missing command %q", locale.Code, key)
			}
		}
		for _, prayer := range prayers {
			if locale.Prayers[prayer] == "" {
				t.Errorf("%s missing prayer %q", locale.Code, prayer)
			}
		}
	}
	if len(seen) != 8 {
		t.Fatalf("got %d supported locales, want 8", len(seen))
	}
}

func TestActionForTextRecognizesEveryLocalizedMainButton(t *testing.T) {
	labels := make(map[string]string)
	for _, locale := range Supported() {
		for _, action := range mainActions {
			label := locale.Button(action)
			if prior, exists := labels[label]; exists && prior != action {
				t.Fatalf("localized label %q maps to both %q and %q", label, prior, action)
			}
			labels[label] = action
			if got := ActionForText(label); got != action {
				t.Errorf("ActionForText(%q) = %q, want %q", label, got, action)
			}
		}
	}
}
