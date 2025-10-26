-- +goose Up
CREATE TABLE prayers (
    bot_id Int64 NOT NULL,
    prayer_date Date NOT NULL,
    fajr Datetime,
    shuruq Datetime,
    dhuhr Datetime,
    asr Datetime,
    maghrib Datetime,
    isha Datetime,
    PRIMARY KEY (bot_id, prayer_date)
);

-- +goose Down
DROP TABLE IF EXISTS prayers;