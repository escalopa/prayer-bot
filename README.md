# prayer-bot 🙏

A serverless Telegram bot that provides Muslim prayer times and sends notifications when prayers are approaching.

The global bot's solar equations, supported calculation methods, high-latitude rules, and Gregorian--Hijri conversion are documented on the public [calculation methodology site](https://escalopa.github.io/prayer-bot/). The versioned [LaTeX source](docs/calculation-methods.tex) is maintained in this repository.

[![wakatime](https://wakatime.com/badge/user/965e81db-2a88-4564-b236-537c4a901130/project/635dffc4-6a06-4e43-9a87-5bb977437cdb.svg)](https://wakatime.com/badge/user/965e81db-2a88-4564-b236-537c4a901130/project/635dffc4-6a06-4e43-9a87-5bb977437cdb)
[![Report card](https://goreportcard.com/badge/github.com/escalopa/gopray)](https://goreportcard.com/report/github.com/escalopa/gopray)

## Currently Available Cities

| City      | Bot                                                        |
|-----------|------------------------------------------------------------|
| Kazan     | [@kazan_prayer_bot](https://t.me/kazan_prayer_bot)         |
| Innopolis | [@innopolis_prayer_bot](https://t.me/innopolis_prayer_bot) |

---

## Architecture 🏗️

The project is a set of **stateless GCP Cloud Functions** written in Go, backed by
**Supabase Postgres** for state and **GCS** for prayer-schedule CSV files. A single
codebase serves **many bots** (one per city); everything is keyed by `bot_id` and the
per-bot config (token, owner, timezone) is loaded from the base64-encoded `APP_CONFIG`
environment variable.

### System overview

```mermaid
flowchart LR
  subgraph clients [External triggers]
    TG[Telegram users]
    SCH[Cloud Scheduler]
    UP[Prayer CSV upload]
  end

  subgraph functions [GCP Cloud Functions]
    D["dispatcher — HTTP webhook"]
    R["reminder — HTTP / cron"]
    L["loader — GCS CloudEvent"]
  end

  PG[(Supabase Postgres)]
  GCS[(GCS data bucket)]

  TG -->|webhook update| D
  D -->|replies| TG
  SCH -->|periodic POST| R
  R -->|reminders| TG
  UP --> GCS
  GCS -->|object finalized| L

  D --> PG
  R --> PG
  L --> PG
```

Each function is an independent Go module under [`serverless/`](serverless/) that depends on a
shared root module for cross-cutting code.

| Component | Trigger | Responsibility | Code |
|-----------|---------|----------------|------|
| **dispatcher** | Telegram webhook (HTTP `POST`) | Authenticates the request by secret header, resolves/creates the chat, and routes commands & inline callbacks | [`serverless/dispatcher`](serverless/dispatcher/) |
| **reminder** | Cloud Scheduler (HTTP `POST`) | For every bot, evaluates each subscriber against the reminder rules and sends due notifications | [`serverless/reminder`](serverless/reminder/) |
| **loader** | GCS object-finalized (CloudEvent) | Parses an uploaded `<bot_id>.csv` schedule and upserts it into Postgres | [`serverless/loader`](serverless/loader/) |
| **domain** | — | Shared models & value types (`Chat`, `PrayerDay`, `Reminder`, `Duration`, errors) | [`domain`](domain/) |
| **config** | — | Decodes `APP_CONFIG` into a per-bot config map | [`config`](config/) |
| **internal/db** | — | `pgx`-based Postgres repository shared by all functions | [`internal/db`](internal/db/) |
| **log** | — | Thin structured-logging wrapper over `log/slog` | [`log`](log/) |

Infrastructure (functions, scheduler, buckets, IAM) is defined as Terraform in
[`infra/gcp/`](infra/gcp/). Each directory has its own `README.md` with details.

### Repository layout

```text
.
├── domain/          # shared models & value types (root module)
├── config/          # APP_CONFIG loader
├── log/             # slog wrapper
├── internal/db/     # Postgres repository (pgx)
├── serverless/
│   ├── dispatcher/  # Telegram webhook handler   (own go.mod)
│   ├── reminder/    # scheduled reminder sender   (own go.mod)
│   └── loader/      # CSV schedule loader         (own go.mod)
├── migrations/      # Goose SQL migrations
├── infra/gcp/       # Terraform (Cloud Functions, Scheduler, GCS)
└── _scripts/        # local helper scripts
```

### Reminder flow

The reminder function is the heart of the system. On each tick it fans out over bots and,
within a bot, over subscribers (bounded to `maxConcurrentReminderSends`), evaluating three
independent reminder types. State is stored per chat in the `reminder` JSONB column so a
reminder is sent at most once, and stale reminders are skipped after downtime.

```mermaid
flowchart TD
  A[Scheduler POST] --> B[for each bot]
  B --> C[GetSubscribers]
  C --> D[GetChatsByIDs + GetPrayerDay today/next]
  D --> E["fan out per chat (≤ maxConcurrentReminderSends)"]
  E --> F{evaluate reminder types}
  F -->|Tomorrow| G[send next-day schedule]
  F -->|Soon| H["send upcoming prayer (or jamaat poll in groups)"]
  F -->|Arrive| I[send prayer-arrived notice]
  G --> J[UpdateReminder: message id + last_at]
  H --> J
  I --> J
```

### Data model

Two tables, both multi-tenant via a composite primary key that starts with `bot_id`.
The flexible reminder configuration is stored as JSONB on `chats`.

```mermaid
erDiagram
  chats {
    bigint      bot_id        PK
    bigint      chat_id       PK
    text        language_code
    text        state
    jsonb       reminder
    boolean     subscribed
    timestamptz subscribed_at
    timestamptz created_at
  }
  prayers {
    bigint      bot_id       PK
    date        prayer_date  PK
    timestamptz fajr
    timestamptz shuruq
    timestamptz dhuhr
    timestamptz asr
    timestamptz maghrib
    timestamptz isha
  }
```

`chats` and `prayers` are linked logically by `bot_id` (and by date at read time); there is
no foreign key, since the two tables are populated by different functions.

### Deployment

**GitHub secrets (per environment):**

| Secret | Purpose |
|--------|---------|
| `APP_CONFIG` | Bot config JSON |
| `GCP_PROJECT_ID` | GCP project ID |
| `GCP_SA_KEY` | GCP deploy service account JSON |
| `GCP_TFSTATE_BUCKET` | GCS bucket for Terraform state |
| `SUPABASE_DB_URL` | Supabase transaction pooler URL (port 6543) — runtime `DATABASE_URL` on functions |
| `SUPABASE_DB_DIRECT_URL` | Supabase direct Postgres URL (port 5432) — Goose schema migrations |

**Automatic deploys**

| Trigger | Environment | What runs |
|---------|-------------|-----------|
| Pull request → `main` | `dev` | lint → validate → plan → Goose migrate → Terraform apply → webhooks → profiles |
| Push / merge to `main` | `prod` | same full chain |

**Manual deploy (hotfixes):** Actions → *Deploy to GCP* → Run workflow → pick `dev` or `prod` and optionally a branch.

---

## Configuration 🛠️

Bot configuration is managed through environment variables.

Below is an example of an `APP_CONFIG` value containing all bot information:

```json
{
  "648252": {
    "bot_id": 648252,          // Bot ID
    "owner_id": 1385434843,    // Bot owner ID
    "location": "Europe/Moscow", // Timezone of the city
    "token": "oa7GmLW3fncbOE0MTfV0mKxH/F37cShhxgZ1mjl614w", // Telegram token
    "secret": "Noe&uPcwjaAxjqJU_JP4C^g2V7ZDQX" // Secret key to verify requests
  },
  ...
}
```

- To find your owner ID, use [ID bot](https://t.me/myidbot)
- Bot ID is the first number before `:` in the bot token
  - TOKEN: `123456789:ABC-DEF1234ghIkl-zyx57W2v1u123ew11`
  - Bot ID: `123456789`

---

## Bot Features 🤖

### User Commands 📝

| Command     | Description                             |
|-------------|-----------------------------------------|
| today       | Get today's prayer times                |
| date        | Get prayer times for a specific date    |
| next        | Find out the next prayer time           |
| subscribe   | Subscribe to daily reminders            |
| unsubscribe | Unsubscribe from daily reminders        |
| remind      | Set reminder offset for the next prayer |
| language    | Change the bot language                 |
| help        | Show help message                       |
| bug         | Report a problem to bot owner           |
| feedback    | Send feedback to bot owner              |

### Admin Commands 📝

| Command  | Description                           |
|----------|---------------------------------------|
| admin    | Show admin help message               |
| stats    | View bot usage statistics             |
| announce | Send message to all users             |
| reply    | Reply to user's bug/feedback message  |

---

## References 📚

- [go-telegram](https://github.com/go-telegram)
- [telegram-api](https://core.telegram.org/bots/api)

---

## How to Contribute 🤝

### [1] Add a City

You do:

1. Get prayer times for a city in CSV format
2. Make a pull request (or open an issue) with the new file

I do:

1. Create a new Telegram bot
2. Upload the city file to the GCS data bucket

### [2] Add a Language

You do:

1. Create translation text for the following files:
    - [./serverless/reminder/internal/handler/languages/text.yaml](./serverless/reminder/internal/handler/languages/text.yaml)
    - [./serverless/dispatcher/internal/handler/languages/en.yaml](./serverless/dispatcher/internal/handler/languages/en.yaml) (replace `en` with the new language code)

I do:

1. Deploy a new version of the code

### [3] Code Contributions

Found a bug? Want to add a new feature? Just open an issue or submit a pull request.

---

## Development Roadmap 🚀

### V1 ✅

- [x] Support date format for `/prayersdate` command with leading zeros and delimiters (. / -)
- [x] Implement subscriptions & notifications
- [x] Update text messages to be more user-friendly

### V2 ✅

- [x] Store prayer times in memory to reduce database requests
- [x] Add response endpoint for admin to address feedback & bug messages
- [x] Add Jumu'ah prayer reminders on Fridays

### V3 ✅

- [x] Add time keyboard to `/date` command
- [x] Remove selection message for `/date` & `/lang` after user interaction or timeout
- [x] Terminate other active channels when user sends new commands
- [x] Add feature to delete old prayer time message when a new one is sent
- [x] Enable admins to broadcast messages to all subscribers
- [x] Add feature to get subscriber count for admins
- [x] Write more robust tests for core features

### V4 ✅

- [x] Add multi-language support (AR, RU, TT, TR, UZ)
- [x] Implement script messages in the bot
- [x] Set user script before command if not set
- [x] Use script commands in notifications
- [x] Fix prayer timetables for other languages

### V5 ✅

- [x] Refactor code for better readability and maintainability
- [x] Enhance logging to be more informative
- [x] Enable using multiple bots with the same codebase

### V6 ✅

- [x] Migrate to serverless architecture
- [x] Automate deployment using Terraform
- [x] Add support for multiple cities
- [x] Add Spanish & French language support
- [x] Add `/stats` command for bot usage statistics

### V7 ✅

- [x] Add jamaat gathering feature for group chats

### V8 🔄

- [ ] Add support for all major world cities

---

## Visualization 🖥️

```bash
cd infra/gcp
terraform plan -out plan.out
terraform show -json plan.out > plan.json
docker run --rm -it -p 9000:9000 -v "$(pwd)/plan.json:/src/plan.json" im2nguyen/rover:latest -planJSONPath=plan.json
```
