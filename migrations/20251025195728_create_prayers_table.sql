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

-- +goose StatementBegin
CREATE INDEX idx_prayers_bot_id ON prayers (bot_id);
CREATE INDEX idx_prayers_date ON prayers (prayer_date);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS prayers;