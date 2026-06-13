-- +goose Up
CREATE TABLE prayers (
    bot_id BIGINT NOT NULL,
    prayer_date DATE NOT NULL,
    fajr TIMESTAMPTZ,
    shuruq TIMESTAMPTZ,
    dhuhr TIMESTAMPTZ,
    asr TIMESTAMPTZ,
    maghrib TIMESTAMPTZ,
    isha TIMESTAMPTZ,
    PRIMARY KEY (bot_id, prayer_date)
);

-- +goose Down
DROP TABLE IF EXISTS prayers;
