-- +goose Up
-- +goose ENVSUB ON
ALTER TABLE ${GLOBAL_DB_SCHEMA}.task_outbox
    ALTER COLUMN schedule_id DROP NOT NULL,
    ADD COLUMN endpoint TEXT NOT NULL DEFAULT '/tasks/send'
        CHECK (endpoint IN ('/tasks/send', '/tasks/delete')),
    ADD COLUMN run_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE TABLE ${GLOBAL_DB_SCHEMA}.notification_message_slots (
    chat_id BIGINT NOT NULL REFERENCES ${GLOBAL_DB_SCHEMA}.chats(telegram_chat_id) ON DELETE CASCADE,
    category TEXT NOT NULL CHECK (category IN ('prayer', 'tomorrow', 'weekly_fasting', 'weekly_kahf')),
    telegram_message_id BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (chat_id, category)
);

WITH delivered AS (
    SELECT
        s.chat_id,
        CASE
            WHEN r.kind IN ('before', 'at') THEN 'prayer'
            WHEN r.kind = 'tomorrow' THEN 'tomorrow'
            WHEN r.kind = 'weekly_fasting' THEN 'weekly_fasting'
            WHEN r.kind = 'weekly_kahf' THEN 'weekly_kahf'
        END AS category,
        d.telegram_message_id,
        d.updated_at
    FROM ${GLOBAL_DB_SCHEMA}.notification_deliveries d
    JOIN ${GLOBAL_DB_SCHEMA}.reminder_schedules s ON s.id = d.schedule_id
    JOIN ${GLOBAL_DB_SCHEMA}.reminder_rules r ON r.id = s.rule_id
    WHERE d.status = 'sent' AND d.telegram_message_id IS NOT NULL
),
latest AS (
    SELECT DISTINCT ON (chat_id, category)
        chat_id, category, telegram_message_id, updated_at
    FROM delivered
    WHERE category IS NOT NULL
    ORDER BY chat_id, category, updated_at DESC
)
INSERT INTO ${GLOBAL_DB_SCHEMA}.notification_message_slots
    (chat_id, category, telegram_message_id, updated_at)
SELECT chat_id, category, telegram_message_id, updated_at FROM latest;

-- +goose Down
DROP TABLE ${GLOBAL_DB_SCHEMA}.notification_message_slots;

DELETE FROM ${GLOBAL_DB_SCHEMA}.task_outbox WHERE schedule_id IS NULL;

ALTER TABLE ${GLOBAL_DB_SCHEMA}.task_outbox
    DROP COLUMN run_at,
    DROP COLUMN endpoint,
    ALTER COLUMN schedule_id SET NOT NULL;
-- +goose ENVSUB OFF
