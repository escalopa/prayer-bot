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

//revive:disable:argument-limit
func (db *DB) CreateChat(
	ctx context.Context,
	botID int64,
	chatID int64,
	languageCode string,
	state string,
	reminder *domain.Reminder,
) error {

	reminderJSON, err := json.Marshal(reminder)
	if err != nil {
		log.Error("marshal reminder json", log.Err(err), log.BotID(botID), log.ChatID(chatID))
		return domain.ErrInternal
	}

	query := `
		DECLARE $bot_id AS Int64;
		DECLARE $chat_id AS Int64;
		DECLARE $language_code AS Utf8;
		DECLARE $state AS Utf8;
		DECLARE $reminder AS Json;

		INSERT INTO chats (bot_id, chat_id, language_code, state, reminder, created_at)
		VALUES ($bot_id, $chat_id, $language_code, $state, $reminder, CurrentUtcDatetime());
	`

	params := table.NewQueryParameters(
		table.ValueParam("$bot_id", types.Int64Value(botID)),
		table.ValueParam("$chat_id", types.Int64Value(chatID)),
		table.ValueParam("$language_code", types.UTF8Value(languageCode)),
		table.ValueParam("$state", types.UTF8Value(state)),
		table.ValueParam("$reminder", types.JSONValue(string(reminderJSON))),
	)

	err = db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, _, err := s.Execute(ctx, writeTx, query, params)
		return err
	})

	if err != nil {
		if ydb.IsOperationError(err, Ydb.StatusIds_PRECONDITION_FAILED) { // chat already exists
			return domain.ErrAlreadyExists
		}
		log.Error("create chat", log.Err(err), log.BotID(botID), log.ChatID(chatID))
		return domain.ErrInternal
	}

	return nil
}

func (db *DB) GetChat(ctx context.Context, botID int64, chatID int64) (chat *domain.Chat, _ error) {
	query := `
		DECLARE $bot_id AS Int64;
		DECLARE $chat_id AS int64;

		SELECT bot_id, chat_id, state, language_code, reminder
		FROM chats
		WHERE bot_id = $bot_id AND chat_id = $chat_id;
	`

	params := table.NewQueryParameters(
		table.ValueParam("$bot_id", types.Int64Value(botID)),
		table.ValueParam("$chat_id", types.Int64Value(chatID)),
	)

	err := db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, res, err := s.Execute(ctx, readTx, query, params)
		if err != nil {
			log.Error("execute get chat query", log.Err(err), log.BotID(botID), log.ChatID(chatID))
			return domain.ErrInternal
		}

		defer func(res result.Result) { _ = res.Close() }(res)
		if res.NextResultSet(ctx) && res.NextRow() {
			chat = &domain.Chat{}
			var reminderJSON string
			err = res.ScanWithDefaults(
				&chat.BotID,
				&chat.ChatID,
				&chat.State,
				&chat.LanguageCode,
				&reminderJSON,
			)
			if err != nil {
				log.Error("scan chat fields", log.Err(err), log.BotID(botID), log.ChatID(chatID))
				return domain.ErrInternal
			}

			var reminder domain.Reminder
			if err := json.Unmarshal([]byte(reminderJSON), &reminder); err != nil {
				log.Error("unmarshal reminder json", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
				return domain.ErrUnmarshalJSON
			}

			chat.Reminder = &reminder

			return nil
		}

		return domain.ErrNotFound
	})

	if err != nil {
		return nil, err
	}

	return chat, nil
}

func (db *DB) GetChats(ctx context.Context, botID int64) (chats []*domain.Chat, _ error) {
	query := `
		DECLARE $bot_id AS Int64;

		SELECT bot_id, chat_id, state, language_code, reminder
		FROM chats
		WHERE bot_id = $bot_id;
	`

	params := table.NewQueryParameters(table.ValueParam("$bot_id", types.Int64Value(botID)))
	err := db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, res, err := s.Execute(ctx, readTx, query, params)
		if err != nil {
			log.Error("execute get chats query", log.Err(err), log.BotID(botID))
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

func (db *DB) SetLanguageCode(ctx context.Context, botID int64, chatID int64, languageCode string) error {
	query := `
		DECLARE $bot_id AS Int64;
		DECLARE $chat_id AS Int64;
		DECLARE $language_code AS Utf8;

		UPDATE chats
		SET language_code = $language_code
		WHERE bot_id = $bot_id AND chat_id = $chat_id;
	`

	params := table.NewQueryParameters(
		table.ValueParam("$bot_id", types.Int64Value(botID)),
		table.ValueParam("$chat_id", types.Int64Value(chatID)),
		table.ValueParam("$language_code", types.UTF8Value(languageCode)),
	)

	err := db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, _, err := s.Execute(ctx, writeTx, query, params)
		if err != nil {
			log.Error("execute set language code query", log.Err(err), log.BotID(botID), log.ChatID(chatID))
			return domain.ErrInternal
		}
		return nil
	})

	return err
}

func (db *DB) SetSubscribed(ctx context.Context, botID int64, chatID int64, subscribed bool) error {
	query := `
		DECLARE $bot_id AS Int64;
		DECLARE $chat_id AS Int64;
		DECLARE $subscribed AS Bool;

		UPDATE chats
		SET subscribed = $subscribed, subscribed_at = CASE WHEN $subscribed THEN CurrentUtcDatetime() ELSE NULL END
		WHERE bot_id = $bot_id AND chat_id = $chat_id;
	`

	params := table.NewQueryParameters(
		table.ValueParam("$bot_id", types.Int64Value(botID)),
		table.ValueParam("$chat_id", types.Int64Value(chatID)),
		table.ValueParam("$subscribed", types.BoolValue(subscribed)),
	)

	err := db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, _, err := s.Execute(ctx, writeTx, query, params)
		if err != nil {
			log.Error("execute set subscribed query", log.Err(err), log.BotID(botID), log.ChatID(chatID))
			return domain.ErrInternal
		}
		return nil
	})

	return err
}

func (db *DB) SetReminderOffset(ctx context.Context, botID int64, chatID int64, reminderType domain.ReminderType, offset time.Duration) error {
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
			reminder.Today.Offset = offset
		case domain.ReminderTypeSoon:
			reminder.Soon.Offset = offset
		case domain.ReminderTypeArrive:
			reminder.Arrive.Offset = offset
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

func (db *DB) SetState(ctx context.Context, botID int64, chatID int64, state string) error {
	query := `
		DECLARE $bot_id AS Int64;
		DECLARE $chat_id AS Int64;
		DECLARE $state AS Utf8;

		UPDATE chats
		SET state = $state
		WHERE bot_id = $bot_id AND chat_id = $chat_id;
	`

	params := table.NewQueryParameters(
		table.ValueParam("$bot_id", types.Int64Value(botID)),
		table.ValueParam("$chat_id", types.Int64Value(chatID)),
		table.ValueParam("$state", types.UTF8Value(state)),
	)

	err := db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, _, err := s.Execute(ctx, writeTx, query, params)
		if err != nil {
			log.Error("execute set state query", log.Err(err), log.BotID(botID), log.ChatID(chatID))
			return domain.ErrInternal
		}
		return nil
	})

	return err
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

func (db *DB) GetStats(ctx context.Context, botID int64) (*domain.Stats, error) {
	query := `
		DECLARE $bot_id AS Int64;

		SELECT
			COUNT(*) AS users,
			COUNT_IF(subscribed) AS subscribed,
			COUNT_IF(NOT subscribed) AS unsubscribed
		FROM chats
		WHERE bot_id = $bot_id;

		SELECT
			language_code,
			COUNT(*) AS count
		FROM chats
		WHERE bot_id = $bot_id
		GROUP BY language_code;
	`

	stats := &domain.Stats{LanguagesGrouped: make(map[string]uint64)}

	params := table.NewQueryParameters(table.ValueParam("$bot_id", types.Int64Value(botID)))
	err := db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, res, err := s.Execute(ctx, readTx, query, params)
		if err != nil {
			log.Error("execute get stats query", log.Err(err), log.BotID(botID))
			return domain.ErrInternal
		}

		defer func(res result.Result) { _ = res.Close() }(res)
		if res.NextResultSet(ctx) && res.NextRow() {
			err = res.ScanWithDefaults(&stats.Users, &stats.Subscribed, &stats.Unsubscribed)
			if err != nil {
				log.Error("scan stats fields", log.Err(err), log.BotID(botID))
				return domain.ErrInternal
			}
		}

		if res.NextResultSet(ctx) {
			for res.NextRow() {
				var (
					languageCode string
					count        uint64
				)

				err = res.ScanWithDefaults(&languageCode, &count)
				if err != nil {
					log.Error("scan language stats", log.Err(err), log.BotID(botID))
					return domain.ErrInternal
				}
				stats.LanguagesGrouped[languageCode] = count
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return stats, nil
}
