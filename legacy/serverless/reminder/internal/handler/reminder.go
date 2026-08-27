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
	ShouldTrigger(ctx context.Context, chat *domain.Chat, prayerDay *domain.PrayerDay, now time.Time) (shouldSend bool, prayerID domain.PrayerID)
	Send(ctx context.Context, b *bot.Bot, chat *domain.Chat, prayerID domain.PrayerID, prayerDay *domain.PrayerDay) (messageID int, err error)
	Name() domain.ReminderType
}

type TomorrowReminder struct {
	lp              *languagesProvider
	botConfig       map[int64]*domain.BotConfig
	formatPrayerDay func(botID int64, prayerDay *domain.PrayerDay, languageCode string) string
}

func (r *TomorrowReminder) ShouldTrigger(ctx context.Context, chat *domain.Chat, _ *domain.PrayerDay, now time.Time) (bool, domain.PrayerID) {
	config := chat.Reminder.Tomorrow

	triggerTime := time.Date(
		now.Year(),
		now.Month(),
		now.Day()+1,                            // move to next day
		int(-config.Offset.Duration().Hours()), // get triggerTime by moving back in time
		0,
		0,
		0,
		now.Location(),
	)
	shouldSend := config.LastAt.Before(triggerTime) && (triggerTime.Before(now) || triggerTime.Equal(now))
	return shouldSend, domain.PrayerIDUnknown
}

func (r *TomorrowReminder) Send(ctx context.Context, b *bot.Bot, chat *domain.Chat, _ domain.PrayerID, prayerDay *domain.PrayerDay) (int, error) {
	deleteMessages(ctx, b, chat, chat.Reminder.Tomorrow.MessageID)
	res, err := b.SendMessage(ctx, markdownMessage(chat.ChatID, r.formatPrayerDay(chat.BotID, prayerDay.NextDay, chat.LanguageCode)))
	if err != nil {
		return 0, err
	}

	return res.ID, nil
}

func (r *TomorrowReminder) Name() domain.ReminderType { return domain.ReminderTypeTomorrow }

type SoonReminder struct {
	lp        *languagesProvider
	botConfig map[int64]*domain.BotConfig
}

func prayerTimeByID(prayerDay *domain.PrayerDay, prayerID domain.PrayerID, now time.Time) time.Time {
	var current time.Time
	var next time.Time

	switch prayerID {
	case domain.PrayerIDFajr:
		current = prayerDay.Fajr
		next = prayerDay.NextDay.Fajr
	case domain.PrayerIDShuruq:
		current = prayerDay.Shuruq
		next = prayerDay.NextDay.Shuruq
	case domain.PrayerIDDhuhr:
		current = prayerDay.Dhuhr
	case domain.PrayerIDAsr:
		current = prayerDay.Asr
	case domain.PrayerIDMaghrib:
		current = prayerDay.Maghrib
	case domain.PrayerIDIsha:
		current = prayerDay.Isha
	}

	if current.Before(now) && !next.IsZero() {
		return next
	}

	return current
}

func (r *SoonReminder) ShouldTrigger(ctx context.Context, chat *domain.Chat, prayerDay *domain.PrayerDay, now time.Time) (bool, domain.PrayerID) {
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
	var (
		found   = false
		foundID domain.PrayerID
	)
	for _, p := range prayers {
		// logic: last_at < (prayer_time - offset) AND (prayer_time - offset) <= now
		// keep updating to get the most recent match, so after downtime we skip
		// stale prayers and send only the latest one
		trigger := p.time.Add(-config.Offset.Duration())
		if config.LastAt.Before(trigger) && (trigger.Before(now) || trigger.Equal(now)) {
			found = true
			foundID = p.id
		}
	}
	return found, foundID
}

func (r *SoonReminder) Send(ctx context.Context, b *bot.Bot, chat *domain.Chat, prayerID domain.PrayerID, prayerDay *domain.PrayerDay) (int, error) {
	text := r.lp.GetText(chat.LanguageCode)
	prayer := text.Prayer[int(prayerID)]

	loc := time.Local
	if cfg, ok := r.botConfig[chat.BotID]; ok && cfg != nil {
		loc = cfg.Location.V()
	}

	now := time.Now().In(loc)
	prayerTime := prayerTimeByID(prayerDay, prayerID, now).In(loc).Format(prayerTimeFormat)

	deleteMessages(ctx, b, chat, chat.Reminder.Soon.MessageID, chat.Reminder.Arrive.MessageID)

	if chat.Reminder.Jamaat.Enabled && prayerID != domain.PrayerIDShuruq {
		delay := chat.Reminder.Jamaat.Delay.GetDelayByPrayerID(prayerID)
		jamaatTime := prayerTimeByID(prayerDay, prayerID, now).Add(delay).In(loc).Format(prayerTimeFormat)
		message := domain.StripMarkdown(
			fmt.Sprintf(text.PrayerSoon, prayer, domain.FormatDuration(chat.Reminder.Soon.Offset.Duration()), prayerTime) + "\n" +
				fmt.Sprintf(text.PrayerJamaat, domain.FormatDuration(delay+chat.Reminder.Soon.Offset.Duration()), jamaatTime),
		)
		isAnonymous := false

		res, err := b.SendPoll(ctx, &bot.SendPollParams{
			ChatID:   chat.ChatID,
			Question: message,
			Options: []models.InputPollOption{
				{Text: text.PrayerJoin},
				{Text: text.PrayerJoinDelay},
			},
			IsAnonymous: &isAnonymous,
		})
		if err != nil {
			return 0, err
		}

		return res.ID, nil
	}

	res, err := b.SendMessage(ctx, markdownMessage(chat.ChatID, fmt.Sprintf(text.PrayerSoon,
		prayer,
		domain.FormatDuration(chat.Reminder.Soon.Offset.Duration()),
		prayerTime,
	)))
	if err != nil {
		return 0, err
	}

	return res.ID, nil
}

func (r *SoonReminder) Name() domain.ReminderType { return domain.ReminderTypeSoon }

type ArriveReminder struct {
	lp *languagesProvider
}

func (r *ArriveReminder) ShouldTrigger(ctx context.Context, chat *domain.Chat, prayerDay *domain.PrayerDay, now time.Time) (bool, domain.PrayerID) {
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

	var (
		found   = false
		foundID domain.PrayerID
	)
	for _, p := range prayers {
		// logic: last_at < prayer_time AND prayer_time <= now
		// keep updating to get the most recent match, so after downtime we skip
		// stale prayers and send only the latest one
		if config.LastAt.Before(p.time) && (p.time.Before(now) || p.time.Equal(now)) {
			found = true
			foundID = p.id
		}
	}
	return found, foundID
}

func (r *ArriveReminder) Send(ctx context.Context, b *bot.Bot, chat *domain.Chat, prayerID domain.PrayerID, _ *domain.PrayerDay) (int, error) {
	deleteMessages(ctx, b, chat, chat.Reminder.Arrive.MessageID)

	text := r.lp.GetText(chat.LanguageCode)
	prayer := text.Prayer[int(prayerID)]
	params := markdownMessage(chat.ChatID, fmt.Sprintf(text.PrayerArrived, prayer))
	params.ReplyParameters = &models.ReplyParameters{
		ChatID:                   chat.ChatID,
		MessageID:                chat.Reminder.Soon.MessageID, // reply to soon's message
		AllowSendingWithoutReply: true,                         // allow sending without reply in case of soon's message is deleted
	}

	res, err := b.SendMessage(ctx, params)
	if err != nil {
		return 0, err
	}

	return res.ID, nil
}

func (r *ArriveReminder) Name() domain.ReminderType { return domain.ReminderTypeArrive }
