package db

import (
	"context"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/escalopa/prayer-bot/log"
)

type Store struct {
	primary   string
	dualWrite bool
	postgres  *Postgres
	ydb       *YDB
}

func Open(ctx context.Context) (*Store, error) {
	cfg := LoadConfig()

	store := &Store{
		primary:   cfg.Primary,
		dualWrite: cfg.DualWrite,
	}

	switch cfg.Primary {
	case "postgres":
		if cfg.DatabaseURL == "" {
			return nil, domain.ErrInternal
		}
		pg, err := openPostgres(ctx, cfg.DatabaseURL)
		if err != nil {
			return nil, err
		}
		store.postgres = pg

		if cfg.DualWrite {
			if cfg.YDBEndpoint == "" {
				return nil, domain.ErrInternal
			}
			ydbClient, err := openYDB(ctx, cfg.YDBEndpoint, cfg.YDBToken)
			if err != nil {
				return nil, err
			}
			store.ydb = ydbClient
		}
	default:
		if cfg.YDBEndpoint == "" {
			return nil, domain.ErrInternal
		}
		ydbClient, err := openYDB(ctx, cfg.YDBEndpoint, cfg.YDBToken)
		if err != nil {
			return nil, err
		}
		store.ydb = ydbClient
	}

	return store, nil
}

func (s *Store) mirrorWrite(ctx context.Context, fn func(*YDB) error) {
	if !s.dualWrite || s.ydb == nil {
		return
	}
	if err := fn(s.ydb); err != nil {
		log.Error("secondary ydb write failed", log.Err(err))
	}
}

func (s *Store) CreateChat(
	ctx context.Context,
	botID int64,
	chatID int64,
	languageCode string,
	state string,
	reminder *domain.Reminder,
) error {
	if s.primary == "postgres" {
		if err := s.postgres.CreateChat(ctx, botID, chatID, languageCode, state, reminder); err != nil {
			return err
		}
		s.mirrorWrite(ctx, func(y *YDB) error {
			return y.CreateChat(ctx, botID, chatID, languageCode, state, reminder)
		})
		return nil
	}

	return s.ydb.CreateChat(ctx, botID, chatID, languageCode, state, reminder)
}

func (s *Store) GetChat(ctx context.Context, botID int64, chatID int64) (*domain.Chat, error) {
	if s.primary == "postgres" {
		return s.postgres.GetChat(ctx, botID, chatID)
	}
	return s.ydb.GetChat(ctx, botID, chatID)
}

func (s *Store) GetChats(ctx context.Context, botID int64) ([]*domain.Chat, error) {
	if s.primary == "postgres" {
		return s.postgres.GetChats(ctx, botID)
	}
	return s.ydb.GetChats(ctx, botID)
}

func (s *Store) GetChatsByIDs(ctx context.Context, botID int64, chatIDs []int64) ([]*domain.Chat, error) {
	if s.primary == "postgres" {
		return s.postgres.GetChatsByIDs(ctx, botID, chatIDs)
	}
	return s.ydb.GetChatsByIDs(ctx, botID, chatIDs)
}

func (s *Store) GetSubscribers(ctx context.Context, botID int64) ([]int64, error) {
	if s.primary == "postgres" {
		return s.postgres.GetSubscribers(ctx, botID)
	}
	return s.ydb.GetSubscribers(ctx, botID)
}

func (s *Store) SetLanguageCode(ctx context.Context, botID int64, chatID int64, languageCode string) error {
	return s.mutate(ctx, func(ctx context.Context, pg *Postgres, y *YDB) error {
		if pg != nil {
			return pg.SetLanguageCode(ctx, botID, chatID, languageCode)
		}
		return y.SetLanguageCode(ctx, botID, chatID, languageCode)
	}, func(y *YDB) error {
		return y.SetLanguageCode(ctx, botID, chatID, languageCode)
	})
}

func (s *Store) SetSubscribed(ctx context.Context, botID int64, chatID int64, subscribed bool) error {
	return s.mutate(ctx, func(ctx context.Context, pg *Postgres, y *YDB) error {
		if pg != nil {
			return pg.SetSubscribed(ctx, botID, chatID, subscribed)
		}
		return y.SetSubscribed(ctx, botID, chatID, subscribed)
	}, func(y *YDB) error {
		return y.SetSubscribed(ctx, botID, chatID, subscribed)
	})
}

func (s *Store) SetState(ctx context.Context, botID int64, chatID int64, state string) error {
	return s.mutate(ctx, func(ctx context.Context, pg *Postgres, y *YDB) error {
		if pg != nil {
			return pg.SetState(ctx, botID, chatID, state)
		}
		return y.SetState(ctx, botID, chatID, state)
	}, func(y *YDB) error {
		return y.SetState(ctx, botID, chatID, state)
	})
}

func (s *Store) SetReminderOffset(
	ctx context.Context,
	botID int64,
	chatID int64,
	reminderType domain.ReminderType,
	offset time.Duration,
) error {
	return s.mutate(ctx, func(ctx context.Context, pg *Postgres, y *YDB) error {
		if pg != nil {
			return pg.SetReminderOffset(ctx, botID, chatID, reminderType, offset)
		}
		return y.SetReminderOffset(ctx, botID, chatID, reminderType, offset)
	}, func(y *YDB) error {
		return y.SetReminderOffset(ctx, botID, chatID, reminderType, offset)
	})
}

func (s *Store) SetJamaatEnabled(ctx context.Context, botID int64, chatID int64, enabled bool) error {
	return s.mutate(ctx, func(ctx context.Context, pg *Postgres, y *YDB) error {
		if pg != nil {
			return pg.SetJamaatEnabled(ctx, botID, chatID, enabled)
		}
		return y.SetJamaatEnabled(ctx, botID, chatID, enabled)
	}, func(y *YDB) error {
		return y.SetJamaatEnabled(ctx, botID, chatID, enabled)
	})
}

func (s *Store) SetJamaatDelay(
	ctx context.Context,
	botID int64,
	chatID int64,
	prayerID domain.PrayerID,
	delay time.Duration,
) error {
	return s.mutate(ctx, func(ctx context.Context, pg *Postgres, y *YDB) error {
		if pg != nil {
			return pg.SetJamaatDelay(ctx, botID, chatID, prayerID, delay)
		}
		return y.SetJamaatDelay(ctx, botID, chatID, prayerID, delay)
	}, func(y *YDB) error {
		return y.SetJamaatDelay(ctx, botID, chatID, prayerID, delay)
	})
}

func (s *Store) UpdateReminder(
	ctx context.Context,
	botID int64,
	chatID int64,
	reminderType domain.ReminderType,
	messageID int,
	lastAt time.Time,
) error {
	return s.mutate(ctx, func(ctx context.Context, pg *Postgres, y *YDB) error {
		if pg != nil {
			return pg.UpdateReminder(ctx, botID, chatID, reminderType, messageID, lastAt)
		}
		return y.UpdateReminder(ctx, botID, chatID, reminderType, messageID, lastAt)
	}, func(y *YDB) error {
		return y.UpdateReminder(ctx, botID, chatID, reminderType, messageID, lastAt)
	})
}

func (s *Store) DeleteChat(ctx context.Context, botID int64, chatID int64) error {
	return s.mutate(ctx, func(ctx context.Context, pg *Postgres, y *YDB) error {
		if pg != nil {
			return pg.DeleteChat(ctx, botID, chatID)
		}
		return y.DeleteChat(ctx, botID, chatID)
	}, func(y *YDB) error {
		return y.DeleteChat(ctx, botID, chatID)
	})
}

func (s *Store) GetPrayerDay(ctx context.Context, botID int64, date time.Time) (*domain.PrayerDay, error) {
	if s.primary == "postgres" {
		return s.postgres.GetPrayerDay(ctx, botID, date)
	}
	return s.ydb.GetPrayerDay(ctx, botID, date)
}

func (s *Store) GetStats(ctx context.Context, botID int64) (*domain.Stats, error) {
	if s.primary == "postgres" {
		return s.postgres.GetStats(ctx, botID)
	}
	return s.ydb.GetStats(ctx, botID)
}

func (s *Store) SetPrayerDays(ctx context.Context, botID int64, rows []*domain.PrayerDay) error {
	return s.mutate(ctx, func(ctx context.Context, pg *Postgres, y *YDB) error {
		if pg != nil {
			return pg.SetPrayerDays(ctx, botID, rows)
		}
		return y.SetPrayerDays(ctx, botID, rows)
	}, func(y *YDB) error {
		return y.SetPrayerDays(ctx, botID, rows)
	})
}

func (s *Store) mutate(
	ctx context.Context,
	primaryFn func(context.Context, *Postgres, *YDB) error,
	mirrorFn func(*YDB) error,
) error {
	if s.primary == "postgres" {
		if err := primaryFn(ctx, s.postgres, s.ydb); err != nil {
			return err
		}
		s.mirrorWrite(ctx, mirrorFn)
		return nil
	}

	return primaryFn(ctx, nil, s.ydb)
}
