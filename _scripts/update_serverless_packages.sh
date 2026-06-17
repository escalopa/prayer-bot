#!/usr/bin/env bash
set -euo pipefail

REPO="github.com/escalopa/prayer-bot"
HASH=$(git rev-parse HEAD)
echo "Using commit hash: $HASH"

if [ ! -d "serverless" ]; then
  echo "❌ Error: 'serverless' directory not found."
  exit 1
fi

update_module() {
  local dir=$1
  echo "➡️  Updating $dir"
  (cd "$dir" && go get "$REPO@$HASH")
}

for dir in serverless/dispatcher serverless/reminder serverless/loader; do
  update_module "$dir"
done

echo "✅ All GCP function packages updated to $HASH"
