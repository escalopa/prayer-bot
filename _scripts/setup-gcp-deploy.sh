#!/usr/bin/env bash
# Bootstrap GCP deploy service account + Terraform state bucket for CI/CD.
#
# Usage:
#   ./_scripts/setup-gcp-deploy.sh
#   ./_scripts/setup-gcp-deploy.sh other-project-id
#
# Optional env:
#   PROJECT_ID=prayer-bot-infra   (default)
#   REGION=europe-west1
#   SA_NAME=prayer-bot-deploy
#   TFSTATE_BUCKET=...            (default: ${PROJECT_ID}-tfstate)
#   KEY_FILE=gcp-sa.json          (gitignored — do not commit)
#   NONINTERACTIVE=1              overwrite existing key without prompt
#   FORCE_KEY=1                   same as NONINTERACTIVE for key file
set -euo pipefail

PROJECT_ID="${1:-${PROJECT_ID:-prayer-bot-infra}}"
REGION="${REGION:-europe-west1}"
SA_NAME="${SA_NAME:-prayer-bot-deploy}"
KEY_FILE="${KEY_FILE:-gcp-sa.json}"

TFSTATE_BUCKET="${TFSTATE_BUCKET:-${PROJECT_ID}-tfstate}"
SA_EMAIL="${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

command -v gcloud >/dev/null || { echo "gcloud CLI not found"; exit 1; }

echo "Project:        $PROJECT_ID"
echo "Region:         $REGION"
echo "Tfstate bucket: $TFSTATE_BUCKET"
echo "Deploy SA:      $SA_EMAIL"
echo ""

gcloud config set project "$PROJECT_ID"

echo "→ Creating Terraform state bucket (if missing)..."
if gcloud storage buckets describe "gs://${TFSTATE_BUCKET}" >/dev/null 2>&1; then
  echo "  Bucket gs://${TFSTATE_BUCKET} already exists"
else
  gcloud storage buckets create "gs://${TFSTATE_BUCKET}" \
    --project="$PROJECT_ID" \
    --location="$REGION" \
    --uniform-bucket-level-access
  gcloud storage buckets update "gs://${TFSTATE_BUCKET}" --versioning
  echo "  Created gs://${TFSTATE_BUCKET} (versioning enabled)"
fi

echo "→ Creating deploy service account (if missing)..."
if gcloud iam service-accounts describe "$SA_EMAIL" --project="$PROJECT_ID" >/dev/null 2>&1; then
  echo "  Service account already exists"
else
  gcloud iam service-accounts create "$SA_NAME" \
    --project="$PROJECT_ID" \
    --display-name="Prayer bot CI/CD deploy"
  echo "  Created $SA_EMAIL"
fi

echo "→ Granting project IAM roles..."
ROLES=(
  roles/serviceusage.serviceUsageAdmin
  roles/resourcemanager.projectIamAdmin
  roles/iam.serviceAccountAdmin
  roles/iam.serviceAccountUser
  roles/cloudfunctions.admin
  roles/run.admin
  roles/storage.admin
  roles/cloudscheduler.admin
  roles/eventarc.admin
  roles/pubsub.admin
)

for role in "${ROLES[@]}"; do
  echo "  $role"
  gcloud projects add-iam-policy-binding "$PROJECT_ID" \
    --member="serviceAccount:${SA_EMAIL}" \
    --role="$role" \
    --quiet >/dev/null
done

echo "→ Granting tfstate bucket access..."
gcloud storage buckets add-iam-policy-binding "gs://${TFSTATE_BUCKET}" \
  --member="serviceAccount:${SA_EMAIL}" \
  --role="roles/storage.objectAdmin" \
  --quiet >/dev/null

if [[ -f "$KEY_FILE" && "${NONINTERACTIVE:-}" != "1" && "${FORCE_KEY:-}" != "1" ]]; then
  echo ""
  read -r -p "$KEY_FILE already exists. Overwrite? [y/N] " confirm
  if [[ "${confirm,,}" != "y" ]]; then
    echo "Skipped key creation."
    echo "Run ./_scripts/set-gcp-github-secrets.sh to update GitHub secrets with an existing key."
    exit 0
  fi
fi

echo "→ Creating service account key → $KEY_FILE"
gcloud iam service-accounts keys create "$KEY_FILE" \
  --iam-account="$SA_EMAIL" \
  --project="$PROJECT_ID"

echo ""
echo "✅ GCP bootstrap complete."
echo "Next: ./_scripts/set-gcp-github-secrets.sh"
