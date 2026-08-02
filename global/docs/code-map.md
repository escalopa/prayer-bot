# Code map

Use this map to locate the owner of a behavior before searching the repository.
All paths are relative to `global/`.

## Hexagonal layout

The module follows ports-and-adapters. Dependencies always point inward:

| Layer | Path | Rule |
| --- | --- | --- |
| Domain | `internal/domain` | Entities, DTOs, pure policy (currency/method mapping, `ErrNotFound`). Standard library only. |
| Ports | `internal/port` | All shared interfaces (`Store`, `Calculator`, `LocationResolver`, `MetalSource`), typed on domain only. |
| Core | `internal/core/*` | Pure calculation and application services (prayer times, Hijri, occasions, qibla, calendar files, i18n, reminder planning/dispatch/delivery). Never imports an adapter. |
| Driving adapters | `internal/adapter/in/*` | Telegram bot and Mini App. Call the core and ports; own their narrow role interfaces where useful. |
| Driven adapters | `internal/adapter/out/*` | PostgreSQL, Google Maps, metal-price APIs, Telegram profile sync. Implement the ports (compile-time `var _ port.X` assertions) and translate technology errors to `domain.ErrNotFound` at the boundary. |
| Platform | `internal/{config,database,httpx,assets}` | Configuration, schema names, HTTP plumbing, embedded media. |
| Composition roots | `cmd/*` | The only places that construct concrete adapters and wire them into ports. |

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
| `internal/domain` | Shared value types (DTOs) for chats, profiles, prayers, reminders, schedules, task payloads, resolved locations, dashboard metrics; pure policy (recommended method, currency mapping); `ErrNotFound` | Standard library only |
| `internal/port` | Shared port interfaces: `Store`, `Calculator`, `LocationResolver`, `MetalSource` | `domain` only |
| `internal/adapter/out/metals` | Key-less gold/silver spot price and USD FX fetcher for the Zakat niSab | `domain`, HTTP APIs |
| `internal/config` | Environment parsing and validation for each executable | `internal/database` for allowed schemas |
| `internal/database` | Environment-schema names and schema validation | Standard library only |
| `internal/adapter/out/store` | All PostgreSQL queries and transaction boundaries | `domain`, pgx |
| `internal/core/prayertime` | Prayer calculation interface and `go-prayer` adapter | `domain` |
| `internal/core/hijri` | Umm al-Qura conversion and per-chat display correction | `go-hijri` |
| `internal/core/occasions` | Curated Hijri occasion definitions, corrected Gregorian matching, category filtering, and recurrence lookup | `hijri` |
| `internal/adapter/out/location` | Google Time Zone and reverse-geocoding integration | Google HTTP APIs |
| `internal/core/reminders` | Recurrence planning, due dispatch, Cloud Tasks enqueueing, Telegram delivery, and cleanup categories | `domain`, `store`, `prayertime`, Telegram and GCP clients |
| `internal/adapter/in/telegram` | Bot commands, callbacks, keyboards, update routing, feedback, and owner dashboard | `store`, `location`, `prayertime`, `reminders`, `i18n` |
| `internal/adapter/in/miniapp` | Embedded web UI, signed init-data authentication, settings APIs, Qibla/bootstrap data, and private calendar subscriptions | `store`, `location`, `prayertime`, `reminders`, `qibla`, `calendarfile`, `i18n` |
| `internal/core/i18n` | All supported locales, messages, buttons, prayer names, method names, and dates | `domain` |
| `internal/core/qibla` | Great-circle bearing and distance to the Kaaba | Standard library only |
| `internal/core/calendarfile` | Localized RFC 5545 prayer and Islamic-occasion calendar generation | `domain`, `i18n`, `prayertime`, `occasions` |
| `internal/adapter/out/botprofile` | Read-before-write Telegram profile synchronization and rate-limit handling | Telegram Bot API |
| `internal/assets` | Embedded bot avatar and welcome media | Go embed |
| `internal/httpx` | Shared HTTP response helpers | Standard library only |

## Persistence and infrastructure

| Path | Responsibility |
| --- | --- |
| `migrations/` | Versioned schema changes for both global environments |
| `infra/gcp/` | Cloud Run, Cloud Tasks, Scheduler, service accounts, IAM, Secret Manager, Artifact Registry, and Maps key |
| `internal/adapter/out/store/integration_test.go` | Env-gated (`TEST_DATABASE_URL`) real-SQL tests; see [Testing](testing.md) |
| `.github/workflows/global-ci.yaml` | Global Go tests, image build, and Terraform validation |
| `.github/workflows/global-deploy.yaml` | Manual testing/production build, migration, Terraform apply, and Telegram profile synchronization |

## Change routing

| Desired change | Primary files | Documents to update |
| --- | --- | --- |
| Add a command or button | `internal/adapter/in/telegram`, `internal/core/i18n` | [Request flows](request-flows.md) if the flow is new |
| Add a Mini App setting | `internal/adapter/in/miniapp`, `internal/adapter/out/store`, possibly migrations | [Request flows](request-flows.md), [Data model](data-model.md) |
| Add a calculation method | `internal/domain`, `internal/core/prayertime`, `internal/core/i18n` | Public calculation methodology and [Architecture](architecture.md) |
| Change reminder timing | `internal/core/reminders/planner.go`, `internal/adapter/out/store` | [Reminder delivery](reminder-delivery.md) |
| Add or revise an Islamic occasion | `internal/core/occasions`, `internal/core/i18n/occasions.go` | [Request flows](request-flows.md), [Reminder delivery](reminder-delivery.md) |
| Change retry or deletion behavior | `internal/core/reminders/sender.go`, `internal/adapter/out/store`, `infra/gcp` | [Reminder delivery](reminder-delivery.md), [Operations](operations.md) |
| Add persistent state | `migrations`, `internal/adapter/out/store`, `internal/domain` | [Data model](data-model.md) |
| Add a service or cloud dependency | `infra/gcp`, `internal/config`, relevant `cmd` | [Architecture](architecture.md), [Runtime and deployment](runtime-and-deployment.md) |
