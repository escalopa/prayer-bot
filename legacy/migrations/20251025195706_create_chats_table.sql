-- +goose Up
CREATE TABLE chats (
    bot_id BIGINT NOT NULL,
    chat_id BIGINT NOT NULL,
    language_code TEXT,
    state TEXT,
    reminder JSONB,
    subscribed BOOLEAN,
    subscribed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ,
    PRIMARY KEY (bot_id, chat_id)
);

-- +goose Down
DROP TABLE IF EXISTS chats;
