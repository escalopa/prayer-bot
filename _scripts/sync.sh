#!/bin/bash
set -euo pipefail

TERRAFORM_WORKSPACE=$(terraform workspace show)
REGION=$(terraform output -raw region)

BUCKET_NAME="prayer-bot-bucket-${TERRAFORM_WORKSPACE}"
S3_ENDPOINT="https://storage.yandexcloud.net"

LOCAL_DIR="_config/${TERRAFORM_WORKSPACE}"

mkdir -p "$LOCAL_DIR"

echo "[INFO] Syncing bucket '${BUCKET_NAME}' into '${LOCAL_DIR}'..."

export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-}"

if [[ -z "$AWS_ACCESS_KEY_ID" || -z "$AWS_SECRET_ACCESS_KEY" ]]; then
  echo "[ERROR] AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be set as environment variables or have default values."
  exit 1
fi

aws --endpoint-url "$S3_ENDPOINT" --region "$REGION" s3 sync "s3://${BUCKET_NAME}/" "$LOCAL_DIR/"

echo "[INFO] sync complete!"
