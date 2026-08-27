package db

import (
	"context"
	"time"

	"github.com/escalopa/prayer-bot/domain"
)

type Store struct {
	postgres *Postgres
}

func Open(ctx context.Context) (*Store, error) {
	cfg := LoadConfig()
	if cfg.DatabaseURL == "" {
		return nil, domain.ErrInternal
	}

	pg, err := openPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	return &Store{postgres: pg}, nil
}

func (s *Store) CreateChat(
	ctx context.Context,
	botID int64,
	chatID int64,
	languageCode string,
	state string,
	reminder *domain.Reminder,
) error {
	return s.postgres.CreateChat(ctx, botID, chatID, languageCode, state, reminder)
}

func (s *Store) GetChat(ctx context.Context, botID int64, chatID int64) (*domain.Chat, error) {
	return s.postgres.GetChat(ctx, botID, chatID)
}

func (s *Store) GetChats(ctx context.Context, botID int64) ([]*domain.Chat, error) {
	return s.postgres.GetChats(ctx, botID)
}

func (s *Store) GetChatsByIDs(ctx context.Context, botID int64, chatIDs []int64) ([]*domain.Chat, error) {
	return s.postgres.GetChatsByIDs(ctx, botID, chatIDs)
}

func (s *Store) GetSubscribers(ctx context.Context, botID int64) ([]int64, error) {
	return s.postgres.GetSubscribers(ctx, botID)
}

func (s *Store) SetLanguageCode(ctx context.Context, botID int64, chatID int64, languageCode string) error {
	return s.postgres.SetLanguageCode(ctx, botID, chatID, languageCode)
}

func (s *Store) SetSubscribed(ctx context.Context, botID int64, chatID int64, subscribed bool) error {
	return s.postgres.SetSubscribed(ctx, botID, chatID, subscribed)
}

func (s *Store) SetState(ctx context.Context, botID int64, chatID int64, state string) error {
	return s.postgres.SetState(ctx, botID, chatID, state)
}

func (s *Store) SetReminderOffset(
	ctx context.Context,
	botID int64,
	chatID int64,
	reminderType domain.ReminderType,
	offset time.Duration,
) error {
	return s.postgres.SetReminderOffset(ctx, botID, chatID, reminderType, offset)
}

func (s *Store) SetJamaatEnabled(ctx context.Context, botID int64, chatID int64, enabled bool) error {
	return s.postgres.SetJamaatEnabled(ctx, botID, chatID, enabled)
}

func (s *Store) SetJamaatDelay(
	ctx context.Context,
	botID int64,
	chatID int64,
	prayerID domain.PrayerID,
	delay time.Duration,
) error {
	return s.postgres.SetJamaatDelay(ctx, botID, chatID, prayerID, delay)
}

func (s *Store) UpdateReminder(
	ctx context.Context,
	botID int64,
	chatID int64,
	reminderType domain.ReminderType,
	messageID int,
	lastAt time.Time,
) error {
	return s.postgres.UpdateReminder(ctx, botID, chatID, reminderType, messageID, lastAt)
}

func (s *Store) DeleteChat(ctx context.Context, botID int64, chatID int64) error {
	return s.postgres.DeleteChat(ctx, botID, chatID)
}

func (s *Store) GetPrayerDay(ctx context.Context, botID int64, date time.Time) (*domain.PrayerDay, error) {
	return s.postgres.GetPrayerDay(ctx, botID, date)
}

func (s *Store) GetStats(ctx context.Context, botID int64) (*domain.Stats, error) {
	return s.postgres.GetStats(ctx, botID)
}

func (s *Store) SetPrayerDays(ctx context.Context, botID int64, rows []*domain.PrayerDay) error {
	return s.postgres.SetPrayerDays(ctx, botID, rows)
}
