.PHONY: test check build migrate-up migrate-down

test:
	go test ./...

check:
	gofmt -w cmd internal
	go vet ./...
	go test ./...

build:
	go build ./cmd/...

migrate-up:
	test -n "$(GLOBAL_DB_SCHEMA)"
	GLOBAL_DB_SCHEMA="$(GLOBAL_DB_SCHEMA)" go run ./cmd/bootstrapdb
	GLOBAL_DB_SCHEMA="$(GLOBAL_DB_SCHEMA)" goose -dir migrations -table="$(GLOBAL_DB_SCHEMA).goose_db_version" postgres "$(DATABASE_URL)" up

migrate-down:
	test -n "$(GLOBAL_DB_SCHEMA)"
	GLOBAL_DB_SCHEMA="$(GLOBAL_DB_SCHEMA)" goose -dir migrations -table="$(GLOBAL_DB_SCHEMA).goose_db_version" postgres "$(DATABASE_URL)" down
