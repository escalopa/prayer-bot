-- +goose Up
-- +goose ENVSUB ON
ALTER TABLE ${GLOBAL_DB_SCHEMA}.reminder_rules
    DROP CONSTRAINT reminder_rules_kind_check;

ALTER TABLE ${GLOBAL_DB_SCHEMA}.reminder_rules
    ADD CONSTRAINT reminder_rules_kind_check
    CHECK (kind IN (
        'before', 'at', 'tomorrow', 'weekly_fasting', 'weekly_kahf',
        'occasion_major', 'occasion_fasting', 'occasion_observed'
    ));

ALTER TABLE ${GLOBAL_DB_SCHEMA}.notification_message_slots
    DROP CONSTRAINT notification_message_slots_category_check;

ALTER TABLE ${GLOBAL_DB_SCHEMA}.notification_message_slots
    ADD CONSTRAINT notification_message_slots_category_check
    CHECK (category IN (
        'prayer', 'tomorrow', 'weekly_fasting', 'weekly_kahf',
        'islamic_occasion'
    ));

-- +goose Down
DELETE FROM ${GLOBAL_DB_SCHEMA}.reminder_rules
WHERE kind IN ('occasion_major', 'occasion_fasting', 'occasion_observed');

DELETE FROM ${GLOBAL_DB_SCHEMA}.notification_message_slots
WHERE category = 'islamic_occasion';

ALTER TABLE ${GLOBAL_DB_SCHEMA}.notification_message_slots
    DROP CONSTRAINT notification_message_slots_category_check;

ALTER TABLE ${GLOBAL_DB_SCHEMA}.notification_message_slots
    ADD CONSTRAINT notification_message_slots_category_check
    CHECK (category IN ('prayer', 'tomorrow', 'weekly_fasting', 'weekly_kahf'));

ALTER TABLE ${GLOBAL_DB_SCHEMA}.reminder_rules
    DROP CONSTRAINT reminder_rules_kind_check;

ALTER TABLE ${GLOBAL_DB_SCHEMA}.reminder_rules
    ADD CONSTRAINT reminder_rules_kind_check
    CHECK (kind IN ('before', 'at', 'tomorrow', 'weekly_fasting', 'weekly_kahf'));
-- +goose ENVSUB OFF
