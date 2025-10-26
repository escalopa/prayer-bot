package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/go-telegram/bot/models"
)

type (
	contextBotIDKey struct{}
	contextChatKey  struct{}
)

func setContextBotID(ctx context.Context, botID int64) context.Context {
	return context.WithValue(ctx, contextBotIDKey{}, botID)
}

func getContextBotID(ctx context.Context) int64 {
	botID, _ := ctx.Value(contextBotIDKey{}).(int64)
	return botID
}

func setContextChat(ctx context.Context, chat *domain.Chat) context.Context {
	return context.WithValue(ctx, contextChatKey{}, chat)
}

func getContextChat(ctx context.Context) *domain.Chat {
	chat, _ := ctx.Value(contextChatKey{}).(*domain.Chat)
	return chat
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

// now returns the current time with seconds and nanoseconds set to 0
func (h *Handler) now(botID int64) time.Time {
	t := time.Now().In(h.cfg[botID].Location.V())
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
}

// nowUTC returns the current time in UTC with seconds and nanoseconds set to 0
// Use this function to get prayerDay or current year for a specific botID timezone.
func (h *Handler) nowUTC(botID int64) time.Time {
	t := time.Now().In(h.cfg[botID].Location.V())
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, time.UTC)
}

// daysInMonth returns the number of days in a month.
func daysInMonth(month time.Month, year int) int {
	// month is incremented by 1 and day is 0 because we want the last day of the month.
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// layoutRowsInfo calculates number of rows needed to display input count and number of empty cells in the last row.
func layoutRowsInfo(totalItems, itemsPerRow int) (filled int, empty int) {
	if totalItems%itemsPerRow == 0 {
		return totalItems / itemsPerRow, 0
	}
	empty = itemsPerRow - (totalItems % itemsPerRow)
	filled = (totalItems / itemsPerRow) + 1
	return
}

func isGroupChat(chat models.Chat) bool {
	return chat.Type == models.ChatTypeGroup || chat.Type == models.ChatTypeSupergroup
}
