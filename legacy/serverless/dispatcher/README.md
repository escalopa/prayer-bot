# dispatcher

The **HTTP Cloud Function that receives Telegram webhook updates** and drives all
user-facing interaction: commands, inline keyboards, multi-step flows, and admin
actions.

Module: `github.com/escalopa/prayer-bot/dispatcher`.

## Request flow

1. [`dispatcher.go`](dispatcher.go) — the GCP entrypoint (`DispatcherHTTP`).
   Rejects non-`POST`, lazily builds the handler (`sync.Once`), then:
   - **Authenticates** the request by matching the
     `X-Telegram-Bot-Api-Secret-Token` header against a bot's `secret` →
     resolves the `bot_id`.
   - Reads the body and calls `handler.Handle(ctx, botID, body)`.
2. [`internal/handler/handler.go`](internal/handler/handler.go) — `Handle`
   decodes the Telegram `Update`, gets/creates the `go-telegram/bot` client for
   that bot, and feeds the update through the library's router (`ProcessUpdate`).

### Middleware chain

Handlers are wrapped in composable middleware (see `opts()`):

```
errorH( chatH( authorizeH( fn ) ) )
```

- **`errorH`** — recovers panics and logs handler errors.
- **`chatH`** — loads (or first-time creates) the `domain.Chat` for the update
  and stashes it in `context`. New chats get default reminder offsets here.
- **`authorizeH`** — gates admin commands to the bot's configured `owner_id`.

> The bot runs in **webhook mode** with `bot.WithNotAsyncHandlers()`: update
> processing must finish before the HTTP response returns, because the function
> context is cancelled on return.

## File map (`internal/handler`)

| File | Responsibility |
|------|----------------|
| [`handler.go`](internal/handler/handler.go) | Entry `Handle`, router options, middleware, auth, chat load/create. |
| [`command.go`](internal/handler/command.go) | All slash commands: `/today`, `/tomorrow`, `/date`, `/next`, `/remind`, `/bug`, `/feedback`, `/language`, `/subscribe`, admin `/stats`, `/announce`, `/reply`, etc. |
| [`query.go`](internal/handler/query.go) | Inline-keyboard callback handlers (date picker, language, reminder menu, Jamaat settings). |
| [`keyboard.go`](internal/handler/keyboard.go) | Builders for the inline keyboards (months, days, languages, reminder menus). |
| [`state.go`](internal/handler/state.go) | Multi-step conversational flows (bug/feedback capture, admin reply/announce) driven by the `chats.state` column. |
| [`languages.go`](internal/handler/languages.go) | Loads embedded per-language YAML (`languages/*.yaml`) into typed `Text`; localization provider. |
| [`helper.go`](internal/handler/helper.go) | `context` accessors for bot id/chat, time helpers, message formatting. |
| [`log.go`](internal/handler/log.go) | Namespaced logging helpers. |

## Sub-packages

- [`internal/service`](internal/service/db.go) — `NewDB` wrapper over the shared
  `internal/db.Store`.
- [`internal/botprofile`](internal/botprofile/sync.go) — applies each bot's name,
  descriptions, and command list via the Telegram Bot API. **Deploy-time only**,
  not on the request path.
- [`cmd/syncprofile`](cmd/syncprofile/main.go) — CLI that runs `botprofile` for
  every bot in the config; invoked from CI after deploy.

## Localization

Add a language by dropping a `languages/<code>.yaml` file — it is embedded at
build time. See the repo root README's "Add a language" section.
