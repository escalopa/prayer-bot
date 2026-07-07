# `config`

Loads the multi-bot application configuration that every serverless function
needs at startup.

## How it works

[`config.go`](config.go) exposes a single function:

```go
func Load() (map[int64]*domain.BotConfig, error)
```

1. Reads the `APP_CONFIG` environment variable.
2. Base64-decodes it.
3. JSON-unmarshals the result into a map keyed by **bot id**.

The value type is [`domain.BotConfig`](../domain/config.go): `bot_id`, `owner_id`,
`token`, `secret`, and `location` (IANA timezone). One process serves many bots;
the map is how each request/reminder is routed to the right bot's token and
timezone.

## Why base64

The config contains bot tokens and webhook secrets. Storing it as a single
base64-encoded environment variable keeps secrets out of the repo and makes it
trivial to inject via GitHub Actions / Terraform (`APP_CONFIG`) without managing
a config file on the function's filesystem.

## Example (decoded) shape

```json
{
  "123456789": {
    "bot_id": 123456789,
    "owner_id": 987654321,
    "token": "123456789:AA...",
    "secret": "webhook-secret",
    "location": "Europe/Moscow"
  }
}
```

> **Note:** this is the *runtime* config read from `APP_CONFIG`. The database URL
> is loaded separately in [`internal/db`](../internal/db/README.md) from
> `DATABASE_URL` / `SUPABASE_DB_URL`.
