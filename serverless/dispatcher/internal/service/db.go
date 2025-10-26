package service

import (
	"context"
	"encoding/json"
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

//revive:disable:argument-limit
func (db *DB) CreateChat(
	ctx context.Context,
	botID int64,
	chatID int64,
	languageCode string,
	state string,
	isGroup bool,
	reminder *domain.Reminder,
) error {

	reminderJSON, err := json.Marshal(reminder)
	if err != nil {
		return err
	}

	query := `
		DECLARE $bot_id AS Int64;
		DECLARE $chat_id AS Int64;
		DECLARE $language_code AS Utf8;
		DECLARE $state AS Utf8;
		DECLARE $is_group AS Bool;
		DECLARE $reminder AS Json;

		INSERT INTO chats (bot_id, chat_id, language_code, state, is_group, reminder, created_at)
		VALUES ($bot_id, $chat_id, $language_code, $state, $is_group, $reminder, CurrentUtcDatetime());
	`

	params := table.NewQueryParameters(
		table.ValueParam("$bot_id", types.Int64Value(botID)),
		table.ValueParam("$chat_id", types.Int64Value(chatID)),
		table.ValueParam("$language_code", types.UTF8Value(languageCode)),
		table.ValueParam("$state", types.UTF8Value(state)),
		table.ValueParam("$is_group", types.BoolValue(isGroup)),
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
		return err
	}

	return nil
}

func (db *DB) GetChat(ctx context.Context, botID int64, chatID int64) (chat *domain.Chat, _ error) {
	query := `
		DECLARE $bot_id AS Int64;
		DECLARE $chat_id AS int64;

		SELECT bot_id, chat_id, state, language_code, is_group, reminder
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
			return err
		}

		defer func(res result.Result) { _ = res.Close() }(res)
		if res.NextResultSet(ctx) && res.NextRow() {
			chat = &domain.Chat{}
			var reminderJSON *string
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
			if err := json.Unmarshal([]byte(*reminderJSON), &reminder); err != nil {
				return err
			}

			chat.Reminder = &reminder

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

	return chat, nil
}

func (db *DB) GetChats(ctx context.Context, botID int64) (chats []*domain.Chat, _ error) {
	query := `
		DECLARE $bot_id AS Int64;

		SELECT bot_id, chat_id, state, language_code, is_group, reminder
		FROM chats
		WHERE bot_id = $bot_id;
	`

	params := table.NewQueryParameters(table.ValueParam("$bot_id", types.Int64Value(botID)))
	err := db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, res, err := s.Execute(ctx, readTx, query, params)
		if err != nil {
			return err
		}

		defer func(res result.Result) { _ = res.Close() }(res)
		if res.NextResultSet(ctx) {
			for res.NextRow() {
				chat := &domain.Chat{}
				var reminderJSON *string
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
				if err := json.Unmarshal([]byte(*reminderJSON), &reminder); err != nil {
					return err
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
		return err
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
		return err
	})

	return err
}

func (db *DB) SetReminderOffset(ctx context.Context, botID int64, chatID int64, reminderType domain.ReminderType, offset time.Duration) error {
	offsetNanos := offset.Nanoseconds()

	query := `
		DECLARE $bot_id AS Int64;
		DECLARE $chat_id AS Int64;
		DECLARE $reminder_type AS Utf8;
		DECLARE $offset AS Int64;

		UPDATE chats
		SET reminder = Json_SetField(reminder, $reminder_type || ".offset", CAST($offset AS Json))
		WHERE bot_id = $bot_id AND chat_id = $chat_id;
	`

	params := table.NewQueryParameters(
		table.ValueParam("$bot_id", types.Int64Value(botID)),
		table.ValueParam("$chat_id", types.Int64Value(chatID)),
		table.ValueParam("$reminder_type", types.UTF8Value(reminderType.String())),
		table.ValueParam("$offset", types.Int64Value(offsetNanos)),
	)

	err := db.client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, _, err := s.Execute(ctx, writeTx, query, params)
		return err
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
		return err
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
			return err
		}

		defer func(res result.Result) { _ = res.Close() }(res)
		if res.NextResultSet(ctx) && res.NextRow() {
			err = res.ScanWithDefaults(&stats.Users, &stats.Subscribed, &stats.Unsubscribed)
			if err != nil {
				return err
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
					return err
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
