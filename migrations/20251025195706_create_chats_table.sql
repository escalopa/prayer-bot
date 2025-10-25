-- +goose Up
CREATE TABLE chats (
    chat_id Int64 NOT NULL,
    bot_id Int64 NOT NULL,
    language_code Utf8,
    state Utf8,
    reminder Json,
    is_group Bool,
    subscribed Bool,
    subscribed_at Datetime,
    created_at Datetime,
    PRIMARY KEY (bot_id, chat_id)
);

-- +goose StatementBegin
CREATE INDEX idx_chats_bot_id ON chats (bot_id);
CREATE INDEX idx_chats_chat_id ON chats (chat_id);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS chats;