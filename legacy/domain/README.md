# `domain`

Core domain types shared by every module (dispatcher, reminder, loader, and the
`internal/db` layer). This package has **no dependencies on infrastructure** —
no database, no Telegram, no GCP — so it can be imported from anywhere and unit
tested in isolation.

Module: `github.com/escalopa/prayer-bot` (the repository root module).

## Files

| File | What it holds |
|------|---------------|
| [`chat.go`](chat.go) | `Chat` (a Telegram chat's stored state: language, subscription, reminder config) and `Stats` (aggregate usage counters). |
| [`prayer.go`](prayer.go) | `PrayerID` enum (`Fajr`…`Isha`) with `String()`/`ParsePrayerID`, the `PrayerDay` schedule struct (six prayer times + linked `NextDay`), and helpers `FormatDuration`, `DateUTC`. |
| [`reminder.go`](reminder.go) | `Reminder` and its three `ReminderConfig` slots (`Tomorrow`, `Soon`, `Arrive`) plus the `Jamaat` (congregation) config. `ReminderType` enum and the per-prayer Jamaat delay getters/setters. |
| [`duration.go`](duration.go) | `Duration`, a `time.Duration` wrapper that (un)marshals to a human string like `"20m"` so JSONB reminder configs stay readable. |
| [`config.go`](config.go) | `BotConfig` (per-bot id/owner/token/secret/timezone) and the custom `location` type that unmarshals an IANA timezone name into a `*time.Location`. Also the shared sentinel errors (`ErrNotFound`, `ErrAlreadyExists`, `ErrInternal`, …). |
| [`markdown.go`](markdown.go) | `StripMarkdown` — removes Telegram MarkdownV2 escaping for plain-text contexts (e.g. poll questions). |

## Key concepts

- **`PrayerDay.NextDay`** — a prayer day carries a pointer to the following day.
  This lets "next prayer" and reminder logic roll past midnight (e.g. the next
  Fajr after tonight's Isha) without a second DB round-trip.
- **Sentinel errors** — the DB layer maps low-level failures to these values so
  callers can branch with `errors.Is` instead of inspecting driver errors.
- **`Duration` JSON form** — reminder offsets are persisted inside the `chats.reminder`
  JSONB column; the string encoding keeps stored configs diff-friendly.

## Tests

- [`prayer_test.go`](prayer_test.go) — `FormatDuration`, `PrayerID` round-tripping.
- [`duration_test.go`](duration_test.go) — `Duration` JSON marshal/unmarshal, including error cases.
- [`markdown_test.go`](markdown_test.go) — `StripMarkdown`.
