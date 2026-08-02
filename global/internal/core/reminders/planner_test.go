package reminders

import (
	"context"
	"testing"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/core/hijri"
	"github.com/escalopa/prayer-bot/global/internal/core/occasions"
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

func TestNextMondayThursdayFastingReminderUsesPreviousEvening(t *testing.T) {
	location, _ := time.LoadLocation("Africa/Cairo")
	after := time.Date(2026, 7, 17, 12, 0, 0, 0, location) // Friday
	planner := &Planner{}
	profile := domain.PrayerProfile{Timezone: "Africa/Cairo", Version: 4}
	rule := domain.ReminderRule{ID: 8, ChatID: 10, Kind: domain.ReminderWeeklyFasting, LocalTime: "20:00"}

	next, err := planner.Next(context.Background(), profile, rule, after)
	if err != nil {
		t.Fatal(err)
	}
	if got := next.NextRunAt.In(location).Format("Monday 2006-01-02 15:04"); got != "Sunday 2026-07-19 20:00" {
		t.Fatalf("unexpected fasting reminder: %s", got)
	}
	if next.LocalDate != "2026-07-20" {
		t.Fatalf("target fasting date = %s", next.LocalDate)
	}
}

func TestNextFridayKahfReminderUsesFridayMorning(t *testing.T) {
	location, _ := time.LoadLocation("Europe/London")
	after := time.Date(2026, 7, 17, 10, 0, 0, 0, location) // after this Friday's reminder
	planner := &Planner{}
	profile := domain.PrayerProfile{Timezone: "Europe/London", Version: 2}
	rule := domain.ReminderRule{ID: 9, ChatID: 10, Kind: domain.ReminderWeeklyKahf, LocalTime: "09:00"}

	next, err := planner.Next(context.Background(), profile, rule, after)
	if err != nil {
		t.Fatal(err)
	}
	if got := next.NextRunAt.In(location).Format("Monday 2006-01-02 15:04"); got != "Friday 2026-07-24 09:00" {
		t.Fatalf("unexpected Al-Kahf reminder: %s", got)
	}
}

func TestNextIslamicOccasionUsesCorrectedHijriDateAndPreviousEvening(t *testing.T) {
	location, _ := time.LoadLocation("Africa/Cairo")
	after := time.Date(2026, 1, 1, 12, 0, 0, 0, location)
	profile := domain.PrayerProfile{Timezone: "Africa/Cairo", Version: 5, HijriAdjustment: 1}
	rule := domain.ReminderRule{
		ID: 10, ChatID: 20, Kind: domain.ReminderOccasionFasting, LocalTime: "20:00",
	}
	occurrence, err := occasions.Next(after, profile.HijriAdjustment, occasions.CategoryFasting)
	if err != nil {
		t.Fatal(err)
	}

	next, err := (&Planner{}).Next(context.Background(), profile, rule, after)
	if err != nil {
		t.Fatal(err)
	}
	if next.LocalDate != occurrence.Date.Format("2006-01-02") {
		t.Fatalf("occasion date = %s, want %s", next.LocalDate, occurrence.Date.Format("2006-01-02"))
	}
	expectedRun := time.Date(
		occurrence.Date.Year(), occurrence.Date.Month(), occurrence.Date.Day()-1,
		20, 0, 0, 0, location,
	)
	if !next.NextRunAt.Equal(expectedRun.UTC()) {
		t.Fatalf("occasion run = %s, want %s", next.NextRunAt, expectedRun.UTC())
	}
}

func TestNextWhiteDaysReminderUsesPreviousEveningOfHijri13to15(t *testing.T) {
	location, _ := time.LoadLocation("Africa/Cairo")
	after := time.Date(2026, 7, 17, 12, 0, 0, 0, location)
	planner := &Planner{}
	profile := domain.PrayerProfile{Timezone: "Africa/Cairo", Version: 5}
	rule := domain.ReminderRule{ID: 9, ChatID: 10, Kind: domain.ReminderWhiteDays, LocalTime: "20:00"}

	next, err := planner.Next(context.Background(), profile, rule, after)
	if err != nil {
		t.Fatal(err)
	}
	target, err := time.ParseInLocation("2006-01-02", next.LocalDate, location)
	if err != nil {
		t.Fatal(err)
	}
	date, err := hijri.FromGregorian(target, 0)
	if err != nil {
		t.Fatal(err)
	}
	if date.Day < 13 || date.Day > 15 {
		t.Fatalf("target %s is Hijri day %d, want 13-15", next.LocalDate, date.Day)
	}
	run := next.NextRunAt.In(location)
	if got := run.Format("15:04"); got != "20:00" {
		t.Fatalf("reminder time = %s, want 20:00", got)
	}
	if got := run.AddDate(0, 0, 1).Format("2006-01-02"); got != next.LocalDate {
		t.Fatalf("reminder day %s is not the evening before %s", run.Format("2006-01-02"), next.LocalDate)
	}
	if !next.NextRunAt.After(after) {
		t.Fatalf("reminder %s is not after %s", next.NextRunAt, after)
	}

	// Planning again from just after this reminder must find the next white day,
	// never repeat the same occurrence.
	following, err := planner.Next(context.Background(), profile, rule, next.NextRunAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if following.LocalDate <= next.LocalDate {
		t.Fatalf("following white day %s does not advance past %s", following.LocalDate, next.LocalDate)
	}
}

func TestNextWhiteDaysRespectsHijriAdjustment(t *testing.T) {
	location, _ := time.LoadLocation("Africa/Cairo")
	after := time.Date(2026, 7, 17, 12, 0, 0, 0, location)
	planner := &Planner{}
	rule := domain.ReminderRule{ID: 9, ChatID: 10, Kind: domain.ReminderWhiteDays, LocalTime: "20:00"}

	base, err := planner.Next(context.Background(), domain.PrayerProfile{Timezone: "Africa/Cairo"}, rule, after)
	if err != nil {
		t.Fatal(err)
	}
	shifted, err := planner.Next(context.Background(), domain.PrayerProfile{Timezone: "Africa/Cairo", HijriAdjustment: 2}, rule, after)
	if err != nil {
		t.Fatal(err)
	}
	if base.LocalDate == shifted.LocalDate {
		t.Fatalf("a +2 day Hijri correction must shift the white day, both were %s", base.LocalDate)
	}
	target, _ := time.ParseInLocation("2006-01-02", shifted.LocalDate, location)
	date, err := hijri.FromGregorian(target, 2)
	if err != nil {
		t.Fatal(err)
	}
	if date.Day < 13 || date.Day > 15 {
		t.Fatalf("corrected target %s is Hijri day %d, want 13-15", shifted.LocalDate, date.Day)
	}
}
