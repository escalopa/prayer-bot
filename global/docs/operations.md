# Operations

## Health and logs

All services expose `GET /healthz`. Application logs include update IDs or delivery keys, never full Telegram updates, coordinates, database URLs, bot tokens, webhook secrets, or Maps keys.

Useful alerts are Cloud Run 5xx rate, Cloud Tasks oldest task age, queue retry count, Scheduler failures, PostgreSQL connection errors, and Google Time Zone/Geocoding non-`OK` statuses.

The webhook, dispatch, and sender default to zero minimum instances, so they incur no fixed idle-instance charge. The first interactive request after inactivity may wait for a cold start. A deployment can explicitly set `webhook_min_instances` to `1` later if lower first-response latency becomes worth the additional monthly cost.

## Secret rotation

- Bot token: add a Secret Manager version, deploy a new revision, then revoke the old token through BotFather if required.
- Webhook secret: add a version, deploy, and rerun `cmd/botprofile` with the same new value.
- Maps key: let Terraform replace/rotate the key and secret version; deploy the webhook revision before deleting the prior version.
- Database URL: update the sensitive Terraform input and apply so the managed secret gets a new version.

Cloud Run secret references use `latest`; a new revision is still recommended after rotation so startup behavior is explicit and auditable.

## Telegram profile synchronization

The final deployment step runs `cmd/botprofile`. It registers both message and callback-query webhook updates, installs one stable default bot name, description, and command menu, removes localized profile variants left by earlier releases, configures the default chat menu button to open `${WEBHOOK_URL}/app/`, and uploads the embedded avatar when the bot has no profile photo. Re-running the deployment is safe; an existing avatar is left in place. The URL is derived by the workflow, so there is no additional GitHub secret or variable.

The language selected inside the bot is per chat. It changes messages, reply keyboards, reminders, dates, and the Mini App, but never calls Telegram's global profile methods. Telegram profile localization is based on the viewer's Telegram client language rather than this saved preference, so the production profile intentionally uses one stable public identity.

After deployment, verify the workflow summary's Mini App URL returns HTTP 200, open a private chat with the selected testing bot, and tap the menu button next to the message field. Check initial location setup, today/tomorrow switching, a settings save, and each reminder toggle. Opening `/app/` in a normal browser should show the “open inside Telegram” state; Mini App API requests without valid signed Telegram init data should return HTTP 401.

The menu button is configured automatically through the Bot API. BotFather's optional “Main Mini App” profile launch button is separate and may be configured manually later if a second entry point on the bot profile is wanted; it is not required for this release.

The welcome illustration is embedded in the webhook binary and sent with the localized `/start` caption. Telegram's empty-chat description supports text only, so the bot avatar is the image shown with that profile section; there is no separate description-image Bot API field. Updating either JPEG requires a normal application deployment; updating an already configured avatar also requires removing the old photo in Telegram before rerunning profile synchronization.

## Database recovery

Set `GLOBAL_DB_SCHEMA` to `global_bot_testing` or `global_bot_production`, then run Goose with `-table="${GLOBAL_DB_SCHEMA}.goose_db_version"`. Never run global migrations with the legacy default migration table. The initial down migration drops only the selected global schema, but production rollback should normally use a forward corrective migration rather than dropping user data.

Migration `00002` adds the per-chat Hijri correction and the two weekly reminder kinds. The normal global deployment runs it before the new webhook and sender revisions are applied, so old revisions never see reminder kinds they do not understand.

## Maps failure mode

Existing profiles and prayer calculations continue working if Google Maps is unavailable. Only new location setup or a location change fails, and Telegram retries the webhook after a server error. No Maps API is called for `/today`, `/tomorrow`, `/next`, or reminder delivery.

The same behavior applies to the Mini App: loading an existing schedule and saving calculation/reminder settings do not call Maps. Only an explicit location update calls the Time Zone and Geocoding APIs.

## Feedback delivery

`GLOBAL_OWNER_ID` is also the destination for feedback and bug reports. The owner must open the selected testing or production bot and send `/start` at least once, because Telegram does not allow a bot to initiate a conversation with a user who has never contacted it. Users open the localized feedback prompt from the persistent keyboard or `/feedback`, then reply with text, media, or a screenshot. Group submissions are redirected to a private chat so reports and user identity are not exposed to group members.

The application does not persist feedback content. Telegram delivers an owner-only context message and a copy of the user's submission. Delivery errors are logged without the feedback content.
