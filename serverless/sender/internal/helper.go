package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/escalopa/prayer-bot/domain"
)

type (
	contextBotIDKey struct{}
)

func setContextBotID(ctx context.Context, botID int64) context.Context {
	return context.WithValue(ctx, contextBotIDKey{}, botID)
}

func getContextBotID(ctx context.Context) int64 {
	botID, _ := ctx.Value(contextBotIDKey{}).(int64)
	return botID
}

// formatPrayerDay formats the domain.PrayerDay into a string.
func (h *Handler) formatPrayerDay(botID int64, date *domain.PrayerDay, languageCode string) string {
	loc := h.cfg[botID].Location.V()
	text := h.lp.GetText(languageCode)
	return fmt.Sprintf(prayerText,
		text.Weekday[int(date.Date.Weekday())], date.Date.Format(prayerDayFormat),
		text.Prayer[int(domain.PrayerIDFajr)], date.Fajr.In(loc).Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDShuruq)], date.Shuruq.In(loc).Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDDhuhr)], date.Dhuhr.In(loc).Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDAsr)], date.Asr.In(loc).Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDMaghrib)], date.Maghrib.In(loc).Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDIsha)], date.Isha.In(loc).Format(prayerTimeFormat),
	)
}

func (h *Handler) now(botID int64) time.Time {
	return domain.Now(h.cfg[botID].Location.V())
}

// daysInMonth returns the number of days in a month.
func daysInMonth(month time.Month, t time.Time) int {
	// month is incremented by 1 and day is 0 because we want the last day of the month.
	return time.Date(t.Year(), month+1, 0, 0, 0, 0, 0, t.Location()).Day()
}

// layoutRowsInfo calculates number of rows needed to display input count and number of empty cells in the last row.
func layoutRowsInfo(totalItems, itemsPerRow int) (int, int) {
	if totalItems%itemsPerRow == 0 {
		return totalItems / itemsPerRow, 0
	}
	empty := itemsPerRow - (totalItems % itemsPerRow)
	return (totalItems / itemsPerRow) + 1, empty
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

func unmarshalPayload[T any](data interface{}) (*T, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var t T
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, fmt.Errorf("unmarshal: %T got: %T: %v", t, data, err)
	}

	return &t, nil
}
