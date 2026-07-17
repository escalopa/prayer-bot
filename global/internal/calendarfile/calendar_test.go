package calendarfile

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
)

type fakeCalculator struct{}

func (fakeCalculator) Day(_ context.Context, date time.Time, profile domain.PrayerProfile) (domain.DaySchedule, error) {
	location, _ := time.LoadLocation(profile.Timezone)
	local := date.In(location)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	times := make(map[domain.Prayer]time.Time)
	for index, prayer := range calendarPrayers {
		times[prayer] = day.Add(time.Duration(4+index*2) * time.Hour)
	}
	return domain.DaySchedule{Date: day, Timezone: profile.Timezone, Times: times}, nil
}

func TestGenerateCreatesPortableLocalizedCalendar(t *testing.T) {
	profile := domain.PrayerProfile{
		Latitude: 30.044, Longitude: 31.236, Timezone: "Africa/Cairo",
		Method: domain.MethodEgyptian, Madhab: domain.MadhabShafii,
		HighLatitudeRule: domain.HighLatitudeAngleBased,
	}
	start := time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC)
	data, err := Generate(context.Background(), fakeCalculator{}, profile, i18n.Resolve("ar"), start, 2, start)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.HasPrefix(content, "BEGIN:VCALENDAR\r\n") || !strings.HasSuffix(content, "END:VCALENDAR\r\n") {
		t.Fatalf("invalid calendar envelope:\n%s", content)
	}
	if count := strings.Count(content, "BEGIN:VEVENT\r\n"); count != 12 {
		t.Fatalf("event count = %d, want 12", count)
	}
	if !strings.Contains(content, "SUMMARY:الفجر\r\n") || !strings.Contains(content, "DTSTART:20260717T010000Z\r\n") {
		t.Fatalf("calendar is missing localized prayer or UTC timestamp:\n%s", content)
	}
	if strings.Contains(content, "42@") {
		t.Fatal("calendar must not expose a Telegram chat ID")
	}
}

func TestGenerateValidatesRange(t *testing.T) {
	profile := domain.PrayerProfile{Timezone: "UTC"}
	if _, err := Generate(context.Background(), fakeCalculator{}, profile, i18n.Resolve("en"), time.Now(), 32, time.Now()); err == nil {
		t.Fatal("expected oversized calendar export to fail")
	}
}

func TestWriteLineFoldsUTF8WithoutSplittingRunes(t *testing.T) {
	var buffer bytes.Buffer
	writeLine(&buffer, "SUMMARY:"+strings.Repeat("الفجر", 20))
	for _, line := range strings.Split(strings.TrimSuffix(buffer.String(), "\r\n"), "\r\n") {
		if len(line) > 75 {
			t.Fatalf("folded line contains %d octets", len(line))
		}
		if !utf8.ValidString(line) {
			t.Fatal("folding split a UTF-8 sequence")
		}
	}
}
