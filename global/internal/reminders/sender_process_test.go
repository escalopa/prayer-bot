package reminders

import (
	"context"
	"errors"
	"testing"
	"time"

	botapi "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/escalopa/prayer-bot/global/internal/domain"
)

// fakeSenderStore records the delivery-lifecycle calls the Sender makes so tests
// can assert idempotency, staleness, and compensation behavior without Postgres.
type fakeSenderStore struct {
	schedule    domain.ReminderSchedule
	scheduleErr error
	acquired    bool
	acquireErr  error
	profile     domain.PrayerProfile
	rule        domain.ReminderRule
	chat        domain.Chat

	completePrev  int64
	completeErr   error
	completeCalls int
	completeArgs  struct {
		messageID int64
		category  string
		expiresAt time.Time
	}

	failedKeys []string
	staleKeys  []string
	cleared    [][2]int64
}

func (f *fakeSenderStore) Schedule(context.Context, int64) (domain.ReminderSchedule, error) {
	return f.schedule, f.scheduleErr
}

func (f *fakeSenderStore) AcquireDelivery(context.Context, domain.DeliveryTask) (bool, error) {
	return f.acquired, f.acquireErr
}

func (f *fakeSenderStore) FailDelivery(_ context.Context, key string, _ error) error {
	f.failedKeys = append(f.failedKeys, key)
	return nil
}

func (f *fakeSenderStore) MarkDeliveryStale(_ context.Context, key string) error {
	f.staleKeys = append(f.staleKeys, key)
	return nil
}

func (f *fakeSenderStore) Profile(context.Context, int64) (domain.PrayerProfile, error) {
	return f.profile, nil
}

func (f *fakeSenderStore) Rule(context.Context, int64) (domain.ReminderRule, error) {
	return f.rule, nil
}

func (f *fakeSenderStore) Chat(context.Context, int64) (domain.Chat, error) {
	return f.chat, nil
}

func (f *fakeSenderStore) CompleteDelivery(_ context.Context, _ domain.DeliveryTask, messageID int64, _ domain.ReminderSchedule, category string, expiresAt time.Time) (int64, error) {
	f.completeCalls++
	f.completeArgs.messageID = messageID
	f.completeArgs.category = category
	f.completeArgs.expiresAt = expiresAt
	return f.completePrev, f.completeErr
}

func (f *fakeSenderStore) ClearNotificationMessage(_ context.Context, chatID, messageID int64) error {
	f.cleared = append(f.cleared, [2]int64{chatID, messageID})
	return nil
}

type fakeBot struct {
	sendID  int
	sendErr error
	sent    []string
	deleted [][]int
}

func (f *fakeBot) SendMessage(_ context.Context, params *botapi.SendMessageParams) (*models.Message, error) {
	if f.sendErr != nil {
		return nil, f.sendErr
	}
	f.sent = append(f.sent, params.Text)
	return &models.Message{ID: f.sendID}, nil
}

func (f *fakeBot) DeleteMessages(_ context.Context, params *botapi.DeleteMessagesParams) (bool, error) {
	f.deleted = append(f.deleted, params.MessageIDs)
	return true, nil
}

type fakeNextPlanner struct {
	schedule domain.ReminderSchedule
	err      error
}

func (f fakeNextPlanner) Next(context.Context, domain.PrayerProfile, domain.ReminderRule, time.Time) (domain.ReminderSchedule, error) {
	return f.schedule, f.err
}

// alignedFixture returns a task, store, and sender whose schedule/profile/rule
// all agree, so Process proceeds to a real send instead of a staleness skip.
func alignedFixture(t *testing.T) (domain.DeliveryTask, *fakeSenderStore, *fakeBot, *Sender) {
	t.Helper()
	runAt := time.Date(2026, time.July, 20, 18, 45, 0, 0, time.UTC)
	task := domain.DeliveryTask{
		DeliveryKey: "chat3:rule2:v5", ScheduleID: 1, RuleID: 2, ChatID: 3,
		ProfileVersion: 5, ScheduledFor: runAt,
	}
	store := &fakeSenderStore{
		schedule: domain.ReminderSchedule{
			ID: 1, RuleID: 2, ChatID: 3, ProfileVersion: 5,
			PrayerAt: runAt, NextRunAt: runAt,
		},
		acquired: true,
		profile:  domain.PrayerProfile{ChatID: 3, Timezone: "UTC", Version: 5},
		rule:     domain.ReminderRule{ID: 2, ChatID: 3, Kind: domain.ReminderAt, Prayer: domain.PrayerMaghrib, Enabled: true},
		chat:     domain.Chat{TelegramChatID: 3, LanguageCode: "en"},
	}
	bot := &fakeBot{sendID: 555}
	sender := NewSender(store, fakeNextPlanner{}, bot)
	return task, store, bot, sender
}

func TestProcessSendsAndCompletesWithDeterministicExpiry(t *testing.T) {
	task, store, bot, sender := alignedFixture(t)
	fixedNow := time.Date(2026, time.July, 20, 18, 35, 0, 0, time.UTC)
	sender.now = func() time.Time { return fixedNow }

	if err := sender.Process(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if len(bot.sent) != 1 {
		t.Fatalf("expected exactly one send, got %d", len(bot.sent))
	}
	if store.completeCalls != 1 || store.completeArgs.messageID != 555 {
		t.Fatalf("unexpected completion: calls=%d messageID=%d", store.completeCalls, store.completeArgs.messageID)
	}
	if want := fixedNow.Add(notificationLifetime); !store.completeArgs.expiresAt.Equal(want) {
		t.Fatalf("expiry = %s, want %s (deterministic clock)", store.completeArgs.expiresAt, want)
	}
	if store.completeArgs.category != "prayer" {
		t.Fatalf("category = %q, want prayer", store.completeArgs.category)
	}
	if len(store.failedKeys) != 0 || len(store.staleKeys) != 0 {
		t.Fatalf("delivery should be neither failed nor stale: failed=%v stale=%v", store.failedKeys, store.staleKeys)
	}
}

func TestProcessDeletesReplacedMessageInSameCategory(t *testing.T) {
	task, store, bot, sender := alignedFixture(t)
	store.completePrev = 100 // a prior message in the "prayer" slot

	if err := sender.Process(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if len(bot.deleted) != 1 || len(bot.deleted[0]) != 1 || bot.deleted[0][0] != 100 {
		t.Fatalf("expected best-effort deletion of replaced message 100, got %v", bot.deleted)
	}
}

// TestProcessCompensatesWhenCompletionFailsAfterSend is required by the delivery
// contract: if PostgreSQL cannot commit the completion after Telegram accepted
// the message, the sender must delete that just-sent message before returning a
// retryable error, so the Cloud Tasks retry leaves only its own copy.
func TestProcessCompensatesWhenCompletionFailsAfterSend(t *testing.T) {
	task, store, bot, sender := alignedFixture(t)
	store.completeErr = errors.New("connection reset")

	err := sender.Process(context.Background(), task)
	if err == nil {
		t.Fatal("expected a retryable error when completion fails after send")
	}
	if len(bot.sent) != 1 {
		t.Fatalf("expected the first message to be sent once, got %d", len(bot.sent))
	}
	if len(bot.deleted) != 1 || bot.deleted[0][0] != 555 {
		t.Fatalf("expected compensating deletion of the orphaned message 555, got %v", bot.deleted)
	}
	if len(store.failedKeys) != 1 {
		t.Fatalf("expected the delivery to be marked failed for retry, got %v", store.failedKeys)
	}

	// The Cloud Tasks retry now succeeds and sends a fresh message. Only that
	// retry's message must survive; the orphaned first message was compensated.
	bot.sendID = 556
	store.completeErr = nil
	if err := sender.Process(context.Background(), task); err != nil {
		t.Fatalf("retry should succeed: %v", err)
	}
	if store.completeArgs.messageID != 556 {
		t.Fatalf("retry completed with message %d, want 556", store.completeArgs.messageID)
	}
}

func TestProcessCompensatesWhenPlanningNextFailsAfterSend(t *testing.T) {
	task, store, bot, sender := alignedFixture(t)
	sender.planner = fakeNextPlanner{err: errors.New("no occurrence")}

	if err := sender.Process(context.Background(), task); err == nil {
		t.Fatal("expected an error when planning the next occurrence fails")
	}
	if len(bot.deleted) != 1 || bot.deleted[0][0] != 555 {
		t.Fatalf("expected compensating deletion after a post-send planning failure, got %v", bot.deleted)
	}
	if store.completeCalls != 0 {
		t.Fatal("completion must not run when planning the next occurrence failed")
	}
}

func TestProcessMarksStaleOnProfileVersionMismatch(t *testing.T) {
	task, store, bot, sender := alignedFixture(t)
	store.profile.Version = 6 // profile changed after the task was queued

	if err := sender.Process(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if len(store.staleKeys) != 1 || store.staleKeys[0] != task.DeliveryKey {
		t.Fatalf("expected the task to be marked stale, got %v", store.staleKeys)
	}
	if len(bot.sent) != 0 {
		t.Fatal("a stale task must not send a message")
	}
}

func TestProcessSkipsWhenLeaseNotAcquired(t *testing.T) {
	task, store, bot, sender := alignedFixture(t)
	store.acquired = false // another sender instance owns the delivery

	if err := sender.Process(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if len(bot.sent) != 0 || store.completeCalls != 0 {
		t.Fatal("an unacquired delivery must do nothing")
	}
}

func TestProcessRejectsInvalidTask(t *testing.T) {
	_, _, _, sender := alignedFixture(t)
	if err := sender.Process(context.Background(), domain.DeliveryTask{}); err == nil {
		t.Fatal("expected an error for a task missing identifiers")
	}
}
