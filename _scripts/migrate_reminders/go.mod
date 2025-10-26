module github.com/escalopa/prayer-bot/scripts/migrate_reminders

go 1.21

require (
	github.com/escalopa/prayer-bot v0.0.0
	github.com/ydb-platform/ydb-go-genproto v0.0.0-20240920120314-0fed943b0136
	github.com/ydb-platform/ydb-go-sdk/v3 v3.82.3
	github.com/ydb-platform/ydb-go-yc v0.12.1
)

replace github.com/escalopa/prayer-bot => ../..
