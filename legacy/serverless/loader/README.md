# loader

The **CloudEvent Cloud Function triggered when a CSV is uploaded to the data
bucket** (GCS object-finalize via Eventarc). It parses a bot's prayer schedule
and upserts it into the `prayers` table.

Module: `github.com/escalopa/prayer-bot/loader`.

## Flow

1. [`loader.go`](loader.go) — the GCP entrypoint (`LoaderCloudEvent`). Decodes the
   GCS event to get `{bucket, name}`, lazily builds the handler, and calls
   `handler.Handle(ctx, bucket, key)`.
2. [`internal/handler/handler.go`](internal/handler/handler.go) — `Handle`:
   - Ignores non-`.csv` objects.
   - Derives the **`bot_id` from the filename** (e.g. `123456789.csv` →
     `123456789`) via `extractBotID`.
   - Fetches the object from GCS, parses it in the bot's timezone, and calls
     `SetPrayerDays` (a single upserting transaction).

## CSV format ([`parser.go`](internal/handler/parser.go))

Seven columns, no assumptions about a header beyond a fixed field count:

```
date, fajr, shuruq, dhuhr, asr, maghrib, isha
2/1/2006, 15:04, 15:04, 15:04, 15:04, 15:04, 15:04
```

- Date format: `d/m/yyyy`; clock format: `HH:MM`.
- Each clock time is combined with the row's date **in the bot's configured
  timezone**, then stored as `TIMESTAMPTZ`.
- Malformed rows produce a wrapped parse error identifying the bot.

A sample input generator lives in [`_scripts/city`](../../_scripts/README.md).

## Files

| File | Responsibility |
|------|----------------|
| [`handler.go`](internal/handler/handler.go) | `Handle`, filename→bot_id, orchestration. |
| [`parser.go`](internal/handler/parser.go) | CSV → `[]*domain.PrayerDay`. |
| [`log.go`](internal/handler/log.go) | Namespaced logging helpers. |
| [`internal/service/db.go`](internal/service/db.go) | `NewDB` wrapper over the shared store. |
| [`internal/service/storage.go`](internal/service/storage.go) | GCS reader (`Get(bucket, key)`), satisfying the handler's `Storage` interface. |

## Upsert semantics

`SetPrayerDays` (in [`internal/db/postgres.go`](../../internal/db/postgres.go))
runs one transaction with `INSERT … ON CONFLICT (bot_id, prayer_date) DO UPDATE`,
so re-uploading a corrected schedule overwrites the affected days in place.
