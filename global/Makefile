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
	go run ./cmd/bootstrapdb
	GOOSE_TABLE=global_bot.goose_db_version goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	GOOSE_TABLE=global_bot.goose_db_version goose -dir migrations postgres "$(DATABASE_URL)" down
