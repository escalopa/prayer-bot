# `migrations`

SQL schema migrations managed by [Goose](https://github.com/pressly/goose). Files
are named `<timestamp>_<description>.sql` and contain `-- +goose Up` /
`-- +goose Down` sections.

## Schema

### `chats` — one row per Telegram chat, per bot
| Column | Type | Notes |
|--------|------|-------|
| `bot_id`, `chat_id` | `BIGINT` | Composite **primary key** (multi-tenant). |
| `language_code` | `TEXT` | UI language. |
| `state` | `TEXT` | Current conversational step (see dispatcher `state.go`). |
| `reminder` | `JSONB` | Serialized [`domain.Reminder`](../domain/reminder.go) (offsets, last-sent timestamps, Jamaat config). |
| `subscribed`, `subscribed_at` | `BOOLEAN` / `TIMESTAMPTZ` | Reminder opt-in. |
| `created_at` | `TIMESTAMPTZ` | |

### `prayers` — one row per day, per bot
| Column | Type | Notes |
|--------|------|-------|
| `bot_id`, `prayer_date` | `BIGINT` / `DATE` | Composite **primary key**. |
| `fajr … isha` | `TIMESTAMPTZ` | The six daily prayer times. |

## Running

Migrations run in CI (the **Migrate** stage of
[`deploy.yaml`](../.github/workflows/deploy.yaml)) against the Supabase **direct**
connection (`SUPABASE_DB_DIRECT_URL`, port 5432) — not the pooler used at runtime.

Locally:

```bash
goose -dir migrations postgres "$SUPABASE_DB_DIRECT_URL" up
```

## Adding a migration

```bash
goose -dir migrations create add_something sql
```

Keep the `Down` section reversible, and remember both tables are keyed by
`bot_id` first to preserve multi-bot isolation.
