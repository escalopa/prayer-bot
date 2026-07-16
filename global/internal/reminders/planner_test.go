package reminders

import (
	"context"
	"testing"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/domain"
)

type fixedCalculator struct{ prayerAt time.Time }

func (f fixedCalculator) Day(_ context.Context, date time.Time, _ domain.PrayerProfile) (domain.DaySchedule, error) {
	at := time.Date(date.Year(), date.Month(), date.Day(), f.prayerAt.Hour(), f.prayerAt.Minute(), 0, 0, date.Location())
	return domain.DaySchedule{Times: map[domain.Prayer]time.Time{domain.PrayerFajr: at}}, nil
}

func TestNextBeforePrayer(t *testing.T) {
	location, _ := time.LoadLocation("Africa/Cairo")
	after := time.Date(2026, 7, 16, 4, 0, 0, 0, location)
	planner := &Planner{calculator: fixedCalculator{prayerAt: time.Date(2026, 7, 16, 5, 0, 0, 0, location)}}
	profile := domain.PrayerProfile{Timezone: "Africa/Cairo", Version: 3}
	rule := domain.ReminderRule{ID: 7, ChatID: 10, Kind: domain.ReminderBefore, Prayer: domain.PrayerFajr, OffsetMinutes: 15}

	next, err := planner.Next(context.Background(), profile, rule, after)
	if err != nil {
		t.Fatal(err)
	}
	if got := next.NextRunAt.In(location).Format("15:04"); got != "04:45" {
		t.Fatalf("got %s", got)
	}
	if next.ProfileVersion != 3 {
		t.Fatalf("got version %d", next.ProfileVersion)
	}
}
