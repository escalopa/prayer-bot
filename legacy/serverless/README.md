# `serverless`

The three GCP Cloud Functions (2nd gen) that make up the running system. Each is
an **independent Go module** with its own `go.mod` and a `replace` directive
pointing at the repo root, so it can be zipped and deployed on its own while
still sharing `domain`, `config`, `log`, and `internal/db`.

```
                   Telegram
                      │ webhook (POST)
                      ▼
              ┌───────────────┐
              │  dispatcher   │  HTTP function — handles user commands & callbacks
              └───────┬───────┘
                      │
      Cloud Scheduler │ (every minute)
                      ▼
              ┌───────────────┐        ┌───────────────┐
              │   reminder    │        │  PostgreSQL   │
              │ HTTP function │◄──────►│  (Supabase)   │
              └───────────────┘        └───────▲───────┘
                                               │
             GCS object created (CSV)          │
                      │ Eventarc               │
                      ▼                         │
              ┌───────────────┐                 │
              │    loader     │─────────────────┘
              │ CloudEvent fn │  parses schedule CSV → prayers table
              └───────────────┘
```

## The three functions

| Function | Trigger | Job | Docs |
|----------|---------|-----|------|
| **dispatcher** | HTTP (Telegram webhook) | Handle commands, inline keyboards, admin actions, per-chat state. | [README](dispatcher/README.md) |
| **reminder** | HTTP (Cloud Scheduler, ~1/min) | Decide which subscribers are due a reminder and send it. | [README](reminder/README.md) |
| **loader** | CloudEvent (GCS finalize) | Parse an uploaded prayer-schedule CSV and upsert it into `prayers`. | [README](loader/README.md) |

## Shared conventions

- **Lazy init with `sync.Once`.** Each function builds its handler (config + DB +
  Telegram client) once per warm instance and reuses it across invocations.
- **`Handle(...)` entrypoint.** Each handler's core logic lives in a `Handle`
  method; the function file is a thin GCP adapter around it.
- **`internal/handler`** holds the business logic; **`internal/service`** holds
  the infra adapters (DB, storage) as thin wrappers over the shared packages.
- **Localization** ships as embedded YAML under each handler's `languages/`
  directory (`//go:embed`), so no runtime file access is needed.
- **Multi-bot.** Every request/tick is scoped to a `bot_id`; one deployment
  serves all configured bots.
