# CLAUDE.md

Guidance for working in this repository. Read this first, then the doc it points you to for the area you're changing.

## What this repo is

Two independent prayer-time Telegram bots share one Git repository but **not** one runtime:

1. **Legacy city bots** (repo root: `serverless/`, `domain/`, `config/`, `internal/db/`, `log/`, `migrations/`, `infra/gcp/`). One codebase, one bot per city, keyed by `bot_id`. GCP Cloud Functions + Supabase Postgres (`public.chats`, `public.prayers`) + GCS CSV schedules. Documented in the root [`README.md`](README.md).
2. **Global worldwide bot** ([`global/`](global/)). A **separate Go module** with its own container image, Cloud Run services, Terraform state, Telegram token, Secret Manager entries, and Postgres schemas. **Nothing under `global/` imports or changes the legacy runtime, and vice-versa.**

**Active work is on the global bot.** Unless a task is explicitly about the legacy city bots, you are working in `global/`.

## Before you touch the global bot

- **Read [`global/docs/README.md`](global/docs/README.md) first** — it indexes every engineering doc. Then read the doc that owns the area you're changing (see routing table below). Architectural or persistence changes must update the owning doc **in the same PR**.
- **Graphify (per [`AGENTS.md`](AGENTS.md)):** read [`graphify-out/GRAPH_REPORT.md`](graphify-out/GRAPH_REPORT.md) before broad source reading or grep. For "how does X relate to Y" use `graphify query "…"` / `graphify path "A" "B"` / `graphify explain "…"`. After changing code, run `graphify update .` (AST-only, no API cost).

### Doc routing — pick the owner before searching

| Doc | Owns |
| --- | --- |
| [`global/docs/architecture.md`](global/docs/architecture.md) | System boundary, trust relationships, Mini App, data ownership, calculation profile, delivery overview, rollout gates |
| [`global/docs/code-map.md`](global/docs/code-map.md) | Which package/executable owns a behavior; **change-routing table** for common tasks |
| [`global/docs/data-model.md`](global/docs/data-model.md) | Tables, invariants, retention, migration rules |
| [`global/docs/request-flows.md`](global/docs/request-flows.md) | Webhook, location, Mini App session, schedule display, occasions, Qibla/calendar, feedback |
| [`global/docs/reminder-delivery.md`](global/docs/reminder-delivery.md) | **Source of truth** for planning, dispatch, retries, idempotency, cleanup categories |
| [`global/docs/runtime-and-deployment.md`](global/docs/runtime-and-deployment.md) | Env isolation, runtime topology, secrets, **DB connection rules**, deploy workflow |
| [`global/docs/operations.md`](global/docs/operations.md) | Health/logs, incident triage table, secret rotation, profile sync, recovery |

## Global bot layout

Three Cloud Run services from **one immutable image**, selected by container command:

- `cmd/webhook` — public. Telegram commands/callbacks, feedback, owner dashboard, and the embedded Mini App (static files + `/api/miniapp/*`) served at `/app/`.
- `cmd/dispatch` — Scheduler-only. Claims due schedules, drains the outbox into Cloud Tasks, runs retention cleanup.
- `cmd/send` — Cloud Tasks-only. Idempotent Telegram delivery, recurrence advance, message cleanup.
- `cmd/botprofile`, `cmd/bootstrapdb` — deploy-time commands (profile/webhook sync; schema creation).

Internal packages (largest UI surfaces first): `internal/telegram` (bot UI), `internal/miniapp` (web UI + signed init-data auth + calendar feed), `internal/store` (all SQL), `internal/i18n` (8 locales: en, ar, es, fr, ru, tr, uz, tt), `internal/reminders` (planner/dispatch/sender), plus `domain`, `prayertime`, `hijri`, `occasions`, `location`, `qibla`, `calendarfile`, `botprofile`, `config`, `database`, `assets`, `httpx`. See [`code-map.md`](global/docs/code-map.md) for ownership and dependencies.

## Commands (run inside `global/`)

```sh
make test     # go test ./...
make check    # gofmt -w cmd internal && go vet ./... && go test ./...
make build    # go build ./cmd/...
```

Run `make check` before finishing a change. The legacy root module has its own `Makefile`.

## Non-obvious constraints (don't relearn these the hard way)

- **pgx + Supabase transaction pooler:** runtime connections MUST use `pgx.QueryExecModeExec` (no named prepared-statement cache), or you get `42P05`/`26000`. In that mode, **JSONB params must be passed as JSON text, not `[]byte`** (else `22P02`) — use the shared JSON-text encoder in `internal/store`.
- **Schema isolation:** all global SQL goes through `internal/store`, which qualifies the logical `global_bot` schema with the env schema (`global_bot_testing` / `global_bot_production`). Never touch legacy `public.*`. Goose must always target `-table="${GLOBAL_DB_SCHEMA}.goose_db_version"`; bootstrap the schema before the first migration.
- **Delivery is at-least-once** in a narrow window (Telegram send succeeds, completion commit fails). The sender must make a compensating `deleteMessages` call after a post-send failure before returning a retryable error. Cleanup uses one message slot per category (`prayer`, `tomorrow`, `weekly_fasting`, `weekly_kahf`, `islamic_occasion`); every message also gets a 36h cleanup task (Telegram can't delete messages older than 48h).
- **Profile version** increments on any change affecting calculated times; queued tasks carry it and go **stale** instead of sending after a change.
- **Mini App auth:** never trust a Telegram user ID from a JSON body. Identity comes only from the signed `initData` header (HMAC verified, dedup fields, <24h). Private-chat scoped; group config stays in the bot with admin authorization.
- **Privacy:** coordinates rounded to 3 decimals on persist; reverse-geocoded city is not stored (only Place ID); feedback is never stored in Postgres. Never log tokens, secrets, DB URLs, coordinates, Maps keys, full Telegram updates, or the calendar bearer URL.
- **Hijri correction** (-2..+2 days) shifts displayed Hijri dates and occasion matching only — never prayer instants.
- **Calculation engine** is hidden behind `prayertime.Calculator` (currently `github.com/hablullah/go-prayer`); keep that boundary so engines stay swappable.

## Deployment

Manual **Deploy global prayer bot** workflow, env `testing` (GitHub `dev` secrets, schema `global_bot_testing`) or `production` (GitHub `prod`, `global_bot_production`). Only three global secrets are new: `GLOBAL_BOT_TOKEN`, `GLOBAL_WEBHOOK_SECRET`, `GLOBAL_OWNER_ID`. Migrations run before the new image; schema changes must be backward-compatible with the previously running revision. Never interchange testing/production state prefixes or tokens.
