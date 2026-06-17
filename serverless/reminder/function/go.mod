module github.com/escalopa/prayer-bot/reminder/function

go 1.25.0

require (
	github.com/GoogleCloudPlatform/functions-framework-go v1.9.2
	github.com/escalopa/prayer-bot v0.0.0-20260617185345-8bb53e057cb4
	github.com/escalopa/prayer-bot/reminder v0.0.0
	golang.org/x/sync v0.19.0
)

require (
	cloud.google.com/go/functions v1.19.3 // indirect
	github.com/cloudevents/sdk-go/v2 v2.15.2 // indirect
	github.com/go-telegram/bot v1.15.0 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.10.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jonboulle/clockwork v0.5.0 // indirect
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v0.0.0-20180701023420-4b7aa43c6742 // indirect
	github.com/rogpeppe/go-internal v1.15.0 // indirect
	github.com/yandex-cloud/go-genproto v0.0.0-20240819112322-98a264d392f6 // indirect
	github.com/ydb-platform/ydb-go-genproto v0.0.0-20260428144813-1c07baab7f7b // indirect
	github.com/ydb-platform/ydb-go-sdk/v3 v3.139.6 // indirect
	github.com/ydb-platform/ydb-go-yc v0.12.3 // indirect
	github.com/ydb-platform/ydb-go-yc-metadata v0.6.1 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251029180050-ab9386a59fda // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/grpc v1.78.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

exclude google.golang.org/genproto v0.0.0-20230306155012-7f2fa6fef1f4

replace github.com/escalopa/prayer-bot/reminder => ../
