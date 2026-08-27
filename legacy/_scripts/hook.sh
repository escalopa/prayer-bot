#!/bin/bash

if [[ -z "$APP_CONFIG_PATH" ]]; then
  echo "[ERROR] APP_CONFIG_PATH is not set"
  exit 1
fi

if [[ -z "$WEBHOOK_URL" ]]; then
  echo "[ERROR] WEBHOOK_URL is not set"
  exit 1
fi

CONFIG_FILE="$APP_CONFIG_PATH"

echo "[INFO] using config file: \"$CONFIG_FILE\""
echo "[INFO] using webhook url: \"$WEBHOOK_URL\""

jq -c 'to_entries[]' "$CONFIG_FILE" | while read -r entry; do
    BOT_ID=$(echo "$entry" | jq -r '.value.bot_id')
    TOKEN=$(echo "$entry" | jq -r '.value.token')
    SECRET=$(echo "$entry" | jq -r '.value.secret')

    echo "[INFO] setting webhook for bot_id: $BOT_ID"

    curl -s -X POST "https://api.telegram.org/bot${TOKEN}/setWebhook" \
         -H "Content-Type: application/json" \
         -d "{\"url\": \"${WEBHOOK_URL}\", \"secret_token\": \"${SECRET}\"}" | jq

    echo "[INFO] webhook set for bot_id: $BOT_ID"
done
