-- +goose Up
-- +goose ENVSUB ON
CREATE SCHEMA IF NOT EXISTS ${GLOBAL_DB_SCHEMA};

CREATE TABLE ${GLOBAL_DB_SCHEMA}.chats (
    telegram_chat_id BIGINT PRIMARY KEY,
    chat_type TEXT NOT NULL CHECK (chat_type IN ('private', 'group', 'supergroup')),
    language_code TEXT NOT NULL DEFAULT 'en',
    blocked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ${GLOBAL_DB_SCHEMA}.prayer_profiles (
    chat_id BIGINT PRIMARY KEY REFERENCES ${GLOBAL_DB_SCHEMA}.chats(telegram_chat_id) ON DELETE CASCADE,
    latitude NUMERIC(6, 3) NOT NULL CHECK (latitude BETWEEN -90 AND 90),
    longitude NUMERIC(7, 3) NOT NULL CHECK (longitude BETWEEN -180 AND 180),
    timezone_id TEXT NOT NULL,
    google_place_id TEXT NOT NULL DEFAULT '',
    user_location_label TEXT NOT NULL DEFAULT '',
    method TEXT NOT NULL,
    madhab TEXT NOT NULL CHECK (madhab IN ('shafii', 'hanafi')),
    high_latitude_rule TEXT NOT NULL CHECK (high_latitude_rule IN ('angle_based', 'middle_of_night', 'one_seventh')),
    adjustments JSONB NOT NULL DEFAULT '{}',
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ${GLOBAL_DB_SCHEMA}.reminder_rules (
    id BIGSERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL REFERENCES ${GLOBAL_DB_SCHEMA}.chats(telegram_chat_id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('before', 'at', 'tomorrow')),
    prayer TEXT NOT NULL CHECK (prayer IN ('fajr', 'sunrise', 'dhuhr', 'asr', 'maghrib', 'isha')),
    offset_minutes INTEGER NOT NULL DEFAULT 0 CHECK (offset_minutes BETWEEN 0 AND 180),
    local_time TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (chat_id, kind, prayer, offset_minutes)
);

CREATE TABLE ${GLOBAL_DB_SCHEMA}.reminder_schedules (
    id BIGSERIAL PRIMARY KEY,
    rule_id BIGINT NOT NULL UNIQUE REFERENCES ${GLOBAL_DB_SCHEMA}.reminder_rules(id) ON DELETE CASCADE,
    chat_id BIGINT NOT NULL REFERENCES ${GLOBAL_DB_SCHEMA}.chats(telegram_chat_id) ON DELETE CASCADE,
    profile_version BIGINT NOT NULL,
    local_date DATE NOT NULL,
    prayer_at TIMESTAMPTZ NOT NULL,
    next_run_at TIMESTAMPTZ NOT NULL,
    state TEXT NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'queued', 'processing')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX reminder_schedules_due_idx
    ON ${GLOBAL_DB_SCHEMA}.reminder_schedules (next_run_at, id)
    WHERE state = 'pending';

CREATE TABLE ${GLOBAL_DB_SCHEMA}.notification_deliveries (
    delivery_key TEXT PRIMARY KEY,
    schedule_id BIGINT NOT NULL REFERENCES ${GLOBAL_DB_SCHEMA}.reminder_schedules(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('processing', 'sent', 'failed', 'stale')),
    attempts INTEGER NOT NULL DEFAULT 1,
    lease_until TIMESTAMPTZ,
    telegram_message_id BIGINT,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ${GLOBAL_DB_SCHEMA}.task_outbox (
    id BIGSERIAL PRIMARY KEY,
    schedule_id BIGINT NOT NULL REFERENCES ${GLOBAL_DB_SCHEMA}.reminder_schedules(id) ON DELETE CASCADE,
    delivery_key TEXT NOT NULL UNIQUE,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX task_outbox_pending_idx
    ON ${GLOBAL_DB_SCHEMA}.task_outbox (id);

CREATE TABLE ${GLOBAL_DB_SCHEMA}.processed_updates (
    update_id BIGINT PRIMARY KEY,
    status TEXT NOT NULL CHECK (status IN ('processing', 'completed', 'failed')),
    attempts INTEGER NOT NULL DEFAULT 1,
    lease_until TIMESTAMPTZ,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX processed_updates_retention_idx
    ON ${GLOBAL_DB_SCHEMA}.processed_updates (updated_at)
    WHERE status IN ('completed', 'failed');

CREATE INDEX notification_deliveries_retention_idx
    ON ${GLOBAL_DB_SCHEMA}.notification_deliveries (updated_at)
    WHERE status IN ('sent', 'failed', 'stale');

-- +goose Down
DROP SCHEMA IF EXISTS ${GLOBAL_DB_SCHEMA} CASCADE;
-- +goose ENVSUB OFF
