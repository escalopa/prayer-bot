package reminders

import (
	"strings"
	"testing"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
	"github.com/escalopa/prayer-bot/global/internal/occasions"
)

func TestReminderTextUsesSelectedLocale(t *testing.T) {
	schedule := domain.ReminderSchedule{PrayerAt: time.Date(2026, time.July, 17, 18, 45, 0, 0, time.UTC)}
	profile := domain.PrayerProfile{Timezone: "UTC"}
	rule := domain.ReminderRule{Kind: domain.ReminderBefore, Prayer: domain.PrayerMaghrib, OffsetMinutes: 10}

	text := reminderText(rule, schedule, profile, i18n.Resolve("ar"))
	for _, expected := range []string{"المغرب", "10", "<code>18:45</code>"} {
		if !strings.Contains(text, expected) {
			t.Errorf("localized reminder missing %q: %s", expected, text)
		}
	}
}

func TestWeeklyReminderTextUsesSelectedLocale(t *testing.T) {
	profile := domain.PrayerProfile{Timezone: "UTC"}
	locale := i18n.Resolve("tr")
	text := reminderText(domain.ReminderRule{Kind: domain.ReminderWeeklyKahf}, domain.ReminderSchedule{}, profile, locale)
	if !strings.Contains(text, "Kehf") || !strings.Contains(text, "Cuma") {
		t.Fatalf("unexpected Turkish Al-Kahf reminder: %s", text)
	}
}

func TestPrayerNotificationsShareOneCleanupCategory(t *testing.T) {
	for _, kind := range []domain.ReminderKind{domain.ReminderBefore, domain.ReminderAt} {
		if got := notificationCategory(kind); got != "prayer" {
			t.Fatalf("notificationCategory(%q) = %q", kind, got)
		}
	}
	if notificationCategory(domain.ReminderWeeklyKahf) == notificationCategory(domain.ReminderWeeklyFasting) {
		t.Fatal("weekly reminder categories must be cleaned independently")
	}
}

func TestNotificationLifetimeFitsTelegramDeletionWindow(t *testing.T) {
	if notificationLifetime <= 0 || notificationLifetime >= 48*time.Hour {
		t.Fatalf("notification lifetime %s must remain inside Telegram's 48-hour deletion window", notificationLifetime)
	}
}

func TestIslamicOccasionReminderIncludesLocalizedGuidanceAndSources(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	occurrence, err := occasions.Next(start, 0, occasions.CategoryFasting)
	if err != nil {
		t.Fatal(err)
	}
	rule := domain.ReminderRule{Kind: domain.ReminderOccasionFasting}
	schedule := domain.ReminderSchedule{
		LocalDate: occurrence.Date.Format("2006-01-02"),
		PrayerAt:  occurrence.Date,
	}
	text := reminderText(rule, schedule, domain.PrayerProfile{Timezone: "UTC"}, i18n.Resolve("en"))
	copy := i18n.Resolve("en").Occasion(occurrence.Definition.ID)
	for _, expected := range []string{copy.Title, copy.Action, occurrence.Definition.Sources[0].URL} {
		if !strings.Contains(text, expected) {
			t.Errorf("occasion reminder missing %q: %s", expected, text)
		}
	}
	for _, kind := range []domain.ReminderKind{
		domain.ReminderOccasionMajor,
		domain.ReminderOccasionFasting,
		domain.ReminderOccasionObserved,
	} {
		if got := notificationCategory(kind); got != "islamic_occasion" {
			t.Fatalf("notificationCategory(%q) = %q", kind, got)
		}
	}
}
