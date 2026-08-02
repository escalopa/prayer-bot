-- +goose Up
-- +goose ENVSUB ON
-- Group chats can opt into receiving the pre-prayer reminder as a
-- non-anonymous "who is joining the jamaa'ah" poll instead of a plain message.
-- The flag lives on the chat because it changes delivery presentation, not
-- scheduling; private chats never use it.
ALTER TABLE ${GLOBAL_DB_SCHEMA}.chats
    ADD COLUMN jamaat_poll BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE ${GLOBAL_DB_SCHEMA}.chats
    DROP COLUMN jamaat_poll;
-- +goose ENVSUB OFF
