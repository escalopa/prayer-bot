package domain

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   time.Duration
		want string
	}{
		{"zero", 0, "0m"},
		{"minutes only", 20 * time.Minute, "20m"},
		{"hours and minutes", 90 * time.Minute, "1h30m"},
		{"whole hours", 3 * time.Hour, "3h0m"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := FormatDuration(tt.in); got != tt.want {
				t.Fatalf("FormatDuration(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParsePrayerIDRoundTrip(t *testing.T) {
	t.Parallel()

	ids := []PrayerID{
		PrayerIDFajr,
		PrayerIDShuruq,
		PrayerIDDhuhr,
		PrayerIDAsr,
		PrayerIDMaghrib,
		PrayerIDIsha,
	}

	for _, id := range ids {
		if got := ParsePrayerID(id.String()); got != id {
			t.Errorf("ParsePrayerID(%q) = %v, want %v", id.String(), got, id)
		}
	}
}

func TestParsePrayerIDUnknown(t *testing.T) {
	t.Parallel()

	if got := ParsePrayerID("not-a-prayer"); got != PrayerIDUnknown {
		t.Fatalf("ParsePrayerID(garbage) = %v, want %v", got, PrayerIDUnknown)
	}
	if got := PrayerIDUnknown.String(); got != "unknown" {
		t.Fatalf("PrayerIDUnknown.String() = %q, want %q", got, "unknown")
	}
}

func TestPrayerDayWithOverrides(t *testing.T) {
	t.Parallel()
	loc := time.FixedZone("test", 3*60*60)
	day := NewPrayerDay(
		time.Date(2026, time.September, 8, 0, 0, 0, 0, loc),
		time.Date(2026, time.September, 8, 5, 10, 0, 0, loc),
		time.Date(2026, time.September, 8, 6, 40, 0, 0, loc),
		time.Date(2026, time.September, 8, 12, 15, 0, 0, loc),
		time.Date(2026, time.September, 8, 15, 40, 0, 0, loc),
		time.Date(2026, time.September, 8, 18, 20, 0, 0, loc),
		time.Date(2026, time.September, 8, 19, 45, 0, 0, loc),
	)
	next := *day
	next.Date = next.Date.AddDate(0, 0, 1)
	next.Fajr = next.Fajr.AddDate(0, 0, 1)
	day.NextDay = &next

	effective := day.WithOverrides(&PrayerOverrideConfig{Dhuhr: "13:00", Fajr: "05:30"}, loc)
	if got := effective.Dhuhr.Format("15:04"); got != "13:00" {
		t.Fatalf("Dhuhr = %s, want 13:00", got)
	}
	if got := effective.NextDay.Fajr.Format("15:04"); got != "05:30" {
		t.Fatalf("next day Fajr = %s, want 05:30", got)
	}
	if got := day.Dhuhr.Format("15:04"); got != "12:15" {
		t.Fatalf("original prayer day was mutated: %s", got)
	}
}

func TestPrayerDayWithOverridesUsesBotTimezone(t *testing.T) {
	t.Parallel()
	botLoc := time.FixedZone("bot", 3*60*60)
	// PostgreSQL returns timestamp-with-time-zone values in UTC on this connection.
	day := NewPrayerDay(time.Time{}, time.Date(2026, time.September, 8, 9, 15, 0, 0, time.UTC), time.Time{}, time.Date(2026, time.September, 8, 9, 15, 0, 0, time.UTC), time.Time{}, time.Time{}, time.Time{})
	effective := day.WithOverrides(&PrayerOverrideConfig{Dhuhr: "13:00"}, botLoc)
	if got := effective.Dhuhr.In(botLoc).Format("15:04"); got != "13:00" {
		t.Fatalf("Dhuhr in bot timezone = %s, want 13:00", got)
	}
}

func TestParsePrayerClock(t *testing.T) {
	t.Parallel()
	if hour, minute, err := ParsePrayerClock("13:05"); err != nil || hour != 13 || minute != 5 {
		t.Fatalf("ParsePrayerClock() = %d:%d, %v", hour, minute, err)
	}
	if _, _, err := ParsePrayerClock("25:00"); err == nil {
		t.Fatal("ParsePrayerClock accepted an invalid time")
	}
}
