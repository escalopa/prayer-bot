# Operations

## Health and logs

All services expose `GET /healthz`. Application logs include update IDs or delivery keys, never full Telegram updates, coordinates, database URLs, bot tokens, webhook secrets, or Maps keys.

Useful alerts are Cloud Run 5xx rate, Cloud Tasks oldest task age, queue retry count, Scheduler failures, PostgreSQL connection errors, and Google Time Zone/Geocoding non-`OK` statuses.

## Incident triage

Start with the [engineering guide](README.md) and use the stable identifiers in
structured logs:

- Telegram webhook problems: `update_id`;
- reminder delivery problems: `delivery_key`;
- Cloud Tasks HTTP requests: service name, timestamp, status, and latency;
- profile deployment problems: the named profile operation and Telegram retry
  interval.

Do not search logs by bot token, database URL, coordinates, or full Telegram
payload.

| Symptom | Likely evidence | First document |
| --- | --- | --- |
| Duplicate reminder messages in the same minute | A sender 5xx after Telegram accepted the first message, followed by a successful Cloud Tasks retry of the same delivery key | [Reminder delivery](reminder-delivery.md) |
| `prepared statement ... already exists` (`42P05`) | pgx named statement caching used through a transaction pooler | [Runtime and deployment](runtime-and-deployment.md#database-connections) |
| `prepared statement ... does not exist` (`26000`) | The pooler moved a cached statement to a different PostgreSQL connection | [Runtime and deployment](runtime-and-deployment.md#database-connections) |
| `invalid input syntax for type json` (`22P02`) during profile or outbox writes | A JSONB value was passed as Go `[]byte` while pgx used `QueryExecModeExec` | [Runtime and deployment](runtime-and-deployment.md#database-connections) |
| Reminder is late but eventually arrives | Cloud Run cold start, queue backoff, or transient sender 5xx | [Reminder delivery](reminder-delivery.md#retry-configuration) |
| Old notification remains | Immediate Telegram deletion failed and its durable deletion task is retrying or expired past Telegram's limit | [Reminder delivery](reminder-delivery.md#cleanup-categories) |
| Existing schedules work but location update fails | Google Time Zone or Geocoding failure | [Maps failure mode](#maps-failure-mode) |
| Mini App says to open it in Telegram | Missing, expired, or invalid signed Telegram init data | [Request flows](request-flows.md#mini-app-session-and-api) |
| Profile sync reports an empty or malformed Telegram response | The deploy command retries the idempotent sync three times, then skips only the cosmetic profile step | [Runtime and deployment](runtime-and-deployment.md#deployment-workflow) |

For a duplicate reminder, reconstruct the sequence in this order:

1. Find sender requests near the displayed local time.
2. Convert the displayed time with the profile's IANA timezone.
3. Group application errors by `delivery_key`.
4. Determine whether the failure happened before or after `sendMessage`.
5. Look for the subsequent Cloud Tasks retry and successful HTTP 204.
6. Check deletion-task requests for the same period.

An error labelled `complete delivery` is post-send: Telegram already returned a
message ID, but PostgreSQL did not commit it. Normal slot cleanup cannot discover
that uncommitted message; the sender needs compensating deletion before it
returns a retryable error.

## Secret rotation

- Bot token: add a Secret Manager version, deploy a new revision, then revoke the old token through BotFather if required.
- Webhook secret: add a version, deploy, and rerun `cmd/botprofile` with the same new value.
- Maps key: let Terraform replace/rotate the key and secret version; deploy the webhook revision before deleting the prior version.
- Database URL: update the sensitive Terraform input and apply so the managed secret gets a new version.

Cloud Run secret references use `latest`; a new revision is still recommended after rotation so startup behavior is explicit and auditable.

## Telegram profile synchronization

The final deployment step runs `cmd/botprofile`. It registers both message and callback-query webhook updates, installs one stable default bot name, description, and command menu, removes localized profile variants left by earlier releases, configures the default chat menu button to open `${WEBHOOK_URL}/app/`, and manages the embedded avatar. Before mutating profile text, commands, or the menu button, it reads the current Telegram value and skips an update when the value already matches. It also downloads the current avatar and compares a recompression-tolerant visual hash with the embedded image, uploading only when the visible photo changed.

Telegram profile changes are cosmetic and rate-limited independently from webhook registration. If Telegram returns `429 Too Many Requests` during profile synchronization, the command reports the requested retry interval and exits successfully. The deployment summary records the skipped profile update; rerun the deployment after that interval to apply any remaining profile change. Other errors, including invalid tokens, invalid webhook configuration, or malformed profile values, remain fatal.

The language selected inside the bot is per chat. It changes messages, reply keyboards, reminders, dates, and the Mini App, but never calls Telegram's global profile methods. Telegram profile localization is based on the viewer's Telegram client language rather than this saved preference, so the production profile intentionally uses one stable public identity.

After deployment, verify the workflow summary's Mini App URL returns HTTP 200, open a private chat with the selected testing bot, and tap the menu button next to the message field. Check initial location setup, today/tomorrow switching, a settings save, and each reminder toggle. Opening `/app/` in a normal browser should show the “open inside Telegram” state; Mini App API requests without valid signed Telegram init data should return HTTP 401.

The menu button is configured automatically through the Bot API. BotFather's optional “Main Mini App” profile launch button is separate and may be configured manually later if a second entry point on the bot profile is wanted; it is not required for this release.

The welcome illustration is embedded in the webhook binary and sent with the localized `/start` caption. Updating either JPEG requires a normal application deployment; updating an already configured avatar also requires removing the old photo in Telegram before rerunning profile synchronization.

## Database recovery

Set `GLOBAL_DB_SCHEMA` to `global_bot_testing` or `global_bot_production`, then run Goose with `-table="${GLOBAL_DB_SCHEMA}.goose_db_version"`. Never run global migrations with the legacy default migration table. The initial down migration drops only the selected global schema, but production rollback should normally use a forward corrective migration rather than dropping user data.

Migration `00002` adds the per-chat Hijri correction and the two weekly reminder kinds. Migration `00003` adds notification message slots and scheduled deletion tasks for pre-prayer/category cleanup. Migration `00004` adds revocable private rolling-calendar subscriptions. Migration `00005` adds the three opt-in Islamic occasion rule kinds and their shared notification cleanup category. The normal global deployment runs migrations before the new webhook and sender revisions are applied.

Telegram only deletes messages younger than 48 hours. The sender schedules every reminder for cleanup after 36 hours, while a new message in the same category also triggers immediate best-effort deletion of its predecessor. If direct deletion fails transiently, the durable cleanup task retries through the existing Cloud Tasks queue.

## Maps failure mode

Existing profiles and prayer calculations continue working if Google Maps is unavailable. Only new location setup or a location change fails, and Telegram retries the webhook after a server error. No Maps API is called for `/today`, `/tomorrow`, `/next`, or reminder delivery.

The same behavior applies to the Mini App: loading an existing schedule and saving calculation/reminder settings do not call Maps. Only an explicit location update calls the Time Zone and Geocoding APIs.

## Calendar subscriptions

The private `.ics` URL is a bearer credential. Do not add it to application
logs, support tickets, screenshots, or public documentation, and restrict
access to platform request logs that may contain requested URLs. A user who
accidentally shares it should press **Disconnect calendar** and reconnect; this
disables the old token and issues a new one.

The feed contains today's prayer times, the following 29 days, and Islamic
occasions that fall within the same corrected local-date window.
It does not need Cloud Scheduler or a background sync job. Google Calendar
chooses when to refetch URL subscriptions and can lag behind local midnight or a
settings change. The feed advertises a 12-hour refresh interval, but that value
is only a hint.

If Google does not show events:

1. Confirm the feed URL returns HTTP 200 and `Content-Type: text/calendar`.
2. Confirm the body has no UTF-8 byte-order mark and uses CRLF line endings.
3. Check that the subscription is still enabled.
4. Unfold RFC 5545 continuation lines before searching logs or test output for a
   long occasion UID or source URL.
5. Add it from Google Calendar on a computer with **Other calendars → From
   URL** using the Mini App's copy-link fallback.
6. Allow for Google's refresh cache before recreating the subscription.

## Feedback delivery

`GLOBAL_OWNER_ID` is also the destination for feedback and bug reports. The owner must open the selected testing or production bot and send `/start` at least once, because Telegram does not allow a bot to initiate a conversation with a user who has never contacted it. Users open the localized feedback prompt from the persistent keyboard or `/feedback`, then reply with text, media, or a screenshot. Group submissions are redirected to a private chat so reports and user identity are not exposed to group members.

The application does not persist feedback content. Telegram delivers an owner-only context message and a copy of the user's submission. Delivery errors are logged without the feedback content.
