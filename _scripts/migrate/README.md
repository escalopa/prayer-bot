# Chat Migration Script

This script migrates chat data from the old YDB table schema to the new one.

## Prerequisites

- Go 1.23 or higher
- Access to both old and new YDB databases

## Environment Variables

Set the following environment variables before running the script:

- `OLD_DB_CONNECTION_STRING`: Connection string for the old YDB database
- `NEW_DB_CONNECTION_STRING`: Connection string for the new YDB database
- `YDB_TOKEN`: Access token for authentication to both databases

## Usage

```bash
# Set environment variables
export OLD_DB_CONNECTION_STRING="grpcs://ydb.serverless.yandexcloud.net:2135/?database=/ru-central1/..."
export NEW_DB_CONNECTION_STRING="grpcs://ydb.serverless.yandexcloud.net:2135/?database=/ru-central1/..."
export YDB_TOKEN="your-access-token"

# Run the migration
cd _scripts/migrate
go run main.go
```

## What the script does

1. Connects to both old and new YDB databases
2. Fetches all chats from the old `chats` table
3. Transforms the data:
   - Most fields remain the same (chat_id, bot_id, language_code, state, subscribed, subscribed_at, created_at)
   - Creates new `reminder` JSON object with:
     - `today`: LastAt set to current time (Moscow timezone)
     - `soon`: Offset from old `reminder_offset`, MessageID from old `reminder_message_id`
     - `arrive`: MessageID from old `jamaat_message_id`
     - `jamaat`: Enabled if both `subscribed` and `jamaat` were true, with default delay configs
4. Uses UPSERT to insert/update records in the new database

## Idempotency

The script uses UPSERT statements, making it safe to run multiple times. Subsequent runs will update existing records rather than failing or creating duplicates.

## Transformation Details

### Reminder Field

The reminder JSON structure:
```json
{
  "today": {
    "offset": 0,
    "message_id": 0,
    "last_at": "2025-10-26T12:00:00+03:00"
  },
  "soon": {
    "offset": 600000000000,
    "message_id": 12345,
    "last_at": "2025-10-26T12:00:00+03:00"
  },
  "arrive": {
    "offset": 0,
    "message_id": 67890,
    "last_at": "2025-10-26T12:00:00+03:00"
  },
  "jamaat": {
    "enabled": true,
    "delay": {
      "fajr": 600000000000,
      "shuruq": 600000000000,
      "dhuhr": 600000000000,
      "asr": 600000000000,
      "maghrib": 600000000000,
      "isha": 1200000000000
    }
  }
}
```

Note: All durations are in nanoseconds (Go's default). For example, 10 minutes = 600000000000 nanoseconds.
