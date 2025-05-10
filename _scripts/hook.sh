#!/bin/bash

DISPATCHER_ENDPOINT="https://functions.yandexcloud.net/$(terraform output -raw dispatcher_fn_id)"
export DISPATCHER_ENDPOINT

CONFIG_FILE="_config/$(terraform workspace show)/config.json"

echo "[INFO] using config file: \"$CONFIG_FILE\""

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
