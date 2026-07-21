# Testing

This document defines how the global bot is tested and how to add tests for new
work. It is the owning document for test strategy; update it in the same pull
request when the strategy changes.

## Principles

- **Never read the wall clock in business logic.** Time is a dependency. Planning,
  dispatch, and store queries already take an explicit `now`/`after time.Time`
  parameter, and the sender takes an injectable `now func() time.Time` (default
  `time.Now`). This is how "real-time execution" is simulated: a test passes the
  instant it wants instead of sleeping. Do not introduce a bare `time.Now()` in
  `internal/*` business logic — thread the instant through, or inject a clock.
- **Isolate collaborators behind interfaces.** Each orchestrator declares the
  narrow store/bot/enqueuer interface it needs (`DispatchStore`, `PlanningStore`,
  `SenderStore`, `MessageSender`, `TaskEnqueuer`, `nextPlanner`). The concrete
  `*store.Store`, `*Planner`, and Telegram client satisfy them in production;
  hand-written fakes satisfy them in tests. No mocking framework is used.
- **Standard library only.** Tests use `testing` and table-driven subtests. Keep
  fakes small, local to the package, and named `fake*`.
- **Deterministic.** No sleeps, no network, no randomness in unit tests. Fixed
  timezones and dates make prayer/reminder assertions reproducible.

## Test layers

| Layer | Scope | Depends on | Runs in `make test` |
| --- | --- | --- | --- |
| Unit | One package, pure logic and orchestration over fakes | Nothing external | Always |
| Integration | Real SQL through `internal/store` against PostgreSQL | `TEST_DATABASE_URL` | Only when the variable is set |

### Unit tests

Most coverage is unit level. Examples worth copying:

- `internal/reminders/planner_test.go` — a `fixedCalculator` fake and explicit
  `after` instants assert next-occurrence timing across weekly, occasion, and
  before-prayer rules.
- `internal/reminders/sender_process_test.go` — fakes for `SenderStore`,
  `nextPlanner`, and `MessageSender`, plus an injected clock, cover the full
  delivery state machine: send, complete, staleness, lease contention, and the
  post-send compensation described in [Reminder delivery](reminder-delivery.md).
- `internal/domain/domain_test.go` — pure validation and value-type behavior.

### Integration tests

`internal/store/integration_test.go` exercises real SQL because the parts that
break under the Supabase transaction pooler — `pgx.QueryExecModeExec`, JSONB
passed as text, and the transactional outbox — are invisible to fakes. A missing
integration test is exactly why the JSONB-as-`[]byte` profile-save bug reached
production; `TestIntegrationProfileRoundTripPreservesJSONBAdjustments` now guards
that incident.

The harness reads every goose `Up` section from `migrations/`, substitutes
`${GLOBAL_DB_SCHEMA}`, and applies it to a freshly recreated
`global_bot_testing` schema. It skips automatically when `TEST_DATABASE_URL` is
unset, so `make test` stays fast and dependency-free.

Run against a disposable database:

```sh
docker run -d --name pb-pg-test -e POSTGRES_PASSWORD=postgres -p 55432:5432 postgres:16-alpine
export TEST_DATABASE_URL='postgres://postgres:postgres@localhost:55432/postgres?sslmode=disable'
go test ./internal/store/
docker rm -f pb-pg-test
```

> The harness drops and recreates the `global_bot_testing` schema. Never point
> `TEST_DATABASE_URL` at a database that holds real data.

## Adding tests for new work

1. If the change adds business logic, keep time and I/O injectable and cover the
   logic with unit tests over fakes.
2. If the change adds or alters SQL, add a store integration test that round-trips
   the data, especially any JSONB column or transactional flow.
3. If the change alters delivery, planning, or cleanup semantics, update the
   assertions here and in [Reminder delivery](reminder-delivery.md) together.
