package prayertime

import (
	"context"
	"testing"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/domain"
)

func TestCairoSchedule(t *testing.T) {
	profile := domain.PrayerProfile{
		Latitude: 30.044, Longitude: 31.236, Timezone: "Africa/Cairo",
		Method: domain.MethodEgyptian, Madhab: domain.MadhabShafii,
		HighLatitudeRule: domain.HighLatitudeAngleBased,
	}
	schedule, err := New().Day(context.Background(), time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC), profile)
	if err != nil {
		t.Fatal(err)
	}
	fajr, ok := schedule.At(domain.PrayerFajr)
	if !ok || fajr.Location().String() != "Africa/Cairo" {
		t.Fatalf("unexpected Fajr: %v", fajr)
	}
	maghrib, ok := schedule.At(domain.PrayerMaghrib)
	if !ok || !maghrib.After(fajr) {
		t.Fatalf("unexpected Maghrib: %v", maghrib)
	}
}

func TestRoundedCoordinates(t *testing.T) {
	lat, lon := domain.RoundedCoordinates(30.0444196, 31.2357116)
	if lat != 30.044 || lon != 31.236 {
		t.Fatalf("got %f, %f", lat, lon)
	}
}
