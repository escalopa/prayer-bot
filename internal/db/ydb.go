package db

import (
	"context"
	"encoding/json"
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

type YDB struct {
	client table.Client
}

func openYDB(ctx context.Context, endpoint, token string) (*YDB, error) {
	var opts []ydb.Option
	if token != "" {
		opts = append(opts, ydb.WithAccessTokenCredentials(token))
	} else {
		opts = append(opts, yc.WithMetadataCredentials(), yc.WithInternalCA())
	}

	sdk, err := ydb.Open(ctx, endpoint, opts...)
	if err != nil {
		return nil, err
	}

	return &YDB{client: sdk.Table()}, nil
}

//revive:disable:argument-limit
func (db *YDB) CreateChat(
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

func (db *YDB) GetChat(ctx context.Context, botID int64, chatID int64) (chat *domain.Chat, _ error) {
	query := `
		DECLARE $bot_id AS Int64;
		DECLARE $chat_id AS int64;

		SELECT bot_id, chat_id, state, language_code, subscribed, reminder
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
				&chat.Subscribed,
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

func (db *YDB) GetChats(ctx context.Context, botID int64) (chats []*domain.Chat, _ error) {
	query := `
		DECLARE $bot_id AS Int64;

		SELECT bot_id, chat_id, state, language_code, subscribed, reminder
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
					&chat.Subscribed,
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

func (db *YDB) GetChatsByIDs(ctx context.Context, botID int64, chatIDs []int64) (chats []*domain.Chat, _ error) {
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

func (db *YDB) GetSubscribers(ctx context.Context, botID int64) (chatIDs []int64, _ error) {
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

func (db *YDB) SetLanguageCode(ctx context.Context, botID int64, chatID int64, languageCode string) error {
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

func (db *YDB) SetSubscribed(ctx context.Context, botID int64, chatID int64, subscribed bool) error {
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

func (db *YDB) SetReminderOffset(ctx context.Context, botID int64, chatID int64, reminderType domain.ReminderType, offset time.Duration) error {
	return db.updateReminder(ctx, botID, chatID, func(reminder *domain.Reminder) {
		switch reminderType {
		case domain.ReminderTypeTomorrow:
			reminder.Tomorrow.Offset = domain.Duration(offset)
			reminder.Tomorrow.LastAt = time.Now()
		case domain.ReminderTypeSoon:
			reminder.Soon.Offset = domain.Duration(offset)
			reminder.Soon.LastAt = time.Now()
		}
	})
}

func (db *YDB) SetJamaatEnabled(ctx context.Context, botID int64, chatID int64, enabled bool) error {
	return db.updateReminder(ctx, botID, chatID, func(reminder *domain.Reminder) {
		reminder.Jamaat.Enabled = enabled
	})
}

func (db *YDB) SetJamaatDelay(ctx context.Context, botID int64, chatID int64, prayerID domain.PrayerID, delay time.Duration) error {
	return db.updateReminder(ctx, botID, chatID, func(reminder *domain.Reminder) {
		reminder.Jamaat.Delay.SetDelayByPrayerID(prayerID, delay)
	})
}

func (db *YDB) SetState(ctx context.Context, botID int64, chatID int64, state string) error {
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

func (db *YDB) GetPrayerDay(ctx context.Context, botID int64, date time.Time) (prayerDay *domain.PrayerDay, _ error) {
	// DateValueFromTime uses absolute UTC seconds, so normalize the local calendar date
	// to UTC midnight to get the correct YDB Date value for that calendar day.
	y, m, d := date.Date()
	date = time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
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

func (db *YDB) SetPrayerDays(ctx context.Context, botID int64, rows []*domain.PrayerDay) error {
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

func (db *YDB) GetStats(ctx context.Context, botID int64) (*domain.Stats, error) {
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

func (db *YDB) UpdateReminder(
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
		case domain.ReminderTypeTomorrow:
			reminder.Tomorrow.MessageID = messageID
			reminder.Tomorrow.LastAt = lastAt
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

func (db *YDB) DeleteChat(ctx context.Context, botID int64, chatID int64) error {
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

func (db *YDB) updateReminder(ctx context.Context, botID int64, chatID int64, mutate func(*domain.Reminder)) error {
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

		mutate(&reminder)

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
