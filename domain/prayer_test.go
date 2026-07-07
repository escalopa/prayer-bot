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
