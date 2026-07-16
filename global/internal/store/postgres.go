package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/escalopa/prayer-bot/global/internal/database"
	"github.com/escalopa/prayer-bot/global/internal/domain"
)

type Store struct {
	pool *schemaPool
}

type schemaPool struct {
	pool   *pgxpool.Pool
	schema string
}

type schemaTx struct {
	pgx.Tx
	schema string
}

func IsNotFound(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func Open(ctx context.Context, databaseURL, databaseSchema string) (*Store, error) {
	if err := database.ValidateSchema(databaseSchema); err != nil {
		return nil, fmt.Errorf("validate postgres schema: %w", err)
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Store{pool: &schemaPool{pool: pool, schema: databaseSchema}}, nil
}

func (p *schemaPool) Close() { p.pool.Close() }

func (p *schemaPool) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return p.pool.Exec(ctx, qualifySQL(query, p.schema), args...)
}

func (p *schemaPool) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return p.pool.Query(ctx, qualifySQL(query, p.schema), args...)
}

func (p *schemaPool) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return p.pool.QueryRow(ctx, qualifySQL(query, p.schema), args...)
}

func (p *schemaPool) Begin(ctx context.Context) (*schemaTx, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &schemaTx{Tx: tx, schema: p.schema}, nil
}

func (tx *schemaTx) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return tx.Tx.Exec(ctx, qualifySQL(query, tx.schema), args...)
}

func (tx *schemaTx) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return tx.Tx.Query(ctx, qualifySQL(query, tx.schema), args...)
}

func (tx *schemaTx) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return tx.Tx.QueryRow(ctx, qualifySQL(query, tx.schema), args...)
}

func qualifySQL(query, schema string) string {
	qualifiedSchema := pgx.Identifier{schema}.Sanitize()
	return strings.ReplaceAll(query, "global_bot.", qualifiedSchema+".")
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) AcquireUpdate(ctx context.Context, updateID int64) (bool, error) {
	var acquired int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO global_bot.processed_updates (update_id, status, lease_until)
		VALUES ($1, 'processing', now() + interval '2 minutes')
		ON CONFLICT (update_id) DO UPDATE SET
			status = 'processing', attempts = global_bot.processed_updates.attempts + 1,
			lease_until = now() + interval '2 minutes', updated_at = now(), last_error = ''
		WHERE global_bot.processed_updates.status = 'failed'
		   OR (global_bot.processed_updates.status = 'processing' AND global_bot.processed_updates.lease_until < now())
		RETURNING update_id`, updateID).Scan(&acquired)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) CompleteUpdate(ctx context.Context, updateID int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE global_bot.processed_updates
		SET status = 'completed', lease_until = NULL, updated_at = now() WHERE update_id = $1`, updateID)
	return err
}

func (s *Store) FailUpdate(ctx context.Context, updateID int64, cause error) error {
	_, err := s.pool.Exec(ctx, `UPDATE global_bot.processed_updates
		SET status = 'failed', lease_until = NULL, last_error = left($2, 500), updated_at = now()
		WHERE update_id = $1`, updateID, errorText(cause))
	return err
}

func (s *Store) UpsertChat(ctx context.Context, chat domain.Chat) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO global_bot.chats (telegram_chat_id, chat_type, language_code)
		VALUES ($1, $2, $3)
		ON CONFLICT (telegram_chat_id) DO UPDATE SET
			chat_type = excluded.chat_type, language_code = excluded.language_code,
			blocked_at = NULL, updated_at = now()`, chat.TelegramChatID, chat.Type, chat.LanguageCode)
	return err
}

func (s *Store) DeleteChat(ctx context.Context, chatID int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM global_bot.chats WHERE telegram_chat_id = $1`, chatID)
	return err
}

type Stats struct {
	Chats            int64
	Profiles         int64
	EnabledRules     int64
	PendingSchedules int64
}

func (s *Store) Stats(ctx context.Context) (Stats, error) {
	var stats Stats
	err := s.pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM global_bot.chats),
		(SELECT count(*) FROM global_bot.prayer_profiles),
		(SELECT count(*) FROM global_bot.reminder_rules WHERE enabled),
		(SELECT count(*) FROM global_bot.reminder_schedules WHERE state = 'pending')`).Scan(
		&stats.Chats, &stats.Profiles, &stats.EnabledRules, &stats.PendingSchedules)
	return stats, err
}

func (s *Store) Profile(ctx context.Context, chatID int64) (domain.PrayerProfile, error) {
	var profile domain.PrayerProfile
	var method, madhab, highLatitude string
	var adjustments []byte
	err := s.pool.QueryRow(ctx, `
		SELECT chat_id, latitude::float8, longitude::float8, timezone_id,
		       google_place_id, user_location_label, method, madhab,
		       high_latitude_rule, adjustments, version, updated_at
		FROM global_bot.prayer_profiles WHERE chat_id = $1`, chatID).Scan(
		&profile.ChatID, &profile.Latitude, &profile.Longitude, &profile.Timezone,
		&profile.PlaceID, &profile.LocationLabel, &method, &madhab,
		&highLatitude, &adjustments, &profile.Version, &profile.UpdatedAt,
	)
	if err != nil {
		return domain.PrayerProfile{}, err
	}
	profile.Method = domain.Method(method)
	profile.Madhab = domain.Madhab(madhab)
	profile.HighLatitudeRule = domain.HighLatitudeRule(highLatitude)
	if err := json.Unmarshal(adjustments, &profile.Adjustments); err != nil {
		return domain.PrayerProfile{}, fmt.Errorf("decode adjustments: %w", err)
	}
	return profile, nil
}

func (s *Store) UpsertProfile(ctx context.Context, profile domain.PrayerProfile) (domain.PrayerProfile, error) {
	if err := profile.Validate(); err != nil {
		return domain.PrayerProfile{}, err
	}
	adjustments, err := json.Marshal(profile.Adjustments)
	if err != nil {
		return domain.PrayerProfile{}, err
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO global_bot.prayer_profiles
			(chat_id, latitude, longitude, timezone_id, google_place_id, user_location_label,
			 method, madhab, high_latitude_rule, adjustments)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (chat_id) DO UPDATE SET
			latitude = excluded.latitude, longitude = excluded.longitude,
			timezone_id = excluded.timezone_id, google_place_id = excluded.google_place_id,
			user_location_label = excluded.user_location_label, method = excluded.method,
			madhab = excluded.madhab, high_latitude_rule = excluded.high_latitude_rule,
			adjustments = excluded.adjustments,
			version = global_bot.prayer_profiles.version + 1, updated_at = now()
		RETURNING version, updated_at`, profile.ChatID, profile.Latitude, profile.Longitude,
		profile.Timezone, profile.PlaceID, profile.LocationLabel, profile.Method,
		profile.Madhab, profile.HighLatitudeRule, adjustments).Scan(&profile.Version, &profile.UpdatedAt)
	if err != nil {
		return domain.PrayerProfile{}, err
	}
	return profile, nil
}

func (s *Store) EnableDefaultRules(ctx context.Context, chatID int64) error {
	prayers := []domain.Prayer{
		domain.PrayerFajr, domain.PrayerDhuhr, domain.PrayerAsr, domain.PrayerMaghrib, domain.PrayerIsha,
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	for _, prayer := range prayers {
		_, err = tx.Exec(ctx, `
			INSERT INTO global_bot.reminder_rules (chat_id, kind, prayer, enabled)
			VALUES ($1, 'at', $2, true)
			ON CONFLICT (chat_id, kind, prayer, offset_minutes) DO UPDATE SET enabled = true, updated_at = now()`,
			chatID, prayer)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) DisableRules(ctx context.Context, chatID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err = tx.Exec(ctx, `UPDATE global_bot.reminder_rules SET enabled = false, updated_at = now() WHERE chat_id = $1`, chatID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM global_bot.reminder_schedules WHERE chat_id = $1`, chatID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) EnabledRules(ctx context.Context, chatID int64) ([]domain.ReminderRule, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, chat_id, kind, prayer, offset_minutes, local_time, enabled
		FROM global_bot.reminder_rules WHERE chat_id = $1 AND enabled ORDER BY id`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []domain.ReminderRule
	for rows.Next() {
		var rule domain.ReminderRule
		if err := rows.Scan(&rule.ID, &rule.ChatID, &rule.Kind, &rule.Prayer,
			&rule.OffsetMinutes, &rule.LocalTime, &rule.Enabled); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (s *Store) Rule(ctx context.Context, ruleID int64) (domain.ReminderRule, error) {
	var rule domain.ReminderRule
	err := s.pool.QueryRow(ctx, `
		SELECT id, chat_id, kind, prayer, offset_minutes, local_time, enabled
		FROM global_bot.reminder_rules WHERE id = $1`, ruleID).Scan(
		&rule.ID, &rule.ChatID, &rule.Kind, &rule.Prayer,
		&rule.OffsetMinutes, &rule.LocalTime, &rule.Enabled)
	return rule, err
}

func (s *Store) UpsertSchedule(ctx context.Context, schedule domain.ReminderSchedule) (domain.ReminderSchedule, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO global_bot.reminder_schedules
			(rule_id, chat_id, profile_version, local_date, prayer_at, next_run_at, state)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending')
		ON CONFLICT (rule_id) DO UPDATE SET
			chat_id = excluded.chat_id, profile_version = excluded.profile_version,
			local_date = excluded.local_date, prayer_at = excluded.prayer_at,
			next_run_at = excluded.next_run_at, state = 'pending', updated_at = now()
		RETURNING id`, schedule.RuleID, schedule.ChatID, schedule.ProfileVersion,
		schedule.LocalDate, schedule.PrayerAt, schedule.NextRunAt).Scan(&schedule.ID)
	return schedule, err
}

func (s *Store) Schedule(ctx context.Context, scheduleID int64) (domain.ReminderSchedule, error) {
	var schedule domain.ReminderSchedule
	err := s.pool.QueryRow(ctx, `
		SELECT id, rule_id, chat_id, profile_version, local_date::text, prayer_at, next_run_at, state
		FROM global_bot.reminder_schedules WHERE id = $1`, scheduleID).Scan(
		&schedule.ID, &schedule.RuleID, &schedule.ChatID, &schedule.ProfileVersion,
		&schedule.LocalDate, &schedule.PrayerAt, &schedule.NextRunAt, &schedule.State)
	return schedule, err
}

type OutboxItem struct {
	ID          int64
	DeliveryKey string
	Payload     []byte
}

func (s *Store) ClaimDue(ctx context.Context, now time.Time, limit int) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	rows, err := tx.Query(ctx, `
		SELECT id, rule_id, chat_id, profile_version, local_date::text, prayer_at, next_run_at, state
		FROM global_bot.reminder_schedules
		WHERE state = 'pending' AND next_run_at <= $1
		ORDER BY next_run_at, id FOR UPDATE SKIP LOCKED LIMIT $2`, now, limit)
	if err != nil {
		return 0, err
	}
	var schedules []domain.ReminderSchedule
	for rows.Next() {
		var schedule domain.ReminderSchedule
		if err := rows.Scan(&schedule.ID, &schedule.RuleID, &schedule.ChatID, &schedule.ProfileVersion,
			&schedule.LocalDate, &schedule.PrayerAt, &schedule.NextRunAt, &schedule.State); err != nil {
			rows.Close()
			return 0, err
		}
		schedules = append(schedules, schedule)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	for _, schedule := range schedules {
		deliveryKey := fmt.Sprintf("schedule:%d:%d:v%d", schedule.ID, schedule.NextRunAt.Unix(), schedule.ProfileVersion)
		payload, err := json.Marshal(domain.DeliveryTask{
			DeliveryKey: deliveryKey, ScheduleID: schedule.ID, RuleID: schedule.RuleID,
			ChatID: schedule.ChatID, ProfileVersion: schedule.ProfileVersion, ScheduledFor: schedule.NextRunAt,
		})
		if err != nil {
			return 0, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO global_bot.task_outbox (schedule_id, delivery_key, payload)
			VALUES ($1, $2, $3) ON CONFLICT (delivery_key) DO NOTHING`, schedule.ID, deliveryKey, payload); err != nil {
			return 0, err
		}
		if _, err = tx.Exec(ctx, `UPDATE global_bot.reminder_schedules SET state = 'queued', updated_at = now() WHERE id = $1`, schedule.ID); err != nil {
			return 0, err
		}
	}
	return len(schedules), tx.Commit(ctx)
}

func (s *Store) PendingOutbox(ctx context.Context, limit int) ([]OutboxItem, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, delivery_key, payload
		FROM global_bot.task_outbox ORDER BY id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []OutboxItem
	for rows.Next() {
		var item OutboxItem
		if err := rows.Scan(&item.ID, &item.DeliveryKey, &item.Payload); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) MarkOutboxEnqueued(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM global_bot.task_outbox WHERE id = $1`, id)
	return err
}

func (s *Store) Cleanup(ctx context.Context, now time.Time, limit int) (int64, error) {
	updates, err := s.pool.Exec(ctx, `WITH doomed AS (
		SELECT update_id FROM global_bot.processed_updates
		WHERE status IN ('completed', 'failed') AND updated_at < $1 - interval '7 days'
		ORDER BY updated_at LIMIT $2
	) DELETE FROM global_bot.processed_updates p USING doomed d WHERE p.update_id = d.update_id`, now, limit)
	if err != nil {
		return 0, err
	}
	deliveries, err := s.pool.Exec(ctx, `WITH doomed AS (
		SELECT delivery_key FROM global_bot.notification_deliveries
		WHERE status IN ('sent', 'failed', 'stale') AND updated_at < $1 - interval '30 days'
		ORDER BY updated_at LIMIT $2
	) DELETE FROM global_bot.notification_deliveries n USING doomed d WHERE n.delivery_key = d.delivery_key`, now, limit)
	if err != nil {
		return updates.RowsAffected(), err
	}
	return updates.RowsAffected() + deliveries.RowsAffected(), nil
}

func (s *Store) AcquireDelivery(ctx context.Context, task domain.DeliveryTask) (bool, error) {
	var key string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO global_bot.notification_deliveries
			(delivery_key, schedule_id, status, lease_until)
		VALUES ($1, $2, 'processing', now() + interval '2 minutes')
		ON CONFLICT (delivery_key) DO UPDATE SET
			status = 'processing', attempts = global_bot.notification_deliveries.attempts + 1,
			lease_until = now() + interval '2 minutes', updated_at = now(), last_error = ''
		WHERE global_bot.notification_deliveries.status = 'failed'
		   OR (global_bot.notification_deliveries.status = 'processing'
		       AND global_bot.notification_deliveries.lease_until < now())
		RETURNING delivery_key`, task.DeliveryKey, task.ScheduleID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) CompleteDelivery(ctx context.Context, task domain.DeliveryTask, messageID int64, next domain.ReminderSchedule) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err = tx.Exec(ctx, `UPDATE global_bot.notification_deliveries
		SET status = 'sent', telegram_message_id = $2, lease_until = NULL, updated_at = now()
		WHERE delivery_key = $1`, task.DeliveryKey, messageID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE global_bot.reminder_schedules SET
		profile_version = $2, local_date = $3, prayer_at = $4, next_run_at = $5,
		state = 'pending', updated_at = now() WHERE id = $1`, task.ScheduleID,
		next.ProfileVersion, next.LocalDate, next.PrayerAt, next.NextRunAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) MarkDeliveryStale(ctx context.Context, deliveryKey string) error {
	_, err := s.pool.Exec(ctx, `UPDATE global_bot.notification_deliveries
		SET status = 'stale', lease_until = NULL, updated_at = now() WHERE delivery_key = $1`, deliveryKey)
	return err
}

func (s *Store) FailDelivery(ctx context.Context, deliveryKey string, cause error) error {
	_, err := s.pool.Exec(ctx, `UPDATE global_bot.notification_deliveries
		SET status = 'failed', lease_until = NULL, last_error = left($2, 500), updated_at = now()
		WHERE delivery_key = $1`, deliveryKey, errorText(cause))
	return err
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
