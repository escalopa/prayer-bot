package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type ReminderType interface {
	Check(ctx context.Context, chat *domain.Chat, prayerDay *domain.PrayerDay, now time.Time) (shouldSend bool, prayerID domain.PrayerID)
	Send(ctx context.Context, b *bot.Bot, chat *domain.Chat, prayerID domain.PrayerID, prayerDay *domain.PrayerDay) (messageID int, err error)
	Name() domain.ReminderType
}

type TomorrowReminder struct {
	lp              *languagesProvider
	botConfig       map[int64]*domain.BotConfig
	formatPrayerDay func(botID int64, prayerDay *domain.PrayerDay, languageCode string) string
}

func (r *TomorrowReminder) Check(ctx context.Context, chat *domain.Chat, prayerDay *domain.PrayerDay, now time.Time) (bool, domain.PrayerID) {
	config := chat.Reminder.Tomorrow
	// Trigger logic: last_at + 24h - offset < now
	lastDate := config.LastAt.Truncate(24 * time.Hour)
	triggerTime := lastDate.Add(24 * time.Hour).Add(-config.Offset)
	return triggerTime.Before(now) || triggerTime.Equal(now), domain.PrayerIDUnknown
}

func (r *TomorrowReminder) Send(ctx context.Context, b *bot.Bot, chat *domain.Chat, prayerID domain.PrayerID, prayerDay *domain.PrayerDay) (int, error) {
	deleteMessages(ctx, b, chat, chat.Reminder.Tomorrow.MessageID)
	res, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   r.formatPrayerDay(chat.BotID, prayerDay, chat.LanguageCode),
	})
	if err != nil {
		return 0, err
	}

	return res.ID, nil
}

func (r *TomorrowReminder) Name() domain.ReminderType { return domain.ReminderTypeTomorrow }

type SoonReminder struct {
	lp *languagesProvider
}

func (r *SoonReminder) Check(ctx context.Context, chat *domain.Chat, prayerDay *domain.PrayerDay, now time.Time) (bool, domain.PrayerID) {
	prayers := []struct {
		id   domain.PrayerID
		time time.Time
	}{
		{domain.PrayerIDFajr, prayerDay.Fajr},
		{domain.PrayerIDShuruq, prayerDay.Shuruq},
		{domain.PrayerIDDhuhr, prayerDay.Dhuhr},
		{domain.PrayerIDAsr, prayerDay.Asr},
		{domain.PrayerIDMaghrib, prayerDay.Maghrib},
		{domain.PrayerIDIsha, prayerDay.Isha},
		{domain.PrayerIDFajr, prayerDay.NextDay.Fajr},
		{domain.PrayerIDShuruq, prayerDay.NextDay.Shuruq},
	}

	config := chat.Reminder.Soon
	for _, p := range prayers {
		// logic: last_at < (prayer_time - offset) AND (prayer_time - offset) <= now
		trigger := p.time.Add(-config.Offset)
		if config.LastAt.Before(trigger) && (trigger.Before(now) || trigger.Equal(now)) {
			return true, p.id
		}
	}
	return false, 0
}

func (r *SoonReminder) Send(ctx context.Context, b *bot.Bot, chat *domain.Chat, prayerID domain.PrayerID, prayerDay *domain.PrayerDay) (int, error) {
	text := r.lp.GetText(chat.LanguageCode)
	prayer := text.Prayer[int(prayerID)]

	deleteMessages(ctx, b, chat, chat.Reminder.Soon.MessageID, chat.Reminder.Arrive.MessageID)

	if chat.Reminder.Jamaat.Enabled {
		delay := chat.Reminder.Jamaat.Delay.GetDelayByPrayerID(prayerID)
		message := fmt.Sprintf("%s\n%s",
			fmt.Sprintf(text.PrayerSoon, prayer, domain.FormatDuration(chat.Reminder.Soon.Offset)),
			fmt.Sprintf(text.PrayerJamaat, domain.FormatDuration(delay)),
		)
		isAnonymous := false

		res, err := b.SendPoll(ctx, &bot.SendPollParams{
			ChatID:   chat.ChatID,
			Question: message,
			Options: []models.InputPollOption{
				{Text: text.PrayerJoin, TextParseMode: models.ParseModeMarkdown},
				{Text: text.PrayerJoinDelay, TextParseMode: models.ParseModeMarkdown},
			},
			IsAnonymous: &isAnonymous,
		})
		if err != nil {
			return 0, err
		}

		return res.ID, nil
	}

	duration := chat.Reminder.Soon.Offset
	message := fmt.Sprintf(text.PrayerSoon, prayer, domain.FormatDuration(duration))

	res, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chat.ChatID,
		Text:      message,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		return 0, err
	}

	return res.ID, nil
}

func (r *SoonReminder) Name() domain.ReminderType { return domain.ReminderTypeSoon }

type ArriveReminder struct {
	lp *languagesProvider
}

func (r *ArriveReminder) Check(ctx context.Context, chat *domain.Chat, prayerDay *domain.PrayerDay, now time.Time) (bool, domain.PrayerID) {
	if chat.Reminder == nil {
		return false, 0
	}

	config := chat.Reminder.Arrive

	prayers := []struct {
		id   domain.PrayerID
		time time.Time
	}{
		{domain.PrayerIDFajr, prayerDay.Fajr},
		{domain.PrayerIDShuruq, prayerDay.Shuruq},
		{domain.PrayerIDDhuhr, prayerDay.Dhuhr},
		{domain.PrayerIDAsr, prayerDay.Asr},
		{domain.PrayerIDMaghrib, prayerDay.Maghrib},
		{domain.PrayerIDIsha, prayerDay.Isha},
	}

	for _, p := range prayers {
		// logic: last_at < prayer_time AND prayer_time <= now
		if config.LastAt.Before(p.time) &&
			(p.time.Before(now) || p.time.Equal(now)) {
			return true, p.id
		}
	}
	return false, 0
}

func (r *ArriveReminder) Send(ctx context.Context, b *bot.Bot, chat *domain.Chat, prayerID domain.PrayerID, prayerDay *domain.PrayerDay) (int, error) {
	deleteMessages(ctx, b, chat, chat.Reminder.Arrive.MessageID)

	text := r.lp.GetText(chat.LanguageCode)
	prayer := text.Prayer[int(prayerID)]
	message := fmt.Sprintf(text.PrayerArrived, prayer)

	params := &bot.SendMessageParams{
		ChatID:    chat.ChatID,
		Text:      message,
		ParseMode: models.ParseModeMarkdown,
		ReplyParameters: &models.ReplyParameters{
			ChatID:                   chat.ChatID,
			MessageID:                chat.Reminder.Soon.MessageID, // reply to soon's message
			AllowSendingWithoutReply: true,                         // allow sending without reply in case of soon's message is deleted
		},
	}

	res, err := b.SendMessage(ctx, params)
	if err != nil {
		return 0, err
	}

	return res.ID, nil
}

func (r *ArriveReminder) Name() domain.ReminderType { return domain.ReminderTypeArrive }
