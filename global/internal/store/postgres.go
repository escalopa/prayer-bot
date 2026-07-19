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
	poolConfig, err := runtimePoolConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Store{pool: &schemaPool{pool: pool, schema: databaseSchema}}, nil
}

func runtimePoolConfig(databaseURL string) (*pgxpool.Config, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	// Supabase's transaction pooler can execute consecutive queries on
	// different PostgreSQL connections. Named prepared statements are scoped
	// to one connection, so pgx's default statement cache produces 42P05
	// ("already exists") and 26000 ("does not exist") errors through the
	// pooler. Exec mode uses the extended protocol without named statements.
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	return config, nil
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

// marshalJSONText deliberately returns string rather than []byte. In pgx
// QueryExecModeExec, []byte is encoded as bytea before PostgreSQL resolves the
// target JSONB column, producing SQLSTATE 22P02 through transaction poolers.
func marshalJSONText(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
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
			chat_type = excluded.chat_type,
			blocked_at = NULL, updated_at = now()`, chat.TelegramChatID, chat.Type, chat.LanguageCode)
	return err
}

func (s *Store) Chat(ctx context.Context, chatID int64) (domain.Chat, error) {
	var chat domain.Chat
	err := s.pool.QueryRow(ctx, `
		SELECT telegram_chat_id, chat_type, language_code, blocked_at
		FROM global_bot.chats WHERE telegram_chat_id = $1`, chatID).Scan(
		&chat.TelegramChatID, &chat.Type, &chat.LanguageCode, &chat.BlockedAt,
	)
	return chat, err
}

func (s *Store) SetLanguage(ctx context.Context, chatID int64, languageCode string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE global_bot.chats SET language_code = $2, updated_at = now()
		WHERE telegram_chat_id = $1`, chatID, languageCode)
	return err
}

func (s *Store) DeleteChat(ctx context.Context, chatID int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM global_bot.chats WHERE telegram_chat_id = $1`, chatID)
	return err
}

type MetricCount struct {
	Key   string
	Count int64
}

type AdminDashboard struct {
	Users                   int64
	Groups                  int64
	ConfiguredUsers         int64
	NewUsers24Hours         int64
	NewUsers7Days           int64
	NewUsers30Days          int64
	ActiveUsers24Hours      int64
	ActiveUsers7Days        int64
	ActiveUsers30Days       int64
	ReminderUsers           int64
	EnabledRules            int64
	PendingSchedules        int64
	QueuedTasks             int64
	SentDeliveries24Hours   int64
	FailedDeliveries24Hours int64
	StaleDeliveries24Hours  int64
	ProcessingDeliveries    int64
	FailedUpdates24Hours    int64
	Languages               []MetricCount
	Methods                 []MetricCount
	ReminderKinds           []MetricCount
}

func (s *Store) AdminMetrics(ctx context.Context) (AdminDashboard, error) {
	var dashboard AdminDashboard
	err := s.pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM global_bot.chats WHERE chat_type = 'private'),
		(SELECT count(*) FROM global_bot.chats WHERE chat_type IN ('group', 'supergroup')),
		(SELECT count(*) FROM global_bot.prayer_profiles p
			JOIN global_bot.chats c ON c.telegram_chat_id = p.chat_id
			WHERE c.chat_type = 'private'),
		(SELECT count(*) FROM global_bot.chats WHERE chat_type = 'private' AND created_at >= now() - interval '24 hours'),
		(SELECT count(*) FROM global_bot.chats WHERE chat_type = 'private' AND created_at >= now() - interval '7 days'),
		(SELECT count(*) FROM global_bot.chats WHERE chat_type = 'private' AND created_at >= now() - interval '30 days'),
		(SELECT count(*) FROM global_bot.chats WHERE chat_type = 'private' AND updated_at >= now() - interval '24 hours'),
		(SELECT count(*) FROM global_bot.chats WHERE chat_type = 'private' AND updated_at >= now() - interval '7 days'),
		(SELECT count(*) FROM global_bot.chats WHERE chat_type = 'private' AND updated_at >= now() - interval '30 days'),
		(SELECT count(DISTINCT r.chat_id) FROM global_bot.reminder_rules r
			JOIN global_bot.chats c ON c.telegram_chat_id = r.chat_id
			WHERE r.enabled AND c.chat_type = 'private'),
		(SELECT count(*) FROM global_bot.reminder_rules WHERE enabled),
		(SELECT count(*) FROM global_bot.reminder_schedules WHERE state = 'pending'),
		(SELECT count(*) FROM global_bot.task_outbox),
		(SELECT count(*) FROM global_bot.notification_deliveries
			WHERE status = 'sent' AND updated_at >= now() - interval '24 hours'),
		(SELECT count(*) FROM global_bot.notification_deliveries
			WHERE status = 'failed' AND updated_at >= now() - interval '24 hours'),
		(SELECT count(*) FROM global_bot.notification_deliveries
			WHERE status = 'stale' AND updated_at >= now() - interval '24 hours'),
		(SELECT count(*) FROM global_bot.notification_deliveries WHERE status = 'processing'),
		(SELECT count(*) FROM global_bot.processed_updates
			WHERE status = 'failed' AND updated_at >= now() - interval '24 hours')`).Scan(
		&dashboard.Users,
		&dashboard.Groups,
		&dashboard.ConfiguredUsers,
		&dashboard.NewUsers24Hours,
		&dashboard.NewUsers7Days,
		&dashboard.NewUsers30Days,
		&dashboard.ActiveUsers24Hours,
		&dashboard.ActiveUsers7Days,
		&dashboard.ActiveUsers30Days,
		&dashboard.ReminderUsers,
		&dashboard.EnabledRules,
		&dashboard.PendingSchedules,
		&dashboard.QueuedTasks,
		&dashboard.SentDeliveries24Hours,
		&dashboard.FailedDeliveries24Hours,
		&dashboard.StaleDeliveries24Hours,
		&dashboard.ProcessingDeliveries,
		&dashboard.FailedUpdates24Hours,
	)
	if err != nil {
		return AdminDashboard{}, err
	}
	if dashboard.Languages, err = s.metricCounts(ctx, `
		SELECT language_code, count(*)
		FROM global_bot.chats
		WHERE chat_type = 'private'
		GROUP BY language_code
		ORDER BY count(*) DESC, language_code`); err != nil {
		return AdminDashboard{}, err
	}
	if dashboard.Methods, err = s.metricCounts(ctx, `
		SELECT p.method, count(*)
		FROM global_bot.prayer_profiles p
		JOIN global_bot.chats c ON c.telegram_chat_id = p.chat_id
		WHERE c.chat_type = 'private'
		GROUP BY p.method
		ORDER BY count(*) DESC, p.method`); err != nil {
		return AdminDashboard{}, err
	}
	dashboard.ReminderKinds, err = s.metricCounts(ctx, `
		SELECT category, count(DISTINCT chat_id)
		FROM (
			SELECT r.chat_id, CASE
				WHEN kind IN ('before', 'at', 'tomorrow') THEN 'prayer'
				WHEN kind = 'weekly_fasting' THEN 'fasting'
				WHEN kind = 'weekly_kahf' THEN 'kahf'
				WHEN kind = 'occasion_major' THEN 'occasion_major'
				WHEN kind = 'occasion_fasting' THEN 'occasion_fasting'
				WHEN kind = 'occasion_observed' THEN 'occasion_observed'
			END AS category
			FROM global_bot.reminder_rules r
			JOIN global_bot.chats c ON c.telegram_chat_id = r.chat_id
			WHERE r.enabled AND c.chat_type = 'private'
		) enabled_rules
		WHERE category IS NOT NULL
		GROUP BY category
		ORDER BY count(DISTINCT chat_id) DESC, category`)
	if err != nil {
		return AdminDashboard{}, err
	}
	return dashboard, nil
}

func (s *Store) metricCounts(ctx context.Context, query string) ([]MetricCount, error) {
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var metrics []MetricCount
	for rows.Next() {
		var metric MetricCount
		if err := rows.Scan(&metric.Key, &metric.Count); err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}
	return metrics, rows.Err()
}

func (s *Store) Profile(ctx context.Context, chatID int64) (domain.PrayerProfile, error) {
	var profile domain.PrayerProfile
	var method, madhab, highLatitude string
	var adjustments []byte
	err := s.pool.QueryRow(ctx, `
		SELECT chat_id, latitude::float8, longitude::float8, timezone_id,
		       google_place_id, user_location_label, method, madhab,
		       high_latitude_rule, adjustments, hijri_adjustment, version, updated_at
		FROM global_bot.prayer_profiles WHERE chat_id = $1`, chatID).Scan(
		&profile.ChatID, &profile.Latitude, &profile.Longitude, &profile.Timezone,
		&profile.PlaceID, &profile.LocationLabel, &method, &madhab,
		&highLatitude, &adjustments, &profile.HijriAdjustment, &profile.Version, &profile.UpdatedAt,
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
	adjustments, err := marshalJSONText(profile.Adjustments)
	if err != nil {
		return domain.PrayerProfile{}, err
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO global_bot.prayer_profiles
			(chat_id, latitude, longitude, timezone_id, google_place_id, user_location_label,
			 method, madhab, high_latitude_rule, adjustments, hijri_adjustment)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (chat_id) DO UPDATE SET
			latitude = excluded.latitude, longitude = excluded.longitude,
			timezone_id = excluded.timezone_id, google_place_id = excluded.google_place_id,
			user_location_label = excluded.user_location_label, method = excluded.method,
			madhab = excluded.madhab, high_latitude_rule = excluded.high_latitude_rule,
			adjustments = excluded.adjustments, hijri_adjustment = excluded.hijri_adjustment,
			version = global_bot.prayer_profiles.version + 1, updated_at = now()
		RETURNING version, updated_at`, profile.ChatID, profile.Latitude, profile.Longitude,
		profile.Timezone, profile.PlaceID, profile.LocationLabel, profile.Method,
		profile.Madhab, profile.HighLatitudeRule, adjustments, profile.HijriAdjustment).Scan(&profile.Version, &profile.UpdatedAt)
	if err != nil {
		return domain.PrayerProfile{}, err
	}
	return profile, nil
}

func (s *Store) EnableDefaultRules(ctx context.Context, chatID int64) error {
	return s.ConfigurePrayerRules(ctx, chatID, true, 0)
}

func (s *Store) ConfigurePrayerRules(ctx context.Context, chatID int64, enabled bool, beforeMinutes int) error {
	if beforeMinutes < 0 || beforeMinutes > 180 {
		return fmt.Errorf("pre-prayer reminder must be between 0 and 180 minutes")
	}
	prayers := []domain.Prayer{
		domain.PrayerFajr, domain.PrayerDhuhr, domain.PrayerAsr, domain.PrayerMaghrib, domain.PrayerIsha,
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err = tx.Exec(ctx, `DELETE FROM global_bot.reminder_schedules s
		USING global_bot.reminder_rules r
		WHERE s.rule_id = r.id AND r.chat_id = $1 AND r.kind IN ('before', 'at', 'tomorrow')`, chatID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE global_bot.reminder_rules
		SET enabled = false, updated_at = now()
		WHERE chat_id = $1 AND kind IN ('before', 'at', 'tomorrow')`, chatID); err != nil {
		return err
	}
	if !enabled {
		return tx.Commit(ctx)
	}
	for _, prayer := range prayers {
		_, err = tx.Exec(ctx, `
			INSERT INTO global_bot.reminder_rules (chat_id, kind, prayer, enabled)
			VALUES ($1, 'at', $2, true)
			ON CONFLICT (chat_id, kind, prayer, offset_minutes) DO UPDATE SET enabled = true, updated_at = now()`,
			chatID, prayer)
		if err != nil {
			return err
		}
		if beforeMinutes > 0 {
			_, err = tx.Exec(ctx, `
				INSERT INTO global_bot.reminder_rules (chat_id, kind, prayer, offset_minutes, enabled)
				VALUES ($1, 'before', $2, $3, true)
				ON CONFLICT (chat_id, kind, prayer, offset_minutes) DO UPDATE SET enabled = true, updated_at = now()`,
				chatID, prayer, beforeMinutes)
			if err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) DisableRules(ctx context.Context, chatID int64) error {
	return s.ConfigurePrayerRules(ctx, chatID, false, 0)
}

func (s *Store) SetWeeklyRule(ctx context.Context, chatID int64, kind domain.ReminderKind, enabled bool) error {
	if !kind.Weekly() {
		return fmt.Errorf("unsupported weekly reminder kind %q", kind)
	}
	localTime := "20:00"
	if kind == domain.ReminderWeeklyKahf {
		localTime = "09:00"
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if enabled {
		if _, err = tx.Exec(ctx, `
			INSERT INTO global_bot.reminder_rules (chat_id, kind, prayer, local_time, enabled)
			VALUES ($1, $2, 'fajr', $3, true)
			ON CONFLICT (chat_id, kind, prayer, offset_minutes) DO UPDATE SET
				local_time = excluded.local_time, enabled = true, updated_at = now()`,
			chatID, kind, localTime); err != nil {
			return err
		}
	} else {
		if _, err = tx.Exec(ctx, `UPDATE global_bot.reminder_rules SET enabled = false, updated_at = now()
			WHERE chat_id = $1 AND kind = $2`, chatID, kind); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM global_bot.reminder_schedules s
			USING global_bot.reminder_rules r
			WHERE s.rule_id = r.id AND r.chat_id = $1 AND r.kind = $2`, chatID, kind); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) SetOccasionRule(ctx context.Context, chatID int64, kind domain.ReminderKind, enabled bool) error {
	if !kind.Occasion() {
		return fmt.Errorf("unsupported occasion reminder kind %q", kind)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if enabled {
		if _, err = tx.Exec(ctx, `
			INSERT INTO global_bot.reminder_rules (chat_id, kind, prayer, local_time, enabled)
			VALUES ($1, $2, 'fajr', '20:00', true)
			ON CONFLICT (chat_id, kind, prayer, offset_minutes) DO UPDATE SET
				local_time = excluded.local_time, enabled = true, updated_at = now()`,
			chatID, kind); err != nil {
			return err
		}
	} else {
		if _, err = tx.Exec(ctx, `UPDATE global_bot.reminder_rules SET enabled = false, updated_at = now()
			WHERE chat_id = $1 AND kind = $2`, chatID, kind); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM global_bot.reminder_schedules s
			USING global_bot.reminder_rules r
			WHERE s.rule_id = r.id AND r.chat_id = $1 AND r.kind = $2`, chatID, kind); err != nil {
			return err
		}
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
	Endpoint    string
	RunAt       time.Time
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
		payload, err := marshalJSONText(domain.DeliveryTask{
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
	rows, err := s.pool.Query(ctx, `SELECT id, delivery_key, endpoint, run_at, payload
		FROM global_bot.task_outbox ORDER BY id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []OutboxItem
	for rows.Next() {
		var item OutboxItem
		if err := rows.Scan(&item.ID, &item.DeliveryKey, &item.Endpoint, &item.RunAt, &item.Payload); err != nil {
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

func (s *Store) CompleteDelivery(
	ctx context.Context,
	task domain.DeliveryTask,
	messageID int64,
	next domain.ReminderSchedule,
	category string,
	expiresAt time.Time,
) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err = tx.Exec(ctx, `UPDATE global_bot.notification_deliveries
		SET status = 'sent', telegram_message_id = $2, lease_until = NULL, updated_at = now()
		WHERE delivery_key = $1`, task.DeliveryKey, messageID); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(ctx, `UPDATE global_bot.reminder_schedules SET
		profile_version = $2, local_date = $3, prayer_at = $4, next_run_at = $5,
		state = 'pending', updated_at = now() WHERE id = $1`, task.ScheduleID,
		next.ProfileVersion, next.LocalDate, next.PrayerAt, next.NextRunAt); err != nil {
		return 0, err
	}
	var previousMessageID int64
	err = tx.QueryRow(ctx, `SELECT telegram_message_id
		FROM global_bot.notification_message_slots
		WHERE chat_id = $1 AND category = $2 FOR UPDATE`, task.ChatID, category).Scan(&previousMessageID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO global_bot.notification_message_slots
		(chat_id, category, telegram_message_id) VALUES ($1, $2, $3)
		ON CONFLICT (chat_id, category) DO UPDATE SET
			telegram_message_id = excluded.telegram_message_id, updated_at = now()`,
		task.ChatID, category, messageID); err != nil {
		return 0, err
	}
	if previousMessageID != 0 && previousMessageID != messageID {
		if err := enqueueMessageDeletion(
			ctx, tx, task.ChatID, previousMessageID, time.Now(), fmt.Sprintf("replaced-by:%d", messageID),
		); err != nil {
			return 0, err
		}
	}
	if err := enqueueMessageDeletion(ctx, tx, task.ChatID, messageID, expiresAt, "expiry"); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return previousMessageID, nil
}

func enqueueMessageDeletion(
	ctx context.Context,
	tx *schemaTx,
	chatID, messageID int64,
	runAt time.Time,
	reason string,
) error {
	key := fmt.Sprintf("delete:%d:%d:%s", chatID, messageID, reason)
	payload, err := marshalJSONText(domain.MessageDeletionTask{
		DeletionKey: key,
		ChatID:      chatID,
		MessageID:   messageID,
	})
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO global_bot.task_outbox
		(schedule_id, delivery_key, endpoint, run_at, payload)
		VALUES (NULL, $1, '/tasks/delete', $2, $3)
		ON CONFLICT (delivery_key) DO NOTHING`,
		key, runAt, payload)
	return err
}

func (s *Store) ClearNotificationMessage(ctx context.Context, chatID, messageID int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM global_bot.notification_message_slots
		WHERE chat_id = $1 AND telegram_message_id = $2`, chatID, messageID)
	return err
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

func (s *Store) CalendarSubscription(ctx context.Context, chatID int64) (domain.CalendarSubscription, error) {
	var subscription domain.CalendarSubscription
	err := s.pool.QueryRow(ctx, `SELECT chat_id, feed_token, uid_namespace, enabled
		FROM global_bot.calendar_subscriptions WHERE chat_id = $1`, chatID).Scan(
		&subscription.ChatID,
		&subscription.FeedToken,
		&subscription.UIDNamespace,
		&subscription.Enabled,
	)
	return subscription, err
}

func (s *Store) CalendarSubscriptionByToken(
	ctx context.Context,
	feedToken string,
) (domain.CalendarSubscription, error) {
	var subscription domain.CalendarSubscription
	err := s.pool.QueryRow(ctx, `SELECT chat_id, feed_token, uid_namespace, enabled
		FROM global_bot.calendar_subscriptions WHERE feed_token = $1`, feedToken).Scan(
		&subscription.ChatID,
		&subscription.FeedToken,
		&subscription.UIDNamespace,
		&subscription.Enabled,
	)
	return subscription, err
}

func (s *Store) EnableCalendarSubscription(
	ctx context.Context,
	chatID int64,
	feedToken string,
	uidNamespace string,
) (domain.CalendarSubscription, error) {
	var subscription domain.CalendarSubscription
	err := s.pool.QueryRow(ctx, `
		INSERT INTO global_bot.calendar_subscriptions AS current_subscription
			(chat_id, feed_token, uid_namespace, enabled)
		VALUES ($1, $2, $3, true)
		ON CONFLICT (chat_id) DO UPDATE SET
			feed_token = CASE
				WHEN current_subscription.enabled
				THEN current_subscription.feed_token
				ELSE excluded.feed_token
			END,
			enabled = true,
			updated_at = now()
		RETURNING chat_id, feed_token, uid_namespace, enabled`,
		chatID, feedToken, uidNamespace,
	).Scan(
		&subscription.ChatID,
		&subscription.FeedToken,
		&subscription.UIDNamespace,
		&subscription.Enabled,
	)
	return subscription, err
}

func (s *Store) DisableCalendarSubscription(ctx context.Context, chatID int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE global_bot.calendar_subscriptions
		SET enabled = false, updated_at = now()
		WHERE chat_id = $1 AND enabled`, chatID)
	return err
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
