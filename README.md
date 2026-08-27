# Global prayer bot 🙏

The Global prayer bot is this repository's primary application. It provides
localized prayer times anywhere in the world through Telegram, with a Mini App
for settings, reminders, Qibla direction, Hijri dates, Islamic occasions, and
a private rolling calendar feed.

The production bot is actively developed. New product work belongs in
[`global/`](global/). The older per-city bots remain in this repository for
maintenance only; they are not the baseline for new features.

## Start here

- **[Global bot overview and local checks](global/README.md)** — feature set,
  services, configuration, and deployment entry points.
- **[Global engineering guide](global/docs/README.md)** — architecture, code
  map, request flows, data model, testing, deployment, and operations.
- **[Calculation methodology](https://escalopa.github.io/prayer-bot/)** — the
  public explanation of prayer-time, Qibla, Hijri-calendar, occasion, and
  calendar calculations. The versioned source is
  [`docs/calculation-methods.tex`](docs/calculation-methods.tex).

## Global bot

The application under [`global/`](global/) is an independent Go module and
deployment. It owns its Cloud Run services, container image, Terraform state,
Telegram token and webhook, Secret Manager entries, and environment-specific
PostgreSQL schemas.

| Service | Access | Responsibility |
| --- | --- | --- |
| `webhook` | Public Telegram endpoint; Mini App requests use signed init data | Telegram updates, onboarding, preferences, Mini App API |
| `dispatch` | Cloud Scheduler service account | Claims due reminder schedules and creates Cloud Tasks |
| `sender` | Cloud Tasks service account | Idempotently delivers Telegram messages and advances schedules |

### Core capabilities

- Prayer times from a saved location, with multiple calculation methods,
  madhab selection, high-latitude rules, and per-prayer adjustments.
- Telegram Mini App with localized schedules, onboarding, settings, reminders,
  Hijri correction, Qibla bearing, optional live compass, and prayer cards.
- Gregorian--Hijri dates, Islamic occasions, voluntary-fasting reminders, and
  a private rolling calendar subscription.
- Localized UI and notifications in English, Arabic, Spanish, French, Russian,
  Turkish, Uzbek, and Tatar.
- Reliable scheduled delivery using Cloud Scheduler, an outbox, Cloud Tasks,
  private sender leases, and idempotency keys.

### Development and verification

```sh
cd global
make test
```

Run the full local check before submitting a change:

```sh
cd global
make check
terraform -chdir=infra/gcp fmt -check -recursive
terraform -chdir=infra/gcp init -backend=false
terraform -chdir=infra/gcp validate
```

Testing and production are deployed manually through the **Deploy global prayer
bot** GitHub Actions workflow. The precise secret inventory, access
requirements, and rollout sequence are maintained in
[`global/README.md`](global/README.md#testing-and-production-secrets).

## Repository layout

```text
.
├── global/              # Primary Global bot application and Go module
│   ├── cmd/             # webhook, dispatch, sender, bootstrap and profile commands
│   ├── internal/        # domain, core logic, adapters, Mini App, configuration
│   ├── migrations/      # Global bot Goose migrations
│   ├── infra/gcp/       # Global Cloud Run, Tasks, Scheduler and Secret Manager Terraform
│   └── docs/            # Global architecture and operating documentation
├── serverless/          # Legacy city-bot Cloud Functions (maintenance only)
├── domain/              # Shared legacy models and value types
├── config/              # Legacy APP_CONFIG loader
├── internal/db/         # Legacy Postgres repository
├── migrations/          # Legacy database migrations
├── infra/gcp/           # Legacy Cloud Functions Terraform
└── _scripts/            # Legacy operational helpers
```

## Legacy city bots (maintenance only)

The original city-bot runtime remains available for the existing Kazan and
Naples deployments. It uses the Cloud Functions architecture under
[`serverless/`](serverless/) and shared root packages. Keep it operational and
apply fixes when required, but do not add new product features there.

- [Legacy runtime documentation](serverless/README.md)
- [Legacy infrastructure documentation](infra/gcp/README.md)
- [Legacy contribution notes](serverless/README.md#shared-conventions)

## Contributing

1. Treat [`global/`](global/) as the default destination for feature work.
2. Read the relevant document in the [Global engineering guide](global/docs/README.md)
   before changing code or infrastructure.
3. Add or update focused unit tests with every behavior change.
4. Open a pull request; the Global CI workflow runs race-enabled tests and a
   container build for changes under `global/`.

## References

- [Telegram Bot API](https://core.telegram.org/bots/api)
- [go-telegram](https://github.com/go-telegram)
