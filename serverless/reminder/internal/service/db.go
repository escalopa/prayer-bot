package service

import (
	"context"
	"os"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/result"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/types"

	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/table"
	yc "github.com/ydb-platform/ydb-go-yc"
)

var (
	readTx = table.TxControl(
		table.BeginTx(table.WithOnlineReadOnly()),
		table.CommitTx(),
	)

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

func (db *DB) GetChatsByIDs(ctx context.Context, botID int64, chatIDs []int64) (chats []*domain.Chat, _ error) {
	query := `
		DECLARE $bot_id AS Int64;
		DECLARE $chat_ids AS List<Int64>;

		SELECT bot_id, chat_id, state, language_code, reminder_message_id
		FROM chats
		WHERE bot_id = $bot_id AND chat_id IN $chat_ids;
	`

	values := make([]types.Value, len(chatIDs))
	for i, chatID := range chatIDs {
		values[i] = types.Int64Value(chatID)
	}

	params := table.NewQueryParameters(
		table.ValueParam("$bot_id", types.Int64Value(botID)),
		table.ValueParam("$chat_ids", types.ListValue(values...)),
	)

	err := db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, res, err := s.Execute(ctx, readTx, query, params)
		if err != nil {
			return err
		}

		defer func(res result.Result) { _ = res.Close() }(res)
		if res.NextResultSet(ctx) {
			for res.NextRow() {
				chat := &domain.Chat{}
				err = res.ScanWithDefaults(
					&chat.BotID,
					&chat.ChatID,
					&chat.State,
					&chat.LanguageCode,
					&chat.ReminderMessageID,
				)
				if err != nil {
					return err
				}
				chats = append(chats, chat)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return chats, nil
}

func (db *DB) GetSubscribers(ctx context.Context, botID int64) (chatIDs []int64, _ error) {
	query := `
		DECLARE $bot_id AS Int64;

		SELECT chat_id
		FROM chats
		WHERE bot_id = $bot_id AND subscribed = true;
	`

	params := table.NewQueryParameters(table.ValueParam("$bot_id", types.Int64Value(botID)))
	err := db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, res, err := s.Execute(ctx, readTx, query, params)
		if err != nil {
			return err
		}

		defer func(res result.Result) { _ = res.Close() }(res)
		for res.NextResultSet(ctx) {
			for res.NextRow() {
				var chatID int64
				err = res.ScanWithDefaults(&chatID)
				if err != nil {
					return err
				}
				chatIDs = append(chatIDs, chatID)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return chatIDs, nil
}

func (db *DB) GetSubscribersByOffset(ctx context.Context, botID int64, offset int32) (chatIDs []int64, _ error) {
	query := `
		DECLARE $bot_id AS Int64;
		DECLARE $offset AS Int32;

		SELECT chat_id
		FROM chats
		WHERE bot_id = $bot_id AND subscribed = true AND reminder_offset = $offset;
	`

	params := table.NewQueryParameters(
		table.ValueParam("$bot_id", types.Int64Value(botID)),
		table.ValueParam("$offset", types.Int32Value(offset)),
	)

	err := db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, res, err := s.Execute(ctx, readTx, query, params)
		if err != nil {
			return err
		}

		defer func(res result.Result) { _ = res.Close() }(res)
		for res.NextResultSet(ctx) {
			for res.NextRow() {
				var chatID int64
				err = res.ScanWithDefaults(&chatID)
				if err != nil {
					return err
				}
				chatIDs = append(chatIDs, chatID)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return chatIDs, nil
}

func (db *DB) GetPrayerDay(ctx context.Context, botID int64, date time.Time) (prayerDay *domain.PrayerDay, _ error) {
	query := `
		DECLARE $bot_id AS Int64;
		DECLARE $date AS Date;

		SELECT prayer_date, fajr, shuruq, dhuhr, asr, maghrib, isha
		FROM prayers
		WHERE bot_id = $bot_id AND prayer_date = $date;
	`

	params := table.NewQueryParameters(
		table.ValueParam("$bot_id", types.Int64Value(botID)),
		table.ValueParam("$date", types.DateValueFromTime(date)),
	)

	err := db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, res, err := s.Execute(ctx, readTx, query, params)
		if err != nil {
			return err
		}

		defer func(res result.Result) { _ = res.Close() }(res)
		if res.NextResultSet(ctx) && res.NextRow() {
			prayerDay = &domain.PrayerDay{}
			err = res.ScanWithDefaults(
				&prayerDay.Date,
				&prayerDay.Fajr, &prayerDay.Shuruq,
				&prayerDay.Dhuhr, &prayerDay.Asr,
				&prayerDay.Maghrib, &prayerDay.Isha,
			)
			if err != nil {
				return err
			}
			return nil
		}

		return domain.ErrNotFound
	})

	if err != nil {
		if ydb.IsOperationError(err, Ydb.StatusIds_NOT_FOUND) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return prayerDay, nil
}

func (db *DB) SetReminderMessageID(ctx context.Context, botID int64, chatID int64, reminderMessageID int32) error {
	query := `
		DECLARE $bot_id AS Int64;
		DECLARE $chat_id AS Int64;
		DECLARE $reminder_message_id AS Int32;

		UPDATE chats
		SET reminder_message_id = $reminder_message_id
		WHERE bot_id = $bot_id AND chat_id = $chat_id;
	`

	params := table.NewQueryParameters(
		table.ValueParam("$bot_id", types.Int64Value(botID)),
		table.ValueParam("$chat_id", types.Int64Value(chatID)),
		table.ValueParam("$reminder_message_id", types.Int32Value(reminderMessageID)),
	)

	err := db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, _, err := s.Execute(ctx, writeTx, query, params)
		return err
	})

	return err
}
