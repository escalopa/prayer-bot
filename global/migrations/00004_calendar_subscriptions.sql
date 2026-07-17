-- +goose Up
-- +goose ENVSUB ON
CREATE TABLE ${GLOBAL_DB_SCHEMA}.calendar_subscriptions (
    chat_id BIGINT PRIMARY KEY
        REFERENCES ${GLOBAL_DB_SCHEMA}.chats(telegram_chat_id) ON DELETE CASCADE,
    feed_token TEXT NOT NULL UNIQUE
        CHECK (length(feed_token) = 64 AND feed_token !~ '[^0-9a-f]'),
    uid_namespace TEXT NOT NULL UNIQUE
        CHECK (length(uid_namespace) = 32 AND uid_namespace !~ '[^0-9a-f]'),
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE ${GLOBAL_DB_SCHEMA}.calendar_subscriptions;
-- +goose ENVSUB OFF
