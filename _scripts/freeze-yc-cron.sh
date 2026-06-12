#!/usr/bin/env bash
set -euo pipefail

if ! command -v yc >/dev/null 2>&1; then
  echo "[ERROR] yc CLI is required"
  exit 1
fi

ENVIRONMENT="${ENVIRONMENT:-${WORKSPACE:-prod}}"
TRIGGER_NAME="reminder"

echo "[INFO] disabling YC reminder timer trigger in ${ENVIRONMENT} workspace context"

yc serverless trigger list --format json | jq -e --arg name "$TRIGGER_NAME" '.[] | select(.name == $name)' >/dev/null

yc serverless trigger disable "$TRIGGER_NAME"
echo "[INFO] disabled trigger: $TRIGGER_NAME"
