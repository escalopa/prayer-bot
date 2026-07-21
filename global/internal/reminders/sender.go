package reminders

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	botapi "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
	"github.com/escalopa/prayer-bot/global/internal/occasions"
	"github.com/escalopa/prayer-bot/global/internal/store"
)

type MessageSender interface {
	SendMessage(context.Context, *botapi.SendMessageParams) (*models.Message, error)
	DeleteMessages(context.Context, *botapi.DeleteMessagesParams) (bool, error)
}

// SenderStore is the subset of *store.Store the Sender depends on. Declaring it
// as an interface keeps the delivery orchestration unit-testable with fakes,
// the same way DispatchStore and PlanningStore already isolate their stores.
type SenderStore interface {
	Schedule(context.Context, int64) (domain.ReminderSchedule, error)
	AcquireDelivery(context.Context, domain.DeliveryTask) (bool, error)
	FailDelivery(context.Context, string, error) error
	MarkDeliveryStale(context.Context, string) error
	Profile(context.Context, int64) (domain.PrayerProfile, error)
	Rule(context.Context, int64) (domain.ReminderRule, error)
	Chat(context.Context, int64) (domain.Chat, error)
	CompleteDelivery(context.Context, domain.DeliveryTask, int64, domain.ReminderSchedule, string, time.Time) (int64, error)
	ClearNotificationMessage(context.Context, int64, int64) error
}

// nextPlanner is satisfied by *Planner. It lets the Sender be tested without a
// real prayer calculator or planning store.
type nextPlanner interface {
	Next(context.Context, domain.PrayerProfile, domain.ReminderRule, time.Time) (domain.ReminderSchedule, error)
}

const notificationLifetime = 36 * time.Hour

type Sender struct {
	store   SenderStore
	planner nextPlanner
	bot     MessageSender
	// now is injected so the scheduled cleanup expiry is deterministic in
	// tests. Production wiring leaves it as time.Now.
	now func() time.Time
}

func NewSender(storage SenderStore, planner nextPlanner, bot MessageSender) *Sender {
	return &Sender{store: storage, planner: planner, bot: bot, now: time.Now}
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
	// After a successful send, any failure must compensate by deleting the
	// just-sent message before returning a retryable error. The message slot
	// never recorded this message ID, so the Cloud Tasks retry cannot find and
	// replace it; without compensation the retry's send would leave a duplicate.
	failAfterSend := func(cause error) error {
		_, _ = s.bot.DeleteMessages(ctx, &botapi.DeleteMessagesParams{
			ChatID: task.ChatID, MessageIDs: []int{message.ID},
		})
		return fail(cause)
	}
	next, err := s.planner.Next(ctx, profile, rule, task.ScheduledFor.Add(time.Second))
	if err != nil {
		return failAfterSend(fmt.Errorf("plan next reminder: %w", err))
	}
	previousMessageID, err := s.store.CompleteDelivery(
		ctx,
		task,
		int64(message.ID),
		next,
		notificationCategory(rule.Kind),
		s.now().Add(notificationLifetime),
	)
	if err != nil {
		return failAfterSend(fmt.Errorf("complete delivery: %w", err))
	}
	if previousMessageID != 0 && previousMessageID != int64(message.ID) {
		// Best effort for immediate chat cleanup. The transaction also created
		// a durable deletion task, so a transient Telegram error is retried.
		_, _ = s.bot.DeleteMessages(ctx, &botapi.DeleteMessagesParams{
			ChatID: task.ChatID, MessageIDs: []int{int(previousMessageID)},
		})
	}
	return nil
}

func (s *Sender) Delete(ctx context.Context, task domain.MessageDeletionTask) error {
	if task.DeletionKey == "" || task.ChatID == 0 || task.MessageID == 0 {
		return fmt.Errorf("invalid message deletion task")
	}
	if _, err := s.bot.DeleteMessages(ctx, &botapi.DeleteMessagesParams{
		ChatID:     task.ChatID,
		MessageIDs: []int{int(task.MessageID)},
	}); err != nil {
		return fmt.Errorf("Telegram reminder cleanup failed: %w", err)
	}
	if err := s.store.ClearNotificationMessage(ctx, task.ChatID, task.MessageID); err != nil {
		return fmt.Errorf("clear notification message slot: %w", err)
	}
	return nil
}

func notificationCategory(kind domain.ReminderKind) string {
	switch kind {
	case domain.ReminderWeeklyFasting:
		return "weekly_fasting"
	case domain.ReminderWeeklyKahf:
		return "weekly_kahf"
	case domain.ReminderTomorrow:
		return "tomorrow"
	case domain.ReminderOccasionMajor, domain.ReminderOccasionFasting, domain.ReminderOccasionObserved:
		return "islamic_occasion"
	default:
		// Before-prayer and at-prayer messages intentionally share a slot.
		// A pre-reminder replaces the previous prayer, and the arrival message
		// replaces that pre-reminder.
		return "prayer"
	}
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
	case domain.ReminderOccasionMajor, domain.ReminderOccasionFasting, domain.ReminderOccasionObserved:
		return occasionReminderText(rule, schedule, profile, locale)
	default:
		return fmt.Sprintf(locale.Message("reminder_at"), name)
	}
}

func occasionReminderText(rule domain.ReminderRule, schedule domain.ReminderSchedule, profile domain.PrayerProfile, locale i18n.Locale) string {
	category, ok := occasionCategory(rule.Kind)
	if !ok {
		return ""
	}
	location := mustLocation(profile.Timezone)
	date, err := time.ParseInLocation("2006-01-02", schedule.LocalDate, location)
	if err != nil {
		return ""
	}
	occurrence, ok := occasions.OnDate(date, profile.HijriAdjustment, category)
	if !ok {
		return ""
	}
	copy := locale.Occasion(occurrence.Definition.ID)
	var builder strings.Builder
	fmt.Fprintf(&builder, "<b>%s %s</b>\n📅 %d %s %d\n\n%s\n\n💡 %s",
		html.EscapeString(occurrence.Definition.Emoji),
		html.EscapeString(copy.Title),
		date.Day(), html.EscapeString(locale.Month(int(date.Month()))), date.Year(),
		html.EscapeString(copy.Summary),
		html.EscapeString(copy.Action),
	)
	if len(occurrence.Definition.Sources) > 0 {
		builder.WriteString("\n\n📚 ")
		for index, source := range occurrence.Definition.Sources {
			if index > 0 {
				builder.WriteString(" · ")
			}
			fmt.Fprintf(&builder, `<a href="%s">%s</a>`,
				html.EscapeString(source.URL), html.EscapeString(source.Label))
		}
	}
	return builder.String()
}

func mustLocation(name string) *time.Location {
	location, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return location
}
