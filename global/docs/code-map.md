# Code map

Use this map to locate the owner of a behavior before searching the repository.
All paths are relative to `global/`.

## Executables

| Path | Runtime | Responsibility |
| --- | --- | --- |
| `cmd/webhook` | Public Cloud Run service | Telegram webhook, commands, callbacks, feedback, owner dashboard, Mini App static files and APIs |
| `cmd/dispatch` | Private Cloud Run service called by Scheduler | Claims due reminder schedules, drains the transactional outbox into Cloud Tasks, runs retention cleanup |
| `cmd/send` | Private Cloud Run service called by Cloud Tasks | Sends reminder messages, advances recurring schedules, and deletes notification messages |
| `cmd/botprofile` | Deployment command | Synchronizes the webhook, stable public profile, command menu, Mini App menu button, and avatar |
| `cmd/bootstrapdb` | Deployment command | Creates only the selected global PostgreSQL schema before Goose runs |

The production image contains all executables. Terraform selects the executable
with the container command, so the three Cloud Run services use the same build.

## Internal packages

| Package | Owns | Depends on |
| --- | --- | --- |
| `internal/domain` | Shared value types for chats, profiles, prayers, reminders, schedules, and task payloads | Standard library only |
| `internal/config` | Environment parsing and validation for each executable | `internal/database` for allowed schemas |
| `internal/database` | Environment-schema names and schema validation | Standard library only |
| `internal/store` | All PostgreSQL queries and transaction boundaries | `domain`, pgx |
| `internal/prayertime` | Prayer calculation interface and `go-prayer` adapter | `domain` |
| `internal/hijri` | Umm al-Qura conversion and per-chat display correction | `go-hijri` |
| `internal/location` | Google Time Zone and reverse-geocoding integration | Google HTTP APIs |
| `internal/reminders` | Recurrence planning, due dispatch, Cloud Tasks enqueueing, Telegram delivery, and cleanup categories | `domain`, `store`, `prayertime`, Telegram and GCP clients |
| `internal/telegram` | Bot commands, callbacks, keyboards, update routing, feedback, and owner dashboard | `store`, `location`, `prayertime`, `reminders`, `i18n` |
| `internal/miniapp` | Embedded web UI, signed init-data authentication, settings APIs, Qibla/bootstrap data, and calendar downloads | `store`, `location`, `prayertime`, `reminders`, `qibla`, `calendarfile`, `i18n` |
| `internal/i18n` | All supported locales, messages, buttons, prayer names, method names, and dates | `domain` |
| `internal/qibla` | Great-circle bearing and distance to the Kaaba | Standard library only |
| `internal/calendarfile` | Localized RFC 5545 prayer-calendar generation | `domain`, `i18n`, `prayertime` |
| `internal/botprofile` | Read-before-write Telegram profile synchronization and rate-limit handling | Telegram Bot API |
| `internal/assets` | Embedded bot avatar and welcome media | Go embed |
| `internal/httpx` | Shared HTTP response helpers | Standard library only |

## Persistence and infrastructure

| Path | Responsibility |
| --- | --- |
| `migrations/` | Versioned schema changes for both global environments |
| `infra/gcp/` | Cloud Run, Cloud Tasks, Scheduler, service accounts, IAM, Secret Manager, Artifact Registry, and Maps key |
| `.github/workflows/global-ci.yaml` | Global Go tests, image build, and Terraform validation |
| `.github/workflows/global-deploy.yaml` | Manual testing/production build, migration, Terraform apply, and Telegram profile synchronization |

## Change routing

| Desired change | Primary files | Documents to update |
| --- | --- | --- |
| Add a command or button | `internal/telegram`, `internal/i18n` | [Request flows](request-flows.md) if the flow is new |
| Add a Mini App setting | `internal/miniapp`, `internal/store`, possibly migrations | [Request flows](request-flows.md), [Data model](data-model.md) |
| Add a calculation method | `internal/domain`, `internal/prayertime`, `internal/i18n` | Public calculation methodology and [Architecture](architecture.md) |
| Change reminder timing | `internal/reminders/planner.go`, `internal/store` | [Reminder delivery](reminder-delivery.md) |
| Change retry or deletion behavior | `internal/reminders/sender.go`, `internal/store`, `infra/gcp` | [Reminder delivery](reminder-delivery.md), [Operations](operations.md) |
| Add persistent state | `migrations`, `internal/store`, `internal/domain` | [Data model](data-model.md) |
| Add a service or cloud dependency | `infra/gcp`, `internal/config`, relevant `cmd` | [Architecture](architecture.md), [Runtime and deployment](runtime-and-deployment.md) |
