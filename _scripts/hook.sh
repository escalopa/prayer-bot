#!/bin/bash

if [[ -z "$CONFIG_FILE_PATH" ]]; then
  echo "[ERROR] CONFIG_FILE_PATH is not set"
  exit 1
fi

if [[ -z "$DISPATCHER_FUNCTION_ID" ]]; then
  echo "[ERROR] DISPATCHER_FUNCTION_ID is not set"
  exit 1
fi

export DISPATCHER_ENDPOINT="https://functions.yandexcloud.net/${DISPATCHER_FUNCTION_ID}"

CONFIG_FILE="$CONFIG_FILE_PATH"

echo "[INFO] using config file: \"$CONFIG_FILE\""
echo "[INFO] using dispatcher function ID: \"$DISPATCHER_FUNCTION_ID\""

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
