module github.com/escalopa/prayer-bot/dispatcher

go 1.25.0

require (
	github.com/GoogleCloudPlatform/functions-framework-go v1.9.2
	github.com/escalopa/prayer-bot v0.0.0-00010101000000-000000000000
	github.com/go-telegram/bot v1.21.0
	golang.org/x/sync v0.19.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloud.google.com/go/functions v1.19.3 // indirect
	github.com/cloudevents/sdk-go/v2 v2.15.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.10.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v0.0.0-20180701023420-4b7aa43c6742 // indirect
	github.com/rogpeppe/go-internal v1.15.0 // indirect
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.10.0 // indirect
	golang.org/x/text v0.32.0 // indirect
)

replace github.com/escalopa/prayer-bot => ../..
