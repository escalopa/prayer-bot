package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/escalopa/prayer-bot/log"
	"github.com/go-telegram/bot"
)

const (
	prayerDayFormat  = "02.01.2006"
	prayerTimeFormat = "15:04"

	prayerText = `
🗓 %s

🕊 %s — %s
🌤 %s — %s
☀️ %s — %s
🌇 %s — %s
🌅 %s — %s
🌙 %s — %s
`
)

func (h *Handler) now(loc *time.Location) time.Time {
	return time.Now().In(loc).Truncate(time.Minute)
}

func (h *Handler) formatPrayerDay(botID int64, prayerDay *domain.PrayerDay, languageCode string) string {
	loc := h.cfg[botID].Location.V()
	text := h.lp.GetText(languageCode)

	return fmt.Sprintf(prayerText,
		prayerDay.Date.Format(prayerDayFormat),
		text.Prayer[int(domain.PrayerIDFajr)], prayerDay.Fajr.In(loc).Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDShuruq)], prayerDay.Shuruq.In(loc).Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDDhuhr)], prayerDay.Dhuhr.In(loc).Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDAsr)], prayerDay.Asr.In(loc).Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDMaghrib)], prayerDay.Maghrib.In(loc).Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDIsha)], prayerDay.Isha.In(loc).Format(prayerTimeFormat),
	)
}

func (h *Handler) deleteChat(ctx context.Context, chat *domain.Chat) {
	err := h.db.DeleteChat(ctx, chat.BotID, chat.ChatID)
	if err != nil {
		log.Error("remindUserJamaat: delete chat", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return
	}
	log.Warn("remindUserJamaat: deleted chat", log.BotID(chat.BotID), log.ChatID(chat.ChatID))
}

func deleteMessages(ctx context.Context, b *bot.Bot, chat *domain.Chat, ids ...int) {
	messageIDs := make([]int, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		messageIDs = append(messageIDs, id)
	}

	if len(messageIDs) == 0 {
		return // nothing to do
	}

	_, err := b.DeleteMessages(ctx, &bot.DeleteMessagesParams{
		ChatID:     chat.ChatID,
		MessageIDs: messageIDs,
	})
	if err != nil {
		log.Error("delete messages", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID), "ids", ids)
	}
}

func isBlockedErr(err error) bool {
	return strings.HasPrefix(err.Error(), bot.ErrorForbidden.Error())
}
