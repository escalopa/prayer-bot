# Global bot engineering guide

This directory is the durable map of the global prayer bot. Start here before
tracing source code or changing infrastructure.

The global bot is intentionally independent from the two legacy city bots.
Unless a document explicitly says otherwise, paths and database objects in this
guide belong to `global/` and the `global_bot_testing` or
`global_bot_production` PostgreSQL schema.

## Choose the document for the task

| Task or question | Read first |
| --- | --- |
| Understand the complete system and its trust boundaries | [Architecture](architecture.md) |
| Find the package, binary, or API endpoint to change | [Code map](code-map.md) |
| Follow a Telegram update, Mini App request, or location change | [Request flows](request-flows.md) |
| Understand tables, ownership, and data invariants | [Data model](data-model.md) |
| Change reminders, retries, idempotency, or message deletion | [Reminder delivery](reminder-delivery.md) |
| Change Cloud Run, Cloud Tasks, Scheduler, secrets, or environments | [Runtime and deployment](runtime-and-deployment.md) |
| Diagnose an incident or perform an operational action | [Operations](operations.md) |

## Five-minute mental model

1. Telegram sends bot updates to the public `webhook` Cloud Run service.
2. The same service embeds the Mini App and exposes its authenticated API.
3. Profiles, preferences, reminder rules, schedules, delivery state, and update
   idempotency live in an environment-specific PostgreSQL schema.
4. Prayer times, Hijri dates, Qibla direction, and calendar files are calculated
   locally. Google APIs are used only when a location changes.
5. Cloud Scheduler invokes `dispatch`, which atomically claims due schedules and
   writes an outbox.
6. `dispatch` creates deterministic Cloud Tasks. The private `sender` service
   leases each delivery, sends through Telegram, advances the schedule, and
   manages deletion of older notification messages.
7. Testing and production reuse the same GCP project and PostgreSQL database but
   use different Cloud resources, bot tokens, secrets, Terraform state, and
   database schemas.

## Documentation rules

- Update the relevant document in the same pull request as an architectural,
  deployment, persistence, or delivery-semantics change.
- Describe invariants and failure behavior, not line-by-line implementation.
- Link to the owning package instead of duplicating source code.
- Never place tokens, database URLs, coordinates, Telegram IDs, or other
  production values in these documents.
- If observed production behavior differs from these documents, treat that as
  either a code defect or a documentation defect and correct both.
