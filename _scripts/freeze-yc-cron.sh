#!/usr/bin/env bash
set -euo pipefail

if ! command -v yc >/dev/null 2>&1; then
  echo "[ERROR] yc CLI is required"
  exit 1
fi

ENVIRONMENT="${ENVIRONMENT:-${WORKSPACE:-prod}}"
TRIGGER_NAME="reminder"

echo "[INFO] pausing YC reminder timer trigger in ${ENVIRONMENT} workspace context"

yc serverless trigger list --format json | jq -e --arg name "$TRIGGER_NAME" '.[] | select(.name == $name)' >/dev/null

yc serverless trigger pause "$TRIGGER_NAME"
echo "[INFO] paused trigger: $TRIGGER_NAME"
