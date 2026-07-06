# `internal/db`

The persistence layer. All SQL lives here; the rest of the codebase talks to it
through the `Store` type and small per-module interfaces (so handlers can be
tested against fakes). Backed by **PostgreSQL** (Supabase) via
[`jackc/pgx/v5`](https://github.com/jackc/pgx).

## Files

| File | Responsibility |
|------|----------------|
| [`store.go`](store.go) | `Store` — the public façade. `Open(ctx)` builds it from config; every method delegates to `Postgres`. |
| [`postgres.go`](postgres.go) | All queries and the `pgxpool` connection. Chat CRUD, subscriber lookups, prayer-day read/write, stats, and the reminder read-modify-write transaction. |
| [`config.go`](config.go) | `LoadConfig()` — reads `DATABASE_URL`, falling back to `SUPABASE_DB_URL`. |
| [`config_test.go`](config_test.go) | Tests for the URL precedence rule. |

## Data model

Two tables (see [`migrations`](../../migrations/README.md)), both **multi-tenant**
via a composite primary key that includes `bot_id`:

- `chats (bot_id, chat_id)` — per-chat language, state, subscription, and the
  `reminder` **JSONB** blob.
- `prayers (bot_id, prayer_date)` — the six daily prayer timestamps.

## Notable design points

- **Errors are translated** to `domain` sentinels (`ErrNotFound`,
  `ErrAlreadyExists`, `ErrInternal`, `ErrUnmarshalJSON`) so callers stay
  driver-agnostic. Unique-violation detection is in `isUniqueViolation`.
- **Reminder updates are transactional** (`updateReminder`): read the JSONB,
  apply a `mutate(*domain.Reminder)` closure, write it back — all inside one
  `BEGIN`/`COMMIT`, with a deferred rollback safety net.
- **`GetPrayerDay` fetches two rows** (the requested date and the next day) in a
  single query and links them via `PrayerDay.NextDay`, powering "next prayer"
  and reminder roll-over past midnight.
- **Simple query protocol** — the pool uses `QueryExecModeSimpleProtocol`, which
  is required by the Supabase transaction pooler (port 6543).
- **Dates are normalized to UTC midnight** (`normalizeUTCDate`) before hitting
  the `prayers` table, keeping the `DATE` key stable regardless of caller TZ.

## How modules consume it

Each function exposes a tiny `service.NewDB(ctx)` that returns `*db.Store`, and
its handler declares a `DB interface { … }` listing only the methods it uses.
`*db.Store` satisfies all of them. This keeps the real implementation in one
place while letting handlers be unit-tested against mocks.
