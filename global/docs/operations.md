# Operations

## Health and logs

All services expose `GET /healthz`. Application logs include update IDs or delivery keys, never full Telegram updates, coordinates, database URLs, bot tokens, webhook secrets, or Maps keys.

Useful alerts are Cloud Run 5xx rate, Cloud Tasks oldest task age, queue retry count, Scheduler failures, PostgreSQL connection errors, and Google Time Zone/Geocoding non-`OK` statuses.

## Secret rotation

- Bot token: add a Secret Manager version, deploy a new revision, then revoke the old token through BotFather if required.
- Webhook secret: add a version, deploy, and rerun `cmd/botprofile` with the same new value.
- Maps key: let Terraform replace/rotate the key and secret version; deploy the webhook revision before deleting the prior version.
- Database URL: update the sensitive Terraform input and apply so the managed secret gets a new version.

Cloud Run secret references use `latest`; a new revision is still recommended after rotation so startup behavior is explicit and auditable.

## Telegram profile synchronization

The final deployment step runs `cmd/botprofile`. It registers both message and callback-query webhook updates, installs the default and localized command menus, names and descriptions, and uploads the embedded avatar when the bot has no profile photo. Re-running the deployment is safe; an existing avatar is left in place.

The welcome illustration is embedded in the webhook binary and sent with the localized `/start` caption. Updating either JPEG requires a normal application deployment; updating an already configured avatar also requires removing the old photo in Telegram before rerunning profile synchronization.

## Database recovery

Set `GLOBAL_DB_SCHEMA` to `global_bot_testing` or `global_bot_production`, then run Goose with `-table="${GLOBAL_DB_SCHEMA}.goose_db_version"`. Never run global migrations with the legacy default migration table. The initial down migration drops only the selected global schema, but production rollback should normally use a forward corrective migration rather than dropping user data.

## Maps failure mode

Existing profiles and prayer calculations continue working if Google Maps is unavailable. Only new location setup or a location change fails, and Telegram retries the webhook after a server error. No Maps API is called for `/today`, `/tomorrow`, `/next`, or reminder delivery.
