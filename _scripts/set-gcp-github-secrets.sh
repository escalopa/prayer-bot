#!/usr/bin/env bash
# Push GCP deploy credentials to GitHub environment secrets.
#
# Usage:
#   ./_scripts/set-gcp-github-secrets.sh              # dev environment (default)
#   ENV=prod ./_scripts/set-gcp-github-secrets.sh
#
# Optional env:
#   PROJECT_ID=prayer-bot-infra
#   TFSTATE_BUCKET=prayer-bot-infra-tfstate
#   KEY_FILE=gcp-sa.json
#   ENV=dev
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-prayer-bot-infra}"
TFSTATE_BUCKET="${TFSTATE_BUCKET:-${PROJECT_ID}-tfstate}"
KEY_FILE="${KEY_FILE:-gcp-sa.json}"
ENV="${ENV:-dev}"

command -v gh >/dev/null || { echo "gh CLI not found"; exit 1; }

if [[ ! -f "$KEY_FILE" ]]; then
  echo "Missing $KEY_FILE — run ./_scripts/setup-gcp-deploy.sh first"
  exit 1
fi

echo "Setting GitHub secrets on environment: $ENV"
echo "  GCP_PROJECT_ID=$PROJECT_ID"
echo "  GCP_TFSTATE_BUCKET=$TFSTATE_BUCKET"
echo "  GCP_SA_KEY=<from $KEY_FILE>"

gh secret set GCP_PROJECT_ID --env "$ENV" --body "$PROJECT_ID"
gh secret set GCP_TFSTATE_BUCKET --env "$ENV" --body "$TFSTATE_BUCKET"
gh secret set GCP_SA_KEY --env "$ENV" < "$KEY_FILE"

echo "✅ GitHub secrets updated for environment: $ENV"
