-- +goose Up
-- +goose ENVSUB ON
-- Persisted country code lets the Mini App default the Zakat calculator to the
-- user's local currency without an extra geocoding request. It is written from
-- the country already resolved on every location change, so it does not affect
-- calculated prayer times and rides on the existing profile version bump.
ALTER TABLE ${GLOBAL_DB_SCHEMA}.prayer_profiles
    ADD COLUMN country_code TEXT NOT NULL DEFAULT '';

-- A single shared row caches the daily gold/silver spot price (USD per troy
-- ounce) and the USD-based currency rate table used to localize niSab. The
-- CHECK (id = 1) constraint keeps it a singleton.
CREATE TABLE ${GLOBAL_DB_SCHEMA}.metal_prices (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    gold_usd_per_ounce NUMERIC NOT NULL CHECK (gold_usd_per_ounce > 0),
    silver_usd_per_ounce NUMERIC NOT NULL CHECK (silver_usd_per_ounce > 0),
    rates JSONB NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE ${GLOBAL_DB_SCHEMA}.metal_prices;

ALTER TABLE ${GLOBAL_DB_SCHEMA}.prayer_profiles
    DROP COLUMN country_code;
-- +goose ENVSUB OFF
