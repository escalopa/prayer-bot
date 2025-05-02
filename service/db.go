package service

import (
	"context"

	"github.com/escalopa/prayer-bot/domain"

	ydb "github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/table"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/types"
)

type DB struct {
	client table.Client
}

func NewDB(ctx context.Context) (*DB, error) {
	sdk, err := ydb.Open(ctx, cfg.YDBEndpoint)
	if err != nil {
		return nil, err
	}

	client := sdk.Table()

	return &DB{client: client}, nil
}

func (db *DB) StorePrayers(ctx context.Context, botID uint8, rows []*domain.PrayerTimes) error {
	query := `
		DECLARE $items AS List<Struct<
			bot_id: Uint8,
			prayer_date: Date,
			fajr: Datetime,
			shuruq: Datetime,
			dhuhr: Datetime,
			asr: Datetime,
			maghrib: Datetime,
			isha: Datetime
		>>;

		UPSERT INTO prayers
		SELECT * FROM AS_TABLE($items)
		ON CONFLICT(bot_id, prayer_date) DO UPDATE
		SET
			fajr = EXCLUDED.fajr,
			shuruq = EXCLUDED.shuruq,
			dhuhr = EXCLUDED.dhuhr,
			asr = EXCLUDED.asr,
			maghrib = EXCLUDED.maghrib,
			isha = EXCLUDED.isha;
	`

	values := make([]types.Value, 0, len(rows))
	for _, row := range rows {
		values = append(values, types.StructValue(
			types.StructFieldValue("bot_id", types.Uint8Value(botID)),
			types.StructFieldValue("prayer_date", types.DateValueFromTime(row.Date)),
			types.StructFieldValue("fajr", types.DatetimeValueFromTime(row.Fajr)),
			types.StructFieldValue("shuruq", types.DatetimeValueFromTime(row.Fajr)),
			types.StructFieldValue("dhuhr", types.DatetimeValueFromTime(row.Dhuhr)),
			types.StructFieldValue("asr", types.DatetimeValueFromTime(row.Asr)),
			types.StructFieldValue("maghrib", types.DatetimeValueFromTime(row.Maghrib)),
			types.StructFieldValue("isha", types.DatetimeValueFromTime(row.Isha)),
		))
	}

	err := db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, _, err := s.Execute(
			ctx,
			table.TxControl(table.BeginTx(table.WithSerializableReadWrite()), table.CommitTx()),
			query,
			table.NewQueryParameters(
				table.ValueParam("$items", types.ListValue(values...)),
			),
		)
		return err
	})

	return err
}
