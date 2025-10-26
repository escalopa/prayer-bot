-- +goose Up
CREATE TABLE chats (
    chat_id Int64 NOT NULL,
    bot_id Int64 NOT NULL,
    language_code Utf8,
    state Utf8,
    reminder Json,
    subscribed Bool,
    subscribed_at Datetime,
    created_at Datetime,
    PRIMARY KEY (bot_id, chat_id)
);

-- +goose Down
DROP TABLE IF EXISTS chats;
