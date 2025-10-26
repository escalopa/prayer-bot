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

# Iterate over first-level subdirectories
for dir in serverless/*/; do
  [ -d "$dir" ] || continue
  echo "➡️  Entering $dir"

  # Run go get inside the subdirectory
  (cd "$dir" && go get "$REPO@$HASH")

done

echo "✅ All serverless subprojects updated to $HASH"
