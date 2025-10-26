package service

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/escalopa/prayer-bot/log"
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
		log.Error("failed to open ydb connection", log.Err(err))
		return nil, domain.ErrInternal
	}

	return &DB{client: sdk.Table()}, nil
}

func (db *DB) GetChatsByIDs(ctx context.Context, botID int64, chatIDs []int64) (chats []*domain.Chat, _ error) {
	query := `
		DECLARE $bot_id AS Int64;
		DECLARE $chat_ids AS List<Int64>;

		SELECT bot_id, chat_id, state, language_code, reminder
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
			log.Error("execute get chats by ids query", log.Err(err), log.BotID(botID))
			return domain.ErrInternal
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
					&reminderJSON,
				)
				if err != nil {
					log.Error("scan chat fields", log.Err(err), log.BotID(botID))
					return domain.ErrInternal
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
			log.Error("execute get subscribers query", log.Err(err), log.BotID(botID))
			return domain.ErrInternal
		}

		defer func(res result.Result) { _ = res.Close() }(res)
		for res.NextResultSet(ctx) {
			for res.NextRow() {
				var chatID int64
				err = res.ScanWithDefaults(&chatID)
				if err != nil {
					log.Error("scan subscriber chat id", log.Err(err), log.BotID(botID))
					return domain.ErrInternal
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
			log.Error("execute get prayer day query", log.Err(err), log.BotID(botID))
			return domain.ErrInternal
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
					log.Error("scan prayer day fields", log.Err(err), log.BotID(botID))
					return domain.ErrInternal
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
					log.Error("scan next prayer day fields", log.Err(err), log.BotID(botID))
					return domain.ErrInternal
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
	err := db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		txBegin := table.TxControl(table.BeginTx(table.WithSerializableReadWrite()))

		selectQuery := `
			DECLARE $bot_id AS Int64;
			DECLARE $chat_id AS Int64;

			SELECT reminder
			FROM chats
			WHERE bot_id = $bot_id AND chat_id = $chat_id;
		`

		selectParams := table.NewQueryParameters(
			table.ValueParam("$bot_id", types.Int64Value(botID)),
			table.ValueParam("$chat_id", types.Int64Value(chatID)),
		)

		txID, res, err := s.Execute(ctx, txBegin, selectQuery, selectParams)
		if err != nil {
			log.Error("execute select query", log.Err(err), log.BotID(botID), log.ChatID(chatID))
			return domain.ErrInternal
		}

		defer func(res result.Result) { _ = res.Close() }(res)

		var reminderJSON string
		if res.NextResultSet(ctx) && res.NextRow() {
			err = res.ScanWithDefaults(&reminderJSON)
			if err != nil {
				log.Error("scan reminder json", log.Err(err), log.BotID(botID), log.ChatID(chatID))
				return domain.ErrInternal
			}
		} else {
			return domain.ErrNotFound
		}

		var reminder domain.Reminder
		if err := json.Unmarshal([]byte(reminderJSON), &reminder); err != nil {
			log.Error("unmarshal reminder json", log.Err(err), log.BotID(botID), log.ChatID(chatID))
			return domain.ErrUnmarshalJSON
		}

		switch reminderType {
		case domain.ReminderTypeToday:
			reminder.Today.MessageID = messageID
			reminder.Today.LastAt = lastAt
		case domain.ReminderTypeSoon:
			reminder.Soon.MessageID = messageID
			reminder.Soon.LastAt = lastAt
		case domain.ReminderTypeArrive:
			reminder.Arrive.MessageID = messageID
			reminder.Arrive.LastAt = lastAt
		}

		updatedReminderJSON, err := json.Marshal(&reminder)
		if err != nil {
			log.Error("marshal updated reminder json", log.Err(err), log.BotID(botID), log.ChatID(chatID))
			return domain.ErrInternal
		}

		updateQuery := `
			DECLARE $bot_id AS Int64;
			DECLARE $chat_id AS Int64;
			DECLARE $reminder AS Json;

			UPDATE chats
			SET reminder = $reminder
			WHERE bot_id = $bot_id AND chat_id = $chat_id;
		`

		updateParams := table.NewQueryParameters(
			table.ValueParam("$bot_id", types.Int64Value(botID)),
			table.ValueParam("$chat_id", types.Int64Value(chatID)),
			table.ValueParam("$reminder", types.JSONValue(string(updatedReminderJSON))),
		)

		txCommit := table.TxControl(table.WithTx(txID), table.CommitTx())
		_, _, err = s.Execute(ctx, txCommit, updateQuery, updateParams)
		if err != nil {
			log.Error("execute update reminder query", log.Err(err), log.BotID(botID), log.ChatID(chatID))
			return domain.ErrInternal
		}
		return nil
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
		if err != nil {
			log.Error("execute delete chat query", log.Err(err), log.BotID(botID), log.ChatID(chatID))
			return domain.ErrInternal
		}
		return nil
	})

	return err
}
