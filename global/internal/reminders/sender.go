package reminders

import (
	"context"
	"fmt"
	"time"

	botapi "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
	"github.com/escalopa/prayer-bot/global/internal/store"
)

type MessageSender interface {
	SendMessage(context.Context, *botapi.SendMessageParams) (*models.Message, error)
}

type Sender struct {
	store   *store.Store
	planner *Planner
	bot     MessageSender
}

func NewSender(storage *store.Store, planner *Planner, bot MessageSender) *Sender {
	return &Sender{store: storage, planner: planner, bot: bot}
}

func (s *Sender) Process(ctx context.Context, task domain.DeliveryTask) error {
	if task.DeliveryKey == "" || task.ScheduleID == 0 || task.RuleID == 0 || task.ChatID == 0 {
		return fmt.Errorf("invalid delivery task")
	}
	schedule, err := s.store.Schedule(ctx, task.ScheduleID)
	if store.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load schedule: %w", err)
	}
	acquired, err := s.store.AcquireDelivery(ctx, task)
	if err != nil {
		return err
	}
	if !acquired {
		return nil
	}
	fail := func(cause error) error {
		_ = s.store.FailDelivery(ctx, task.DeliveryKey, cause)
		return cause
	}
	profile, err := s.store.Profile(ctx, task.ChatID)
	if err != nil {
		return fail(fmt.Errorf("load profile: %w", err))
	}
	rule, err := s.store.Rule(ctx, task.RuleID)
	if err != nil {
		return fail(fmt.Errorf("load rule: %w", err))
	}
	if !rule.Enabled || schedule.ChatID != task.ChatID || schedule.RuleID != task.RuleID ||
		schedule.ProfileVersion != task.ProfileVersion || profile.Version != task.ProfileVersion ||
		!schedule.NextRunAt.Equal(task.ScheduledFor) {
		return s.store.MarkDeliveryStale(ctx, task.DeliveryKey)
	}

	chat, err := s.store.Chat(ctx, task.ChatID)
	if err != nil {
		return fail(fmt.Errorf("load chat language: %w", err))
	}
	text := reminderText(rule, schedule, profile, i18n.Resolve(chat.LanguageCode))
	message, err := s.bot.SendMessage(ctx, &botapi.SendMessageParams{
		ChatID: task.ChatID, Text: text, ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		return fail(fmt.Errorf("Telegram reminder send failed"))
	}
	next, err := s.planner.Next(ctx, profile, rule, task.ScheduledFor.Add(time.Second))
	if err != nil {
		return fail(fmt.Errorf("plan next reminder: %w", err))
	}
	if err := s.store.CompleteDelivery(ctx, task, int64(message.ID), next); err != nil {
		return fail(fmt.Errorf("complete delivery: %w", err))
	}
	return nil
}

func reminderText(rule domain.ReminderRule, schedule domain.ReminderSchedule, profile domain.PrayerProfile, locale i18n.Locale) string {
	name := locale.Prayer(rule.Prayer)
	timeText := schedule.PrayerAt.In(mustLocation(profile.Timezone)).Format("15:04")
	switch rule.Kind {
	case domain.ReminderWeeklyFasting:
		return locale.Message("reminder_fasting")
	case domain.ReminderWeeklyKahf:
		return locale.Message("reminder_kahf")
	case domain.ReminderBefore:
		return fmt.Sprintf(locale.Message("reminder_before"), name, rule.OffsetMinutes, timeText)
	case domain.ReminderTomorrow:
		return fmt.Sprintf(locale.Message("reminder_tomorrow"), name, timeText)
	default:
		return fmt.Sprintf(locale.Message("reminder_at"), name)
	}
}

func mustLocation(name string) *time.Location {
	location, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return location
}
