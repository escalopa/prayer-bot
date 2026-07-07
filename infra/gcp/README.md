# `infra/gcp`

Terraform that provisions the entire GCP footprint for prayer-bot. Applied from
CI (the **Deploy** stage of [`deploy.yaml`](../../.github/workflows/deploy.yaml)),
with per-environment state in a GCS backend.

## Files

| File | Contents |
|------|----------|
| [`main.tf`](main.tf) | All resources (functions, buckets, IAM, scheduler, Eventarc). |
| [`variables.tf`](variables.tf) | Inputs (see below). |
| [`outputs.tf`](outputs.tf) | `dispatcher_url`, `reminder_url`, `loader_name`, `data_bucket_name`. |
| [`versions.tf`](versions.tf) | Terraform / provider version pins and backend. |

## What it creates

- **Three Cloud Functions (2nd gen / Cloud Run):**
  - `dispatcher` — public HTTP (Telegram calls it).
  - `reminder` — HTTP, invoked only by the scheduler's service account.
  - `loader` — CloudEvent, wired to the data bucket via an Eventarc trigger.
- **Two GCS buckets:** `source` (zipped function code) and `data` (uploaded
  schedule CSVs that trigger the loader).
- **Cloud Scheduler job** that pings the reminder function on a cron.
- **Service accounts & IAM:** a runtime SA (logging, storage, Eventarc receiver)
  and a scheduler SA allowed to invoke the reminder; plus the deploy SA's admin
  bindings. `APP_CONFIG` and the runtime `DATABASE_URL` (Supabase pooler) are
  passed to the functions as environment variables.

## Key variables ([`variables.tf`](variables.tf))

| Variable | Purpose |
|----------|---------|
| `project_id`, `region` | Target GCP project/region. |
| `environment` | `dev` / `prod` — drives naming and state separation. |
| `app_config_path` | Path to the base64 bot config injected as `APP_CONFIG`. |
| `supabase_db_url` | Runtime DB URL (pooler, port 6543). |
| `enable_loader_trigger` | Toggle the GCS→loader Eventarc trigger. |
| `enable_scheduler` | Toggle the reminder cron. |
| `deploy_service_account_email` | SA used by CI to apply. |
| `dispatcher_timeout_seconds` | Function timeout tuning. |

## Runtime vs. migration DB URLs

- **Functions** use the Supabase **pooler** URL (port 6543, simple query protocol).
- **Goose migrations** use the **direct** URL (port 5432). See
  [`migrations`](../../migrations/README.md).
