package reminders

import (
	"strings"
	"testing"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
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
