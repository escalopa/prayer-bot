package telegram

import (
	"strings"
	"testing"
	"time"

	"github.com/go-telegram/bot/models"

	"github.com/escalopa/prayer-bot/global/internal/store"
)

func TestOwnerDashboardRequiresOwnerPrivateChat(t *testing.T) {
	handler := &Handler{ownerID: 42}
	owner := &models.User{ID: 42}
	if !handler.isOwner(models.Chat{Type: models.ChatTypePrivate}, owner) {
		t.Fatal("owner should have access in a private chat")
	}
	if handler.isOwner(models.Chat{Type: models.ChatTypeGroup}, owner) {
		t.Fatal("owner dashboard must not be available in a group")
	}
	if handler.isOwner(models.Chat{Type: models.ChatTypePrivate}, &models.User{ID: 7}) {
		t.Fatal("non-owner should not have access")
	}
	if handler.isOwner(models.Chat{Type: models.ChatTypePrivate}, nil) {
		t.Fatal("missing Telegram user should not have access")
	}
}

func TestAdminKeyboardProvidesEveryDashboardView(t *testing.T) {
	keyboard := adminKeyboard(adminViewOverview)
	seen := make(map[adminView]bool)
	for _, row := range keyboard.InlineKeyboard {
		for _, button := range row {
			view, ok := parseAdminView(button.CallbackData)
			if !ok {
				t.Fatalf("invalid admin callback %q", button.CallbackData)
			}
			seen[view] = true
		}
	}
	for _, view := range []adminView{
		adminViewOverview,
		adminViewActivity,
		adminViewLanguages,
		adminViewMethods,
		adminViewReminders,
		adminViewHealth,
		adminViewFeedback,
	} {
		if !seen[view] {
			t.Errorf("dashboard keyboard is missing %q", view)
		}
	}
}

func TestAdminDashboardFormatsAggregateViews(t *testing.T) {
	metrics := store.AdminDashboard{
		Users:                   100,
		Groups:                  4,
		ConfiguredUsers:         80,
		NewUsers24Hours:         2,
		NewUsers7Days:           12,
		NewUsers30Days:          40,
		ActiveUsers24Hours:      15,
		ActiveUsers7Days:        50,
		ActiveUsers30Days:       90,
		ReminderUsers:           30,
		EnabledRules:            120,
		PendingSchedules:        110,
		QueuedTasks:             3,
		SentDeliveries24Hours:   95,
		FailedDeliveries24Hours: 5,
		StaleDeliveries24Hours:  2,
		ProcessingDeliveries:    1,
		FailedUpdates24Hours:    4,
		Languages:               []store.MetricCount{{Key: "en", Count: 70}, {Key: "ar", Count: 30}},
		Methods:                 []store.MetricCount{{Key: "egyptian", Count: 60}, {Key: "mwl", Count: 20}},
		ReminderKinds:           []store.MetricCount{{Key: "prayer", Count: 30}, {Key: "fasting", Count: 10}, {Key: "kahf", Count: 8}},
	}
	now := time.Date(2026, time.July, 17, 12, 30, 0, 0, time.FixedZone("EET", 2*60*60))
	expectations := map[adminView]string{
		adminViewOverview:  "Owner dashboard",
		adminViewActivity:  "User activity",
		adminViewLanguages: "العربية",
		adminViewMethods:   "Egyptian General Authority",
		adminViewReminders: "Monday &amp; Thursday fasting",
		adminViewHealth:    "Failed bot updates",
		adminViewFeedback:  "Contact user",
	}
	for view, expected := range expectations {
		formatted := formatAdminDashboard(metrics, view, now)
		if !strings.Contains(formatted, expected) {
			t.Errorf("%s dashboard does not contain %q: %s", view, expected, formatted)
		}
		if !strings.Contains(formatted, "17 Jul 2026 10:30 UTC") {
			t.Errorf("%s dashboard has an unexpected refresh time: %s", view, formatted)
		}
	}
}

func TestPercentageHandlesEmptyDashboard(t *testing.T) {
	if got := percentage(0, 0); got != 0 {
		t.Fatalf("percentage(0, 0) = %v, want 0", got)
	}
}
