# Global prayer bot

This directory is a separate application derived from the existing city bots. It has its own Go module, container image, Cloud Run services, Terraform state, Telegram token, Secret Manager entries, and PostgreSQL schema. Nothing under `global/` imports or changes the legacy runtime.

## What is implemented

- Location onboarding from Telegram coordinates, with group changes restricted to group administrators.
- Google Time Zone and reverse-geocoding lookups only when the location changes.
- Local calculation of prayer times with MWL, Egyptian, Umm al-Qura, Karachi, ISNA, Diyanet, Kemenag, MUIS, and JAKIM methods.
- Shafii/Hanafi Asr selection, three high-latitude rules, and per-prayer minute adjustments.
- `/today`, `/tomorrow`, `/next`, `/settings`, `/method`, `/madhab`, `/highlat`, `/adjust`, `/remind`, `/privacy`, and `/delete_me`.
- Indexed reminder scheduling through Cloud Scheduler, an outbox, Cloud Tasks, a private sender service, leases, and delivery idempotency keys.
- Dedicated `global_bot_testing` and `global_bot_production` PostgreSQL schemas, each with its own Goose migration table.

English is the initial UI language. The stored Telegram language code gives us a clean path to add the existing translations without coupling the two applications.

## Services

| Service | Access | Responsibility |
| --- | --- | --- |
| `webhook` | Public URL, protected by Telegram's webhook secret header | Commands, location setup, calculation settings |
| `dispatch` | Cloud Scheduler service account only | Claim due indexed schedules and create Cloud Tasks |
| `sender` | Cloud Tasks service account only | Idempotent Telegram delivery and next-occurrence planning |

The three services use one immutable image and select `/webhook`, `/dispatch`, or `/send` as the command.

## Testing and production secrets

The repository must have two GitHub environments named `testing` and `production`. Both environments use the same secret names, while GitHub supplies the value belonging to the selected environment:

| Secret | Testing value | Production value |
| --- | --- | --- |
| `GLOBAL_BOT_TOKEN` | Testing Telegram bot token | Production Telegram bot token |
| `GLOBAL_WEBHOOK_SECRET` | Testing webhook secret | Production webhook secret |
| `GLOBAL_OWNER_ID` | Testing owner ID | Production owner ID |
| `GCP_PROJECT_ID` | Existing testing GCP project | Existing production GCP project |
| `GCP_SA_KEY` | Testing deployment credentials | Production deployment credentials |
| `GCP_TFSTATE_BUCKET` | Existing testing state bucket | Existing production state bucket |
| `SUPABASE_DB_URL` | Existing database URL | Existing database URL |

Add the three new values separately in both GitHub environments, for example:

```sh
gh secret set GLOBAL_BOT_TOKEN --env testing
gh secret set GLOBAL_WEBHOOK_SECRET --env testing
gh secret set GLOBAL_OWNER_ID --env testing

gh secret set GLOBAL_BOT_TOKEN --env production
gh secret set GLOBAL_WEBHOOK_SECRET --env production
gh secret set GLOBAL_OWNER_ID --env production
```

The webhook secret must be 1-256 characters and contain only letters, numbers, `_`, or `-`. During deployment, the workflow copies the selected values into environment-specific Secret Manager secrets such as `global-prayer-bot-token-testing` and `global-prayer-bot-token-production`; values are never shared between environments. Testing and production may reuse the same database URL because their tables and migration history live in separate schemas. The global workflow never uses the legacy `APP_CONFIG`.

The existing deployment service account may need additional roles that the legacy Cloud Functions deployment did not use: `roles/run.admin`, `roles/artifactregistry.admin`, `roles/cloudtasks.admin`, `roles/cloudscheduler.admin`, `roles/secretmanager.admin`, `roles/serviceusage.apiKeysAdmin`, `roles/serviceusage.serviceUsageAdmin`, `roles/iam.serviceAccountAdmin`, `roles/iam.serviceAccountUser`, and `roles/resourcemanager.projectIamAdmin`. It also needs object access to the existing Terraform state bucket. Scope these roles to the selected testing/production project (and the state bucket) rather than organization-wide.

## Google Maps key

Terraform enables the Time Zone and Geocoding APIs, creates a dedicated API key restricted to those two APIs, creates a Secret Manager secret, and injects the key only into the webhook service. No manually created Maps key is required.

The key has API restrictions but no client-IP restriction because Cloud Run does not have a stable egress IP by default. If static egress is added later through a VPC connector and Cloud NAT, add that NAT IP as an application restriction. The Terraform state contains the sensitive key and database URL; keep the existing state bucket access tightly restricted.

## Local checks

```sh
cd global
make check
terraform -chdir=infra/gcp fmt -check -recursive
terraform -chdir=infra/gcp init -backend=false
terraform -chdir=infra/gcp validate
```

For migrations, use the existing database connection but the global migration table:

```sh
export GLOBAL_DB_SCHEMA=global_bot_testing
go run ./cmd/bootstrapdb
goose -dir migrations -table="${GLOBAL_DB_SCHEMA}.goose_db_version" postgres "$DATABASE_URL" up
```

The bootstrap command only creates the selected empty global schema. `GLOBAL_DB_SCHEMA` accepts `global_bot_testing` or `global_bot_production`. Bootstrap must happen before Goose's first run because Goose creates its schema-qualified version table before applying migration `Up` statements.

## Deployment

Run the separate **Deploy global prayer bot** GitHub workflow and choose `testing` or `production`. It uses a distinct state prefix (`prayer-bot/global-testing` or `prayer-bot/global-production`), migrates the matching `global_bot_testing` or `global_bot_production` schema, builds the global image, provisions the global resources, and then configures the selected Telegram webhook.

The workflow is intentionally manual until the new token, secrets, API quotas, privacy text, and prayer-time samples are approved in `testing`.
