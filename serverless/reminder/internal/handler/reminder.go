package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/escalopa/prayer-bot/log"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type ReminderType interface {
	Check(ctx context.Context, chat *domain.Chat, prayerDay *domain.PrayerDay, now time.Time) (shouldSend bool, prayerID domain.PrayerID)
	Send(ctx context.Context, b *bot.Bot, chat *domain.Chat, prayerID domain.PrayerID, prayerDay *domain.PrayerDay) (messageID int, err error)
	Name() string
}

// TodayReminder - Daily schedule
type TodayReminder struct {
	lp          *languagesProvider
	botConfig   map[int64]*domain.BotConfig
	formatPrayerDay func(botID int64, prayerDay *domain.PrayerDay, languageCode string) string
}

func (r *TodayReminder) Check(ctx context.Context, chat *domain.Chat, prayerDay *domain.PrayerDay, now time.Time) (bool, domain.PrayerID) {
	if chat.Reminder == nil || chat.Reminder.Today.Offset == 0 {
		return false, 0 // Disabled
	}

	config := chat.Reminder.Today
	// Trigger logic: last_at + 24h - offset < now
	lastDate := config.LastAt.Truncate(24 * time.Hour)
	triggerTime := lastDate.Add(24 * time.Hour).Add(-config.Offset)
	return triggerTime.Before(now) || triggerTime.Equal(now), 0 // Not prayer-specific
}

func (r *TodayReminder) Send(ctx context.Context, b *bot.Bot, chat *domain.Chat, prayerID domain.PrayerID, prayerDay *domain.PrayerDay) error {
	// Format using the same logic as dispatcher service
	text := r.formatPrayerDay(chat.BotID, prayerDay, chat.LanguageCode)

	// Delete old message if exists
	if chat.Reminder.Today.MessageID != 0 {
		_, _ = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chat.ChatID,
			MessageID: chat.Reminder.Today.MessageID,
		})
	}

	// Send new message
	res, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   text,
	})
	if err != nil {
		return err
	}

	// Store message ID for later cleanup
	chat.Reminder.Today.MessageID = res.ID
	return nil
}

func (r *TodayReminder) UpdateState(ctx context.Context, db DB, chat *domain.Chat, messageID int, now time.Time) error {
	lastAt := now.Truncate(24 * time.Hour) // Date only
	chat.Reminder.Today.MessageID = messageID
	chat.Reminder.Today.LastAt = lastAt
	return db.UpdateReminder(ctx, chat.BotID, chat.ChatID, domain.ReminderTypeToday, messageID, lastAt)
}

func (r *TodayReminder) Name() string { return "today" }

// SoonReminder - "Prayer coming soon"
type SoonReminder struct {
	lp *languagesProvider
}

func (r *SoonReminder) Check(ctx context.Context, chat *domain.Chat, prayerDay *domain.PrayerDay, now time.Time) (bool, domain.PrayerID) {
	if chat.Reminder == nil || chat.Reminder.Soon.Offset == 0 {
		return false, 0 // Disabled
	}

	config := chat.Reminder.Soon

	// Check all prayers
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
		// Trigger logic: last_at < prayer_time - offset AND prayer_time - offset <= now
		triggerTime := p.time.Add(-config.Offset)
		if config.LastAt.Before(triggerTime) &&
			(triggerTime.Before(now) || triggerTime.Equal(now)) {
			return true, p.id
		}
	}
	return false, 0
}

func (r *SoonReminder) Send(ctx context.Context, b *bot.Bot, chat *domain.Chat, prayerID domain.PrayerID, prayerDay *domain.PrayerDay) error {
	text := r.lp.GetText(chat.LanguageCode)
	duration := chat.Reminder.Soon.Offset
	prayer := text.Prayer[int(prayerID)]
	message := fmt.Sprintf(text.PrayerSoon, prayer, domain.FormatDuration(duration))

	// Delete old message
	if chat.Reminder.Soon.MessageID != 0 {
		_, _ = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chat.ChatID,
			MessageID: chat.Reminder.Soon.MessageID,
		})
	}

	res, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chat.ChatID,
		Text:      message,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		return err
	}

	chat.Reminder.Soon.MessageID = res.ID
	return nil
}

func (r *SoonReminder) UpdateState(ctx context.Context, db DB, chat *domain.Chat, messageID int, now time.Time) error {
	chat.Reminder.Soon.MessageID = messageID
	chat.Reminder.Soon.LastAt = now
	return db.UpdateReminder(ctx, chat.BotID, chat.ChatID, domain.ReminderTypeSoon, messageID, now)
}

func (r *SoonReminder) Name() string { return "soon" }

// ArriveReminder - "Prayer time arrived"
type ArriveReminder struct {
	lp *languagesProvider
}

func (r *ArriveReminder) Check(ctx context.Context, chat *domain.Chat, prayerDay *domain.PrayerDay, now time.Time) (bool, domain.PrayerID) {
	if chat.Reminder == nil {
		return false, 0
	}

	config := chat.Reminder.Arrive
	// Offset not used (always 0), check if any prayer time has arrived

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
		// Trigger logic: last_at < prayer_time AND prayer_time <= now
		if config.LastAt.Before(p.time) &&
			(p.time.Before(now) || p.time.Equal(now)) {
			return true, p.id
		}
	}
	return false, 0
}

func (r *ArriveReminder) Send(ctx context.Context, b *bot.Bot, chat *domain.Chat, prayerID domain.PrayerID, prayerDay *domain.PrayerDay) error {
	text := r.lp.GetText(chat.LanguageCode)
	prayer := text.Prayer[int(prayerID)]
	message := fmt.Sprintf(text.PrayerArrived, prayer)

	// Delete old message
	if chat.Reminder.Arrive.MessageID != 0 {
		_, _ = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chat.ChatID,
			MessageID: chat.Reminder.Arrive.MessageID,
		})
	}

	res, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chat.ChatID,
		Text:      message,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		return err
	}

	chat.Reminder.Arrive.MessageID = res.ID
	return nil
}

func (r *ArriveReminder) UpdateState(ctx context.Context, db DB, chat *domain.Chat, messageID int, now time.Time) error {
	chat.Reminder.Arrive.MessageID = messageID
	chat.Reminder.Arrive.LastAt = now
	return db.UpdateReminder(ctx, chat.BotID, chat.ChatID, domain.ReminderTypeArrive, messageID, now)
}

func (r *ArriveReminder) Name() string { return "arrive" }

// JamaatReminder - Group Jamaat polls
type JamaatReminder struct {
	lp *languagesProvider
}

func (r *JamaatReminder) Check(ctx context.Context, chat *domain.Chat, prayerDay *domain.PrayerDay, now time.Time) (bool, domain.PrayerID) {
	if !chat.IsGroup || chat.Reminder == nil {
		return false, 0 // Only for group chats
	}

	delays := chat.Reminder.JamaatDelay // Read as-is

	prayers := []struct {
		id   domain.PrayerID
		time time.Time
	}{
		{domain.PrayerIDFajr, prayerDay.Fajr},
		// Skip Shuruq (no Jamaat for sunrise)
		{domain.PrayerIDDhuhr, prayerDay.Dhuhr},
		{domain.PrayerIDAsr, prayerDay.Asr},
		{domain.PrayerIDMaghrib, prayerDay.Maghrib},
		{domain.PrayerIDIsha, prayerDay.Isha},
	}

	// Track state per prayer using a simple in-memory approach or check based on lastAt
	// Since we removed the Jamaat ReminderConfig, we'll use a simpler approach:
	// Check if the Jamaat time (prayer_time + delay) has arrived
	for _, p := range prayers {
		delay := delays.GetDelayByPrayerID(p.id)
		if delay == 0 {
			continue // Disabled for this prayer
		}

		// Trigger logic: prayer_time + delay <= now
		// We'll check if jamaat time has arrived
		jamaatTime := p.time.Add(delay)
		if jamaatTime.Before(now) || jamaatTime.Equal(now) {
			return true, p.id
		}
	}
	return false, 0
}

func (r *JamaatReminder) Send(ctx context.Context, b *bot.Bot, chat *domain.Chat, prayerID domain.PrayerID, prayerDay *domain.PrayerDay) error {
	text := r.lp.GetText(chat.LanguageCode)
	prayer := text.Prayer[int(prayerID)]

	delay := chat.Reminder.JamaatDelay.GetDelayByPrayerID(prayerID)
	prayerTime := getPrayerTimeByID(prayerDay, prayerID)
	jamaatTime := prayerTime.Add(delay)

	// Check if Jamaat time has arrived
	now := time.Now()
	if now.After(jamaatTime) || now.Equal(jamaatTime) {
		// Jamaat has started - send arrival message
		message := fmt.Sprintf(text.PrayerArrived, prayer)
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chat.ChatID,
			Text:      message,
			ParseMode: models.ParseModeMarkdown,
		})
		if err != nil {
			return err
		}
	} else {
		// Send poll for upcoming Jamaat
		message := fmt.Sprintf(text.PrayerSoon, prayer, domain.FormatDuration(delay))
		message = strings.ReplaceAll(message, "*", "") // Remove markdown for poll

		isAnonymous := false
		_, err := b.SendPoll(ctx, &bot.SendPollParams{
			ChatID:   chat.ChatID,
			Question: message,
			Options: []models.InputPollOption{
				{Text: text.PrayerJoin, TextParseMode: models.ParseModeMarkdown},
				{Text: text.PrayerJoinDelay, TextParseMode: models.ParseModeMarkdown},
			},
			IsAnonymous: &isAnonymous,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *JamaatReminder) UpdateState(ctx context.Context, db DB, chat *domain.Chat, messageID int, now time.Time) error {
	// Jamaat reminders are stateless - no state to update
	return nil
}

func (r *JamaatReminder) Name() string { return "jamaat" }

// Helper function to get prayer time by ID
func getPrayerTimeByID(prayerDay *domain.PrayerDay, prayerID domain.PrayerID) time.Time {
	switch prayerID {
	case domain.PrayerIDFajr:
		return prayerDay.Fajr
	case domain.PrayerIDShuruq:
		return prayerDay.Shuruq
	case domain.PrayerIDDhuhr:
		return prayerDay.Dhuhr
	case domain.PrayerIDAsr:
		return prayerDay.Asr
	case domain.PrayerIDMaghrib:
		return prayerDay.Maghrib
	case domain.PrayerIDIsha:
		return prayerDay.Isha
	default:
		return time.Time{}
	}
}
