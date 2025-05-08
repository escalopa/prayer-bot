package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/escalopa/prayer-bot/domain"
)

type (
	contextBotIDKey struct{}
)

func setContextBotID(ctx context.Context, botID int32) context.Context {
	return context.WithValue(ctx, contextBotIDKey{}, botID)
}

func getContextBotID(ctx context.Context) int32 {
	botID, _ := ctx.Value(contextBotIDKey{}).(int32)
	return botID
}

func (h *Handler) now(botID int32) time.Time {
	loc, _ := time.LoadLocation(h.cfg[botID].Location)
	return domain.Now(loc)
}

// formatPrayerDay formats the domain.PrayerDay into a string.
func (h *Handler) formatPrayerDay(date *domain.PrayerDay, languageCode string) string {
	text := h.lp.GetText(languageCode)
	return fmt.Sprintf(prayerText,
		text.Weekday[int(date.Date.Weekday())], date.Date.Format(prayerDayFormat),
		text.Prayer[int(domain.PrayerIDFajr)], date.Fajr.Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDShuruq)], date.Shuruq.Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDDhuhr)], date.Dhuhr.Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDAsr)], date.Asr.Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDMaghrib)], date.Maghrib.Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDIsha)], date.Isha.Format(prayerTimeFormat),
	)
}

// daysInMonth returns the number of days in a month.
func daysInMonth(m int, t time.Time) int {
	// month is incremented by 1 and day is 0 because we want the last day of the month.
	return time.Date(t.Year(), time.Month(m+1), 0, 0, 0, 0, 0, t.Location()).Day()
}

// rowsCount calculates number of rows needed to display input count and number of empty cells in the last row.
func rowsCount(inputCount, inputPerRow int) (int, int) {
	if inputCount%inputPerRow == 0 {
		return inputCount / inputPerRow, 0
	}
	reminder := inputPerRow - (inputCount % inputPerRow)
	return (inputCount / inputPerRow) + 1, reminder
}

// formatDuration formats the duration into a string with hours and minutes only.
func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dh%dm", h, m)
}
