package service

import (
	"context"
	"os"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/table"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/types"
	yc "github.com/ydb-platform/ydb-go-yc"
)

var (
	writeTx = table.TxControl(
		table.BeginTx(table.WithSerializableReadWrite()),
		table.CommitTx(),
	)
)

type DB struct {
	client table.Client
}

func NewDB(ctx context.Context) (*DB, error) {
	ydbEndpoint := os.Getenv("YDB_ENDPOINT")

	sdk, err := ydb.Open(ctx, ydbEndpoint,
		yc.WithMetadataCredentials(),
		yc.WithInternalCA(),
	)
	if err != nil {
		return nil, err
	}

	return &DB{client: sdk.Table()}, nil
}

func (db *DB) SetPrayerDays(ctx context.Context, botID int64, rows []*domain.PrayerDay) error {
	query := `
		DECLARE $items AS List<Struct<
			bot_id: Int64,
			prayer_date: Date,
			fajr: Datetime,
			shuruq: Datetime,
			dhuhr: Datetime,
			asr: Datetime,
			maghrib: Datetime,
			isha: Datetime
		>>;

		UPSERT INTO prayers
		SELECT * FROM AS_TABLE($items);
	`

	values := make([]types.Value, len(rows))
	for i, row := range rows {
		values[i] = types.StructValue(
			types.StructFieldValue("bot_id", types.Int64Value(botID)),
			types.StructFieldValue("prayer_date", types.DateValueFromTime(row.Date)),
			types.StructFieldValue("fajr", types.DatetimeValueFromTime(row.Fajr)),
			types.StructFieldValue("shuruq", types.DatetimeValueFromTime(row.Shuruq)),
			types.StructFieldValue("dhuhr", types.DatetimeValueFromTime(row.Dhuhr)),
			types.StructFieldValue("asr", types.DatetimeValueFromTime(row.Asr)),
			types.StructFieldValue("maghrib", types.DatetimeValueFromTime(row.Maghrib)),
			types.StructFieldValue("isha", types.DatetimeValueFromTime(row.Isha)),
		)
	}

	params := table.NewQueryParameters(table.ValueParam("$items", types.ListValue(values...)))
	err := db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, _, err := s.Execute(ctx, writeTx, query, params)
		return err
	})

	return err
}
