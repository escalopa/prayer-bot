#!/usr/bin/env bash
set -euo pipefail

# Base module path
REPO="github.com/escalopa/prayer-bot"

# Current commit hash
HASH=$(git rev-parse HEAD)
echo "Using commit hash: $HASH"

# Ensure serverless directory exists
if [ ! -d "serverless" ]; then
  echo "❌ Error: 'serverless' directory not found."
  exit 1
fi

update_module() {
  local dir=$1
  echo "➡️  Updating $dir"
  (cd "$dir" && go get "$REPO@$HASH")
}

# Yandex Cloud serverless modules (dispatcher, reminder, loader)
for dir in serverless/*/; do
  [ -d "$dir" ] || continue
  update_module "$dir"

  # GCP Cloud Functions deployable modules (serverless/*/function)
  if [ -d "${dir}function" ]; then
    update_module "${dir}function"
  fi
done

# GCP webhook proxy
update_module "proxy/function"

echo "✅ All serverless and GCP function packages updated to $HASH"
