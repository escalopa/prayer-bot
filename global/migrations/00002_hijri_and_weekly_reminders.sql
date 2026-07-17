-- +goose Up
-- +goose ENVSUB ON
ALTER TABLE ${GLOBAL_DB_SCHEMA}.prayer_profiles
    ADD COLUMN hijri_adjustment SMALLINT NOT NULL DEFAULT 0
    CHECK (hijri_adjustment BETWEEN -2 AND 2);

ALTER TABLE ${GLOBAL_DB_SCHEMA}.reminder_rules
    DROP CONSTRAINT reminder_rules_kind_check;

ALTER TABLE ${GLOBAL_DB_SCHEMA}.reminder_rules
    ADD CONSTRAINT reminder_rules_kind_check
    CHECK (kind IN ('before', 'at', 'tomorrow', 'weekly_fasting', 'weekly_kahf'));

-- +goose Down
DELETE FROM ${GLOBAL_DB_SCHEMA}.reminder_rules
WHERE kind IN ('weekly_fasting', 'weekly_kahf');

ALTER TABLE ${GLOBAL_DB_SCHEMA}.reminder_rules
    DROP CONSTRAINT reminder_rules_kind_check;

ALTER TABLE ${GLOBAL_DB_SCHEMA}.reminder_rules
    ADD CONSTRAINT reminder_rules_kind_check
    CHECK (kind IN ('before', 'at', 'tomorrow'));

ALTER TABLE ${GLOBAL_DB_SCHEMA}.prayer_profiles
    DROP COLUMN hijri_adjustment;
-- +goose ENVSUB OFF
