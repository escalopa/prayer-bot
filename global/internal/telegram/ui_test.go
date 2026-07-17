package telegram

import (
	"strings"
	"testing"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
)

func TestParseCommand(t *testing.T) {
	command, argument := parseCommand(" /adjust@global_bot fajr -2 ")
	if command != "adjust" || argument != "fajr -2" {
		t.Fatalf("parseCommand returned %q, %q", command, argument)
	}
	if command, _ := parseCommand("plain text"); command != "" {
		t.Fatalf("plain text parsed as command %q", command)
	}
}

func TestMainKeyboardIsPersistentAndButtonFirst(t *testing.T) {
	keyboard := mainKeyboard(i18n.Resolve("ar"))
	if !keyboard.IsPersistent || !keyboard.ResizeKeyboard {
		t.Fatal("main keyboard should be persistent and resized")
	}
	if len(keyboard.Keyboard) != 5 {
		t.Fatalf("main keyboard has %d rows, want 5", len(keyboard.Keyboard))
	}
	for index, row := range keyboard.Keyboard {
		if index == len(keyboard.Keyboard)-1 {
			if len(row) != 1 || row[0].Text != i18n.Resolve("ar").Button(i18n.ActionFeedback) {
				t.Fatalf("unexpected feedback row: %+v", row)
			}
			continue
		}
		if len(row) != 2 {
			t.Fatalf("main keyboard row has %d buttons, want 2", len(row))
		}
	}
}

func TestFormatScheduleUsesLocalizedNamesAndHTMLTimes(t *testing.T) {
	location := time.FixedZone("test", 2*60*60)
	date := time.Date(2026, time.July, 17, 0, 0, 0, 0, location)
	schedule := domain.DaySchedule{Date: date, Times: map[domain.Prayer]time.Time{
		domain.PrayerFajr:  time.Date(2026, time.July, 17, 4, 12, 0, 0, location),
		domain.PrayerDhuhr: time.Date(2026, time.July, 17, 12, 3, 0, 0, location),
	}}
	profile := domain.PrayerProfile{Timezone: "Africa/Cairo", Method: domain.MethodEgyptian}
	text := formatSchedule("مواقيت صلاة اليوم", schedule, profile, i18n.Resolve("ar"))
	for _, expected := range []string{"<b>مواقيت صلاة اليوم</b>", "17 يوليو 2026", "صفر", "1448", "الفجر", "<code>04:12</code>", "الظهر", "Africa/Cairo"} {
		if !strings.Contains(text, expected) {
			t.Errorf("formatted schedule missing %q:\n%s", expected, text)
		}
	}
	for _, unwanted := range []string{"هـ", "أم القرى", "محسوب"} {
		if strings.Contains(text, unwanted) {
			t.Errorf("formatted schedule contains unnecessary Hijri detail %q:\n%s", unwanted, text)
		}
	}
}

func TestHijriKeyboardOffersSafeRegionalCorrections(t *testing.T) {
	keyboard := hijriKeyboard(1, i18n.Resolve("en"))
	if len(keyboard.InlineKeyboard) != 2 || len(keyboard.InlineKeyboard[0]) != 5 {
		t.Fatalf("unexpected Hijri keyboard shape: %+v", keyboard.InlineKeyboard)
	}
	for _, button := range keyboard.InlineKeyboard[0] {
		if !strings.HasPrefix(button.CallbackData, "hijri:") {
			t.Errorf("unexpected Hijri callback %q", button.CallbackData)
		}
	}
}

func TestAdjustmentCallbacksStayWithinTelegramLimit(t *testing.T) {
	keyboard := adjustmentDetailKeyboard(domain.PrayerMaghrib, i18n.Resolve("ru"))
	for _, row := range keyboard.InlineKeyboard {
		for _, button := range row {
			if len(button.CallbackData) > 64 {
				t.Errorf("callback data is %d bytes: %q", len(button.CallbackData), button.CallbackData)
			}
		}
	}
}
