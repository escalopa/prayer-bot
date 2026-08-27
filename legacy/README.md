# Legacy city bots

This directory contains the original city-specific Telegram bots, including the
existing Kazan and Naples deployments. They are retained in maintenance mode.
New user-facing features belong in the repository's primary
[`global/`](../global/) application.

The legacy bots share one Go module and are selected by `bot_id` from
`APP_CONFIG`. They run as GCP Cloud Functions backed by Supabase Postgres and
CSV prayer schedules stored in GCS.

## Layout

```text
legacy/
├── domain/, config/, log/, internal/db/  # shared Go module packages
├── serverless/                           # dispatcher, reminder and loader modules
├── migrations/                           # shared public-schema Goose migrations
├── infra/gcp/                            # legacy Cloud Functions Terraform
├── _scripts/                             # deploy, webhook and city-data helpers
├── Makefile                              # local checks across all legacy Go modules
└── revive.toml                           # legacy lint configuration
```

## Maintenance checks

```sh
cd legacy
make check
terraform -chdir=infra/gcp fmt -check -recursive
terraform -chdir=infra/gcp init -backend=false
terraform -chdir=infra/gcp validate
```

## Legacy deployment

The **Deploy to GCP** workflow continues to deploy only this directory. Its
paths, configuration output, module preparation, migration step, Terraform
working directory, webhook registration, and profile sync all target
`legacy/...`.

For operational details, see:

- [Legacy Cloud Functions](serverless/README.md)
- [Legacy Terraform](infra/gcp/README.md)
- [Legacy scripts](./_scripts/README.md)
- [Legacy migrations](migrations/README.md)
