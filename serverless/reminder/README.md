# reminder

The **HTTP Cloud Function invoked on a schedule** (Cloud Scheduler, roughly once
a minute) that decides which subscribers are due a prayer reminder and sends it.

Module: `github.com/escalopa/prayer-bot/reminder`.

## Flow

1. [`reminder.go`](reminder.go) — the GCP entrypoint (`ReminderHTTP`). Lazily
   builds the handler, then fans out over **every configured bot** with an
   `errgroup`, calling `handler.Handle(ctx, botID)` for each. Per-bot errors are
   logged, not propagated (one bot failing must not block the others).
2. [`internal/handler/handler.go`](internal/handler/handler.go) — `Handle` for a
   single bot:
   - Loads subscriber chat ids → their `Chat` rows.
   - Loads today's `PrayerDay` (with `NextDay`) in the bot's timezone.
   - Fans out over chats (bounded by `maxConcurrentReminderSends`) and, for each,
     evaluates the three reminder types.

## The three reminder types ([`reminder.go`](internal/handler/reminder.go))

Each implements the `ReminderType` interface (`ShouldTrigger` / `Send` / `Name`):

| Type | Fires when | Message |
|------|-----------|---------|
| **`TomorrowReminder`** | `offset` hours before midnight (e.g. 21:00 for a 3h offset) | Tomorrow's full schedule. |
| **`SoonReminder`** | `offset` minutes before each prayer time | "«Prayer» in Nm". For groups with Jamaat enabled, sends a poll with a congregation delay. |
| **`ArriveReminder`** | at each prayer's exact time | "«Prayer» has arrived", replying to the matching *soon* message. |

### Trigger logic & idempotency

`ShouldTrigger` is a **pure function** of `(chat, prayerDay, now)` — no DB, no
Telegram — which is what makes it unit-testable
([`reminder_test.go`](internal/handler/reminder_test.go)).

The core rule for Soon/Arrive is:

```
lastAt < triggerTime  AND  triggerTime <= now
```

and the loop keeps the **latest** matching prayer. This is deliberate: after a
scheduler gap (downtime), stale prayers are skipped and only the most recent one
is sent, instead of a burst of back-dated reminders. After sending, `Handle`
persists `lastAt` via `UpdateReminder`, so the same reminder never re-fires.

## Concurrency

- **Per-bot:** unbounded `errgroup` (there are only a handful of bots).
- **Per-chat:** `errG.SetLimit(maxConcurrentReminderSends)` bounds goroutine
  fan-out and smooths bursts against Telegram's per-bot rate limits as the
  subscriber count grows.

## Other files

| File | Responsibility |
|------|----------------|
| [`helper.go`](internal/handler/helper.go) | Message formatting, `deleteMessages`, blocked-user detection, `now(loc)`. |
| [`languages.go`](internal/handler/languages.go) | Embedded `languages/text.yaml` → localized prayer names and templates. |
| [`log.go`](internal/handler/log.go) | Namespaced logging helpers. |
| [`internal/service/db.go`](internal/service/db.go) | `NewDB` wrapper over the shared store. |

## Cleanup behavior

If a send fails with a "forbidden" error (user blocked the bot), the chat is
deleted (`deleteChat`) so it stops consuming work on future ticks.
