package service

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/escalopa/prayer-bot/log"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb"
	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/table"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/result"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/types"
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

		SELECT bot_id, chat_id, state, language_code, is_group, reminder
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
				var reminderJSON string
				err = res.ScanWithDefaults(
					&chat.BotID,
					&chat.ChatID,
					&chat.State,
					&chat.LanguageCode,
					&chat.IsGroup,
					&reminderJSON,
				)
				if err != nil {
					return err
				}

				var reminder domain.Reminder
				if err := json.Unmarshal([]byte(reminderJSON), &reminder); err != nil {
					log.Error("unmarshal reminder json", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
					return domain.ErrUnmarshalJSON
				}
				chat.Reminder = &reminder

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

func (db *DB) GetPrayerDay(ctx context.Context, botID int64, date time.Time) (prayerDay *domain.PrayerDay, _ error) {
	nextDate := date.Add(24 * time.Hour)

	query := `
		DECLARE $bot_id AS Int64;
		DECLARE $date AS Date;
		DECLARE $next_date AS Date;

		SELECT prayer_date, fajr, shuruq, dhuhr, asr, maghrib, isha
		FROM prayers
		WHERE bot_id = $bot_id AND prayer_date IN ($date, $next_date)
		ORDER BY prayer_date;
	`

	params := table.NewQueryParameters(
		table.ValueParam("$bot_id", types.Int64Value(botID)),
		table.ValueParam("$date", types.DateValueFromTime(date)),
		table.ValueParam("$next_date", types.DateValueFromTime(nextDate)),
	)

	err := db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, res, err := s.Execute(ctx, readTx, query, params)
		if err != nil {
			return err
		}

		defer func(res result.Result) { _ = res.Close() }(res)
		if res.NextResultSet(ctx) {
			if res.NextRow() {
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
			} else {
				return domain.ErrNotFound
			}

			if res.NextRow() {
				nextDay := &domain.PrayerDay{}
				err = res.ScanWithDefaults(
					&nextDay.Date,
					&nextDay.Fajr, &nextDay.Shuruq,
					&nextDay.Dhuhr, &nextDay.Asr,
					&nextDay.Maghrib, &nextDay.Isha,
				)
				if err != nil {
					return err
				}
				prayerDay.NextDay = nextDay
			} else {
				return domain.ErrNotFound // no next day found (cannot happen)
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

func (db *DB) UpdateReminder(
	ctx context.Context,
	botID int64,
	chatID int64,
	reminderType domain.ReminderType,
	messageID int,
	lastAt time.Time,
) error {
	lastAtNanos := lastAt.UnixNano()

	query := `
		DECLARE $bot_id AS Int64;
		DECLARE $chat_id AS Int64;
		DECLARE $reminder_type AS Utf8;
		DECLARE $message_id AS Int64;
		DECLARE $last_at AS Int64;

		UPDATE chats
		SET reminder = Json_SetField(
			Json_SetField(reminder, $reminder_type || ".message_id", CAST($message_id AS Json)),
			$reminder_type || ".last_at",
			CAST($last_at AS Json)
		)
		WHERE bot_id = $bot_id AND chat_id = $chat_id;
	`

	params := table.NewQueryParameters(
		table.ValueParam("$bot_id", types.Int64Value(botID)),
		table.ValueParam("$chat_id", types.Int64Value(chatID)),
		table.ValueParam("$reminder_type", types.UTF8Value(reminderType.String())),
		table.ValueParam("$message_id", types.Int64Value(int64(messageID))),
		table.ValueParam("$last_at", types.Int64Value(lastAtNanos)),
	)

	err := db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, _, err := s.Execute(ctx, writeTx, query, params)
		return err
	})

	return err
}

func (db *DB) DeleteChat(ctx context.Context, botID int64, chatID int64) error {
	query := `
		DECLARE $bot_id AS Int64;
		DECLARE $chat_id AS Int64;

		DELETE FROM chats
		WHERE bot_id = $bot_id AND chat_id = $chat_id;
	`

	params := table.NewQueryParameters(
		table.ValueParam("$bot_id", types.Int64Value(botID)),
		table.ValueParam("$chat_id", types.Int64Value(chatID)),
	)

	err := db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, _, err := s.Execute(ctx, writeTx, query, params)
		return err
	})

	return err
}
