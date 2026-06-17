package db

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/escalopa/prayer-bot/log"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const pgComponent = "db.postgres"

func logPG(op, detail string, args ...any) {
	log.Error(pgComponent+"."+op+": "+detail,
		append([]any{log.Op(op)}, args...)...)
}

type Postgres struct {
	pool *pgxpool.Pool
}

func openPostgres(ctx context.Context, databaseURL string) (*Postgres, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	return &Postgres{pool: pool}, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func normalizeUTCDate(date time.Time) time.Time {
	y, m, d := date.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func unmarshalReminder(reminderJSON []byte, botID, chatID int64) (*domain.Reminder, error) {
	if len(reminderJSON) == 0 {
		return &domain.Reminder{}, nil
	}

	var reminder domain.Reminder
	if err := json.Unmarshal(reminderJSON, &reminder); err != nil {
		logPG("unmarshalReminder", "failed to decode reminder json", log.Err(err), log.BotID(botID), log.ChatID(chatID))
		return nil, domain.ErrUnmarshalJSON
	}

	return &reminder, nil
}

func scanChat(row pgx.Row, withSubscribed bool) (*domain.Chat, error) {
	chat := &domain.Chat{}
	var reminderJSON []byte

	var err error
	if withSubscribed {
		err = row.Scan(
			&chat.BotID,
			&chat.ChatID,
			&chat.State,
			&chat.LanguageCode,
			&chat.Subscribed,
			&reminderJSON,
		)
	} else {
		err = row.Scan(
			&chat.BotID,
			&chat.ChatID,
			&chat.State,
			&chat.LanguageCode,
			&reminderJSON,
		)
	}
	if err != nil {
		return nil, err
	}

	reminder, err := unmarshalReminder(reminderJSON, chat.BotID, chat.ChatID)
	if err != nil {
		return nil, err
	}
	chat.Reminder = reminder

	return chat, nil
}

//revive:disable:argument-limit
func (p *Postgres) CreateChat(
	ctx context.Context,
	botID int64,
	chatID int64,
	languageCode string,
	state string,
	reminder *domain.Reminder,
) error {
	reminderJSON, err := json.Marshal(reminder)
	if err != nil {
		logPG("CreateChat.marshalReminder", "failed to encode reminder json", log.Err(err), log.BotID(botID), log.ChatID(chatID))
		return domain.ErrInternal
	}

	_, err = p.pool.Exec(ctx, `
		INSERT INTO chats (bot_id, chat_id, language_code, state, reminder, created_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6)
	`, botID, chatID, languageCode, state, string(reminderJSON), time.Now().UTC())
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrAlreadyExists
		}
		logPG("CreateChat", "insert failed", log.Err(err), log.BotID(botID), log.ChatID(chatID))
		return domain.ErrInternal
	}

	return nil
}

func (p *Postgres) GetChat(ctx context.Context, botID int64, chatID int64) (chat *domain.Chat, _ error) {
	row := p.pool.QueryRow(ctx, `
		SELECT bot_id, chat_id, state, language_code, COALESCE(subscribed, false), reminder
		FROM chats
		WHERE bot_id = $1 AND chat_id = $2
	`, botID, chatID)

	chat, err := scanChat(row, true)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		logPG("GetChat", "select failed", log.Err(err), log.BotID(botID), log.ChatID(chatID))
		return nil, domain.ErrInternal
	}

	return chat, nil
}

func (p *Postgres) GetChats(ctx context.Context, botID int64) (chats []*domain.Chat, _ error) {
	rows, err := p.pool.Query(ctx, `
		SELECT bot_id, chat_id, state, language_code, COALESCE(subscribed, false), reminder
		FROM chats
		WHERE bot_id = $1
	`, botID)
	if err != nil {
		logPG("GetChats", "query failed", log.Err(err), log.BotID(botID))
		return nil, domain.ErrInternal
	}
	defer rows.Close()

	for rows.Next() {
		chat, err := scanChat(rows, true)
		if err != nil {
			if errors.Is(err, domain.ErrUnmarshalJSON) {
				return nil, err
			}
			logPG("GetChats.scanRow", "scan failed", log.Err(err), log.BotID(botID))
			return nil, domain.ErrInternal
		}
		chats = append(chats, chat)
	}

	if err := rows.Err(); err != nil {
		logPG("GetChats.iterateRows", "iteration failed", log.Err(err), log.BotID(botID))
		return nil, domain.ErrInternal
	}

	return chats, nil
}

func (p *Postgres) GetChatsByIDs(ctx context.Context, botID int64, chatIDs []int64) (chats []*domain.Chat, _ error) {
	rows, err := p.pool.Query(ctx, `
		SELECT bot_id, chat_id, state, language_code, reminder
		FROM chats
		WHERE bot_id = $1 AND chat_id = ANY($2)
	`, botID, chatIDs)
	if err != nil {
		logPG("GetChatsByIDs", "query failed", log.Err(err), log.BotID(botID))
		return nil, domain.ErrInternal
	}
	defer rows.Close()

	for rows.Next() {
		chat, err := scanChat(rows, false)
		if err != nil {
			if errors.Is(err, domain.ErrUnmarshalJSON) {
				return nil, err
			}
			logPG("GetChatsByIDs.scanRow", "scan failed", log.Err(err), log.BotID(botID))
			return nil, domain.ErrInternal
		}
		chats = append(chats, chat)
	}

	if err := rows.Err(); err != nil {
		logPG("GetChatsByIDs.iterateRows", "iteration failed", log.Err(err), log.BotID(botID))
		return nil, domain.ErrInternal
	}

	return chats, nil
}

func (p *Postgres) GetSubscribers(ctx context.Context, botID int64) (chatIDs []int64, _ error) {
	rows, err := p.pool.Query(ctx, `
		SELECT chat_id
		FROM chats
		WHERE bot_id = $1 AND COALESCE(subscribed, false) = true
	`, botID)
	if err != nil {
		logPG("GetSubscribers", "query failed", log.Err(err), log.BotID(botID))
		return nil, domain.ErrInternal
	}
	defer rows.Close()

	for rows.Next() {
		var chatID int64
		if err := rows.Scan(&chatID); err != nil {
			logPG("GetSubscribers.scanRow", "scan failed", log.Err(err), log.BotID(botID))
			return nil, domain.ErrInternal
		}
		chatIDs = append(chatIDs, chatID)
	}

	if err := rows.Err(); err != nil {
		logPG("GetSubscribers.iterateRows", "iteration failed", log.Err(err), log.BotID(botID))
		return nil, domain.ErrInternal
	}

	return chatIDs, nil
}

func (p *Postgres) SetLanguageCode(ctx context.Context, botID int64, chatID int64, languageCode string) error {
	_, err := p.pool.Exec(ctx, `
		UPDATE chats
		SET language_code = $3
		WHERE bot_id = $1 AND chat_id = $2
	`, botID, chatID, languageCode)
	if err != nil {
		logPG("SetLanguageCode", "update failed", log.Err(err), log.BotID(botID), log.ChatID(chatID))
		return domain.ErrInternal
	}

	return nil
}

func (p *Postgres) SetSubscribed(ctx context.Context, botID int64, chatID int64, subscribed bool) error {
	_, err := p.pool.Exec(ctx, `
		UPDATE chats
		SET subscribed = $3,
		    subscribed_at = CASE WHEN $3 THEN NOW() AT TIME ZONE 'UTC' ELSE NULL END
		WHERE bot_id = $1 AND chat_id = $2
	`, botID, chatID, subscribed)
	if err != nil {
		logPG("SetSubscribed", "update failed", log.Err(err), log.BotID(botID), log.ChatID(chatID))
		return domain.ErrInternal
	}

	return nil
}

func (p *Postgres) SetReminderOffset(ctx context.Context, botID int64, chatID int64, reminderType domain.ReminderType, offset time.Duration) error {
	return p.updateReminder(ctx, botID, chatID, func(reminder *domain.Reminder) {
		switch reminderType {
		case domain.ReminderTypeTomorrow:
			reminder.Tomorrow.Offset = domain.Duration(offset)
			reminder.Tomorrow.LastAt = time.Now()
		case domain.ReminderTypeSoon:
			reminder.Soon.Offset = domain.Duration(offset)
			reminder.Soon.LastAt = time.Now()
		}
	})
}

func (p *Postgres) SetJamaatEnabled(ctx context.Context, botID int64, chatID int64, enabled bool) error {
	return p.updateReminder(ctx, botID, chatID, func(reminder *domain.Reminder) {
		reminder.Jamaat.Enabled = enabled
	})
}

func (p *Postgres) SetJamaatDelay(ctx context.Context, botID int64, chatID int64, prayerID domain.PrayerID, delay time.Duration) error {
	return p.updateReminder(ctx, botID, chatID, func(reminder *domain.Reminder) {
		reminder.Jamaat.Delay.SetDelayByPrayerID(prayerID, delay)
	})
}

func (p *Postgres) SetState(ctx context.Context, botID int64, chatID int64, state string) error {
	_, err := p.pool.Exec(ctx, `
		UPDATE chats
		SET state = $3
		WHERE bot_id = $1 AND chat_id = $2
	`, botID, chatID, state)
	if err != nil {
		logPG("SetState", "update failed", log.Err(err), log.BotID(botID), log.ChatID(chatID))
		return domain.ErrInternal
	}

	return nil
}

func (p *Postgres) GetPrayerDay(ctx context.Context, botID int64, date time.Time) (prayerDay *domain.PrayerDay, _ error) {
	date = normalizeUTCDate(date)
	nextDate := date.Add(24 * time.Hour)

	rows, err := p.pool.Query(ctx, `
		SELECT prayer_date, fajr, shuruq, dhuhr, asr, maghrib, isha
		FROM prayers
		WHERE bot_id = $1 AND prayer_date IN ($2, $3)
		ORDER BY prayer_date
	`, botID, date, nextDate)
	if err != nil {
		logPG("GetPrayerDay", "query failed", log.Err(err), log.BotID(botID))
		return nil, domain.ErrInternal
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			logPG("GetPrayerDay.iterateRows", "iteration failed", log.Err(err), log.BotID(botID))
			return nil, domain.ErrInternal
		}
		return nil, domain.ErrNotFound
	}

	prayerDay = &domain.PrayerDay{}
	if err := rows.Scan(
		&prayerDay.Date,
		&prayerDay.Fajr, &prayerDay.Shuruq,
		&prayerDay.Dhuhr, &prayerDay.Asr,
		&prayerDay.Maghrib, &prayerDay.Isha,
	); err != nil {
		logPG("GetPrayerDay.scanRow", "scan failed", log.Err(err), log.BotID(botID))
		return nil, domain.ErrInternal
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			logPG("GetPrayerDay.iterateRows", "iteration failed", log.Err(err), log.BotID(botID))
			return nil, domain.ErrInternal
		}
		return nil, domain.ErrNotFound
	}

	nextDay := &domain.PrayerDay{}
	if err := rows.Scan(
		&nextDay.Date,
		&nextDay.Fajr, &nextDay.Shuruq,
		&nextDay.Dhuhr, &nextDay.Asr,
		&nextDay.Maghrib, &nextDay.Isha,
	); err != nil {
		logPG("GetPrayerDay.scanNextRow", "scan failed", log.Err(err), log.BotID(botID))
		return nil, domain.ErrInternal
	}
	prayerDay.NextDay = nextDay

	if err := rows.Err(); err != nil {
		logPG("GetPrayerDay.iterateRows", "iteration failed", log.Err(err), log.BotID(botID))
		return nil, domain.ErrInternal
	}

	return prayerDay, nil
}

func (p *Postgres) GetStats(ctx context.Context, botID int64) (*domain.Stats, error) {
	stats := &domain.Stats{LanguagesGrouped: make(map[string]uint64)}

	err := p.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) AS users,
			COUNT(*) FILTER (WHERE COALESCE(subscribed, false)) AS subscribed,
			COUNT(*) FILTER (WHERE NOT COALESCE(subscribed, false)) AS unsubscribed
		FROM chats
		WHERE bot_id = $1
	`, botID).Scan(&stats.Users, &stats.Subscribed, &stats.Unsubscribed)
	if err != nil {
		logPG("GetStats", "query failed", log.Err(err), log.BotID(botID))
		return nil, domain.ErrInternal
	}

	rows, err := p.pool.Query(ctx, `
		SELECT language_code, COUNT(*) AS count
		FROM chats
		WHERE bot_id = $1
		GROUP BY language_code
	`, botID)
	if err != nil {
		logPG("GetStats.languages", "query failed", log.Err(err), log.BotID(botID))
		return nil, domain.ErrInternal
	}
	defer rows.Close()

	for rows.Next() {
		var (
			languageCode string
			count        uint64
		)
		if err := rows.Scan(&languageCode, &count); err != nil {
			logPG("GetStats.scanLanguage", "scan failed", log.Err(err), log.BotID(botID))
			return nil, domain.ErrInternal
		}
		stats.LanguagesGrouped[languageCode] = count
	}

	if err := rows.Err(); err != nil {
		logPG("GetStats.iterateLanguages", "iteration failed", log.Err(err), log.BotID(botID))
		return nil, domain.ErrInternal
	}

	return stats, nil
}

func (p *Postgres) UpdateReminder(
	ctx context.Context,
	botID int64,
	chatID int64,
	reminderType domain.ReminderType,
	messageID int,
	lastAt time.Time,
) error {
	return p.updateReminder(ctx, botID, chatID, func(reminder *domain.Reminder) {
		switch reminderType {
		case domain.ReminderTypeTomorrow:
			reminder.Tomorrow.MessageID = messageID
			reminder.Tomorrow.LastAt = lastAt
		case domain.ReminderTypeSoon:
			reminder.Soon.MessageID = messageID
			reminder.Soon.LastAt = lastAt
		case domain.ReminderTypeArrive:
			reminder.Arrive.MessageID = messageID
			reminder.Arrive.LastAt = lastAt
		}
	})
}

func (p *Postgres) DeleteChat(ctx context.Context, botID int64, chatID int64) error {
	_, err := p.pool.Exec(ctx, `
		DELETE FROM chats
		WHERE bot_id = $1 AND chat_id = $2
	`, botID, chatID)
	if err != nil {
		logPG("DeleteChat", "delete failed", log.Err(err), log.BotID(botID), log.ChatID(chatID))
		return domain.ErrInternal
	}

	return nil
}

func (p *Postgres) SetPrayerDays(ctx context.Context, botID int64, rows []*domain.PrayerDay) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const query = `
		INSERT INTO prayers (bot_id, prayer_date, fajr, shuruq, dhuhr, asr, maghrib, isha)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (bot_id, prayer_date) DO UPDATE SET
			fajr = EXCLUDED.fajr,
			shuruq = EXCLUDED.shuruq,
			dhuhr = EXCLUDED.dhuhr,
			asr = EXCLUDED.asr,
			maghrib = EXCLUDED.maghrib,
			isha = EXCLUDED.isha
	`

	for _, row := range rows {
		prayerDate := normalizeUTCDate(row.Date)
		_, err = tx.Exec(ctx, query,
			botID,
			prayerDate,
			row.Fajr,
			row.Shuruq,
			row.Dhuhr,
			row.Asr,
			row.Maghrib,
			row.Isha,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (p *Postgres) updateReminder(ctx context.Context, botID int64, chatID int64, mutate func(*domain.Reminder)) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		logPG("updateReminder.beginTx", "transaction start failed", log.Err(err), log.BotID(botID), log.ChatID(chatID))
		return domain.ErrInternal
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var reminderJSON []byte
	err = tx.QueryRow(ctx, `
		SELECT reminder
		FROM chats
		WHERE bot_id = $1 AND chat_id = $2
	`, botID, chatID).Scan(&reminderJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		logPG("updateReminder.select", "select failed", log.Err(err), log.BotID(botID), log.ChatID(chatID))
		return domain.ErrInternal
	}

	reminder, err := unmarshalReminder(reminderJSON, botID, chatID)
	if err != nil {
		return err
	}

	mutate(reminder)

	updatedReminderJSON, err := json.Marshal(reminder)
	if err != nil {
		logPG("updateReminder.marshal", "failed to encode reminder json", log.Err(err), log.BotID(botID), log.ChatID(chatID))
		return domain.ErrInternal
	}

	_, err = tx.Exec(ctx, `
		UPDATE chats
		SET reminder = $3::jsonb
		WHERE bot_id = $1 AND chat_id = $2
	`, botID, chatID, string(updatedReminderJSON))
	if err != nil {
		logPG("updateReminder.update", "update failed", log.Err(err), log.BotID(botID), log.ChatID(chatID))
		return domain.ErrInternal
	}

	if err := tx.Commit(ctx); err != nil {
		logPG("updateReminder.commit", "commit failed", log.Err(err), log.BotID(botID), log.ChatID(chatID))
		return domain.ErrInternal
	}

	return nil
}
