#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: generate-github-env.sh --sa-key-file PATH [--project-id ID] [--tfstate-bucket NAME] [--environment NAME] [--output FILE]

Writes a shell-style env file with the GCP values needed for a GitHub environment.
EOF
}

project_id="${GCP_PROJECT_ID:-}"
sa_key_file=""
tfstate_bucket="${GCP_TFSTATE_BUCKET:-}"
environment="${GCP_ENVIRONMENT:-github}"
output_file=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --project-id)
      project_id="${2:-}"
      shift 2
      ;;
    --sa-key-file)
      sa_key_file="${2:-}"
      shift 2
      ;;
    --tfstate-bucket)
      tfstate_bucket="${2:-}"
      shift 2
      ;;
    --environment)
      environment="${2:-}"
      shift 2
      ;;
    --output)
      output_file="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ -z "$project_id" ]] && command -v gcloud >/dev/null 2>&1; then
  project_id="$(gcloud config get-value project 2>/dev/null || true)"
fi

if [[ -z "$project_id" ]]; then
  echo "project id is required" >&2
  exit 1
fi

if [[ -z "$sa_key_file" ]]; then
  echo "service account key file is required" >&2
  exit 1
fi

if [[ -z "$tfstate_bucket" ]]; then
  tfstate_bucket="${project_id}-tfstate"
fi

if [[ -z "$output_file" ]]; then
  output_file="_config/github-gcp.${environment}.env"
fi

if [[ ! -f "$sa_key_file" ]]; then
  echo "service account key file not found: $sa_key_file" >&2
  exit 1
fi

mkdir -p "$(dirname "$output_file")"

cat >"$output_file" <<EOF
# GitHub environment bundle: $environment
# Add these values to the GitHub environment named "$environment".
GCP_PROJECT_ID=$project_id
GCP_TFSTATE_BUCKET=$tfstate_bucket
GCP_REGION=europe-west1
GCP_SA_KEY=$(jq -c . "$sa_key_file")
EOF

echo "Wrote $output_file"
