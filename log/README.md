# `log`

A thin wrapper over the standard library's `log/slog` that gives every module a
single, consistent structured logger writing **JSON to stderr** — the format
Google Cloud Logging ingests automatically from Cloud Functions.

## Usage

```go
log.Error("reminder.handler.send: failed",
    log.Op("send"), log.BotID(botID), log.ChatID(chatID), log.Err(err))
```

## What it provides ([`log.go`](log.go))

- **Level functions:** `Debug`, `Info`, `Warn`, `Error`.
- **Typed attribute helpers** for the fields used across the codebase:
  `Op(name)`, `BotID(id)`, `ChatID(id)`, `Err(err)`, `String(k, v)`, `Int(k, v)`.

## Log level

The level is set once at `init()` from the `LOG_LEVEL` environment variable
(`debug` / `info` / `warn` / `error`). It **defaults to `warn`**, so `Info`/`Debug`
lines are suppressed in production unless `LOG_LEVEL` is lowered.

## Convention

Each package defines small local helpers (e.g. `logDispatcher`, `logReminder`,
`logPG`) that prefix messages with a `component.operation` string and attach
`Op(...)`. This keeps log messages greppable and consistently namespaced. See
[`serverless/dispatcher/internal/handler/log.go`](../serverless/dispatcher/internal/handler/log.go)
for the pattern.
