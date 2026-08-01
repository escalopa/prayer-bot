-- +goose Up
-- +goose ENVSUB ON
-- Ayyam al-Bid (white days) fasting reminders: one rule kind that reminds at
-- 20:00 on the evening before each of the 13th, 14th, and 15th Hijri days.
-- Delivered messages reuse the existing weekly_fasting cleanup slot, so no
-- message-slot category change is needed.
ALTER TABLE ${GLOBAL_DB_SCHEMA}.reminder_rules
    DROP CONSTRAINT reminder_rules_kind_check;

ALTER TABLE ${GLOBAL_DB_SCHEMA}.reminder_rules
    ADD CONSTRAINT reminder_rules_kind_check
    CHECK (kind IN (
        'before', 'at', 'tomorrow', 'weekly_fasting', 'weekly_kahf',
        'occasion_major', 'occasion_fasting', 'occasion_observed',
        'white_days'
    ));

-- +goose Down
DELETE FROM ${GLOBAL_DB_SCHEMA}.reminder_schedules s
USING ${GLOBAL_DB_SCHEMA}.reminder_rules r
WHERE s.rule_id = r.id AND r.kind = 'white_days';

DELETE FROM ${GLOBAL_DB_SCHEMA}.reminder_rules
WHERE kind = 'white_days';

ALTER TABLE ${GLOBAL_DB_SCHEMA}.reminder_rules
    DROP CONSTRAINT reminder_rules_kind_check;

ALTER TABLE ${GLOBAL_DB_SCHEMA}.reminder_rules
    ADD CONSTRAINT reminder_rules_kind_check
    CHECK (kind IN (
        'before', 'at', 'tomorrow', 'weekly_fasting', 'weekly_kahf',
        'occasion_major', 'occasion_fasting', 'occasion_observed'
    ));
-- +goose ENVSUB OFF
