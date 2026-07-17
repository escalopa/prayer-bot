# Global prayer bot

This directory is a separate application derived from the existing city bots. It has its own Go module, container image, Cloud Run services, Terraform state, Telegram token, Secret Manager entries, and PostgreSQL schema. Nothing under `global/` imports or changes the legacy runtime.

## What is implemented

- Location onboarding from Telegram coordinates, with group changes restricted to group administrators.
- Google Time Zone and reverse-geocoding lookups only when the location changes.
- Local calculation of prayer times with MWL, Egyptian, Umm al-Qura, Karachi, ISNA, Diyanet, Kemenag, MUIS, and JAKIM methods.
- Shafii/Hanafi Asr selection, three high-latitude rules, and per-prayer minute adjustments.
- A persistent two-column Telegram menu for today, tomorrow, the next prayer, location, settings, reminders, language, and help.
- A Telegram Mini App, opened from the bot menu, for today/tomorrow schedules, location, calculation settings, Hijri correction, and all reminder toggles without typed commands.
- Inline button pickers for calculation method, madhab, high-latitude rule, per-prayer adjustments, reminder state, and language. The equivalent typed commands remain available.
- Localized messages, reply keyboards, prayer names, dates, Mini App, and reminder deliveries in English, Arabic, Spanish, French, Russian, Turkish, Uzbek, and Tatar. The public Telegram bot name and description remain stable for every user.
- Gregorian and calculated Umm al-Qura Hijri dates on every daily schedule, with a per-chat moon-sighting correction from -2 to +2 days.
- Opt-in weekly reminders for Monday/Thursday voluntary fasting (20:00 on the preceding evening) and reading Surah Al-Kahf on Friday (09:00), scheduled in the saved local timezone.
- An embedded welcome illustration sent on `/start` and a generated bot avatar installed during profile synchronization.
- A localized feedback and bug-report flow that accepts text or screenshots in a private chat and delivers them directly to the configured owner with the sender's disclosed Telegram identity.
- An owner-only `/admin` dashboard with aggregate user activity, onboarding, language, calculation-method, reminder-adoption, queue, and delivery-health metrics.
- Indexed reminder scheduling through Cloud Scheduler, an outbox, Cloud Tasks, a private sender service, leases, and delivery idempotency keys.
- Dedicated `global_bot_testing` and `global_bot_production` PostgreSQL schemas, each with its own Goose migration table.

The initial UI language follows the user's Telegram language when supported and otherwise falls back to English. A language selected inside the bot is persisted and is not overwritten by later Telegram updates.

Hijri dates use the calculated Umm al-Qura calendar. Because official local moon-sighting dates can differ by a day or two, users can correct the displayed date under **Settings → Hijri date correction**. The correction is applied only to the Hijri display; it never shifts prayer calculations.

## Owner dashboard and feedback

The Telegram account configured by `GLOBAL_OWNER_ID` can open the private owner dashboard with `/admin` or the backward-compatible `/status` command. The command is intentionally absent from the public command menu, is ignored for every other user, and is unavailable in groups. Its inline buttons show aggregate metrics only; the dashboard never lists Telegram IDs, coordinates, or individual user records.

Feedback arrives in the owner's private bot chat as a metadata message followed by a copy of the user's original text or screenshot. A **Contact user** button opens the sender's Telegram profile. Replies typed inside the bot chat are not forwarded, so contact the sender through that button or the linked username.

## Services

| Service | Access | Responsibility |
| --- | --- | --- |
| `webhook` | Public URL; webhook protected by Telegram's secret header, Mini App API protected by signed init data | Commands, Mini App, location setup, calculation settings |
| `dispatch` | Cloud Scheduler service account only | Claim due indexed schedules and create Cloud Tasks |
| `sender` | Cloud Tasks service account only | Idempotent Telegram delivery and next-occurrence planning |

The three services use one immutable image and select `/webhook`, `/dispatch`, or `/send` as the command. The webhook binary embeds the Mini App and serves it at `/app/`, so the feature does not add another Cloud Run service, container image, database, migration, or secret.

## Testing and production secrets

The global workflow reuses the existing GitHub environments: logical `testing` deployments read secrets from `dev`, and logical `production` deployments read secrets from `prod`. No duplicate infrastructure environments or credentials are required.

| Secret | `dev` value for testing | `prod` value for production |
| --- | --- | --- |
| `GLOBAL_BOT_TOKEN` | Testing Telegram bot token | Production Telegram bot token |
| `GLOBAL_WEBHOOK_SECRET` | Testing webhook secret | Production webhook secret |
| `GLOBAL_OWNER_ID` | Testing owner ID | Production owner ID |
| `GCP_PROJECT_ID` | Existing dev value | Existing prod value |
| `GCP_SA_KEY` | Existing dev value | Existing prod value |
| `GCP_TFSTATE_BUCKET` | Existing dev value | Existing prod value |
| `SUPABASE_DB_URL` | Existing database URL | Existing database URL |
| `SUPABASE_DB_DIRECT_URL` | Existing direct database URL | Existing direct database URL |

Only the three global-bot values are new; add them to the existing environments:

```sh
gh secret set GLOBAL_BOT_TOKEN --env dev
gh secret set GLOBAL_WEBHOOK_SECRET --env dev
gh secret set GLOBAL_OWNER_ID --env dev

gh secret set GLOBAL_BOT_TOKEN --env prod
gh secret set GLOBAL_WEBHOOK_SECRET --env prod
gh secret set GLOBAL_OWNER_ID --env prod
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

For migrations, use the existing direct database connection but the global migration table:

```sh
export GLOBAL_DB_SCHEMA=global_bot_testing
go run ./cmd/bootstrapdb
goose -dir migrations -table="${GLOBAL_DB_SCHEMA}.goose_db_version" postgres "$DATABASE_URL" up
```

The bootstrap command only creates the selected empty global schema. `GLOBAL_DB_SCHEMA` accepts `global_bot_testing` or `global_bot_production`. Bootstrap must happen before Goose's first run because Goose creates its schema-qualified version table before applying migration `Up` statements.

## Deployment

Run the separate **Deploy global prayer bot** GitHub workflow and choose `testing` or `production`. It uses a distinct state prefix (`prayer-bot/global-testing` or `prayer-bot/global-production`), migrates the matching `global_bot_testing` or `global_bot_production` schema, builds the global image, provisions the global resources, and then configures the selected Telegram webhook, Mini App menu button, stable public profile, command menu, and avatar. The Mini App URL is derived from the environment's existing webhook URL; no new GitHub variable is required.

The workflow is intentionally manual until the new token, secrets, API quotas, privacy text, and prayer-time samples are approved through the logical `testing` deployment.
