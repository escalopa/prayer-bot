# `_scripts`

Operational and one-off helper scripts. Not part of the deployed functions — the
leading underscore keeps the directory out of Go's default package globs.

## Contents

| Path | What it does |
|------|--------------|
| [`hook.sh`](hook.sh) | Registers each bot's Telegram webhook (URL + secret token). Run in the CI **Hook** stage after deploy. |
| [`setup-gcp-deploy.sh`](setup-gcp-deploy.sh) | Bootstraps the GCP project for deployment (APIs, service accounts, state bucket). |
| [`set-gcp-github-secrets.sh`](set-gcp-github-secrets.sh) | Pushes the required secrets/vars into the GitHub repository for the Actions pipeline. |
| [`update_serverless_packages.sh`](update_serverless_packages.sh) | Bulk-updates Go dependencies across the root module and all three serverless modules. |
| [`city/`](city) | A small standalone Go module that generates a prayer-schedule CSV for a city (see [`city/main.go`](city/main.go)); `moskva.csv` is a sample output in the loader's expected format. |

## Notes

- `city/` has its **own `go.mod`** and is intentionally separate from the
  application modules.
- The CSV `city/` produces matches the format the [loader](../serverless/loader/README.md)
  parses: `date, fajr, shuruq, dhuhr, asr, maghrib, isha`.
