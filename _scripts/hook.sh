#!/bin/bash

if [[ -z "$APP_CONFIG_PATH" ]]; then
  echo "[ERROR] APP_CONFIG_PATH is not set"
  exit 1
fi

if [[ -z "$WEBHOOK_URL" && -z "$DISPATCHER_FUNCTION_ID" ]]; then
  echo "[ERROR] WEBHOOK_URL or DISPATCHER_FUNCTION_ID must be set"
  exit 1
fi

if [[ -n "$WEBHOOK_URL" ]]; then
  export DISPATCHER_ENDPOINT="$WEBHOOK_URL"
else
  export DISPATCHER_ENDPOINT="https://functions.yandexcloud.net/${DISPATCHER_FUNCTION_ID}"
fi

CONFIG_FILE="$APP_CONFIG_PATH"

echo "[INFO] using config file: \"$CONFIG_FILE\""
if [[ -n "$WEBHOOK_URL" ]]; then
  echo "[INFO] using webhook url: \"$WEBHOOK_URL\""
else
  echo "[INFO] using dispatcher function ID: \"$DISPATCHER_FUNCTION_ID\""
fi

jq -c 'to_entries[]' "$CONFIG_FILE" | while read -r entry; do
    BOT_ID=$(echo "$entry" | jq -r '.value.bot_id')
    TOKEN=$(echo "$entry" | jq -r '.value.token')
    SECRET=$(echo "$entry" | jq -r '.value.secret')

    echo "[INFO] setting webhook for bot_id: $BOT_ID"

    curl -s -X POST "https://api.telegram.org/bot${TOKEN}/setWebhook" \
         -H "Content-Type: application/json" \
         -d "{\"url\": \"${DISPATCHER_ENDPOINT}\", \"secret_token\": \"${SECRET}\"}" | jq

    echo "[INFO] webhook set for bot_id: $BOT_ID"
done
