package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/table"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/result/named"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/types"
)

// export OLD_DB_CONNECTION_STRING="your-old-db-connection-string"
// export NEW_DB_CONNECTION_STRING="your-new-db-connection-string"
// export YDB_TOKEN="your-access-token"

// cd _scripts/migrate
// go run main.go

// Domain types matching the new schema
type JamaatDelayConfig struct {
	Fajr    int64 `json:"fajr"`    // nanoseconds
	Shuruq  int64 `json:"shuruq"`  // nanoseconds
	Dhuhr   int64 `json:"dhuhr"`   // nanoseconds
	Asr     int64 `json:"asr"`     // nanoseconds
	Maghrib int64 `json:"maghrib"` // nanoseconds
	Isha    int64 `json:"isha"`    // nanoseconds
}

type JamaatConfig struct {
	Enabled bool               `json:"enabled"`
	Delay   *JamaatDelayConfig `json:"delay"`
}

type ReminderConfig struct {
	Offset    int64  `json:"offset"` // nanoseconds
	MessageID int    `json:"message_id"`
	LastAt    string `json:"last_at"` // RFC3339 format
}

type Reminder struct {
	Tomorrow *ReminderConfig `json:"tomorrow"`
	Soon     *ReminderConfig `json:"soon"`
	Arrive   *ReminderConfig `json:"arrive"`
	Jamaat   *JamaatConfig   `json:"jamaat"`
}

// Old table row structure
type OldChat struct {
	ChatID            int64
	BotID             int64
	LanguageCode      *string
	State             *string
	ReminderOffset    *int32
	ReminderMessageID *int32
	Jamaat            *bool
	JamaatMessageID   *int32
	Subscribed        *bool
	SubscribedAt      *time.Time
	CreatedAt         *time.Time
}

// New table row structure
type NewChat struct {
	ChatID       int64
	BotID        int64
	LanguageCode *string
	State        *string
	Reminder     string // JSON string
	Subscribed   *bool
	SubscribedAt *time.Time
	CreatedAt    *time.Time
}

func main() {
	ctx := context.Background()

	// Get environment variables
	oldDBConnectionString := os.Getenv("OLD_DB_CONNECTION_STRING")
	newDBConnectionString := os.Getenv("NEW_DB_CONNECTION_STRING")
	ydbToken := os.Getenv("YDB_TOKEN")

	if oldDBConnectionString == "" || newDBConnectionString == "" || ydbToken == "" {
		log.Fatal("Missing required environment variables: OLD_DB_CONNECTION_STRING, NEW_DB_CONNECTION_STRING, YDB_TOKEN")
	}

	// Connect to old database
	log.Println("Connecting to old database...")
	oldDB, err := ydb.Open(ctx, oldDBConnectionString,
		ydb.WithAccessTokenCredentials(ydbToken),
	)
	if err != nil {
		log.Fatalf("Failed to connect to old database: %v", err)
	}
	defer func() { _ = oldDB.Close(ctx) }()

	// Connect to new database
	log.Println("Connecting to new database...")
	newDB, err := ydb.Open(ctx, newDBConnectionString,
		ydb.WithAccessTokenCredentials(ydbToken),
	)
	if err != nil {
		log.Fatalf("Failed to connect to new database: %v", err)
	}
	defer func() { _ = newDB.Close(ctx) }()

	// Fetch all chats from old database
	log.Println("Fetching chats from old database...")
	oldChats, err := fetchOldChats(ctx, oldDB)
	if err != nil {
		log.Fatalf("Failed to fetch old chats: %v", err)
	}
	log.Printf("Fetched %d chats from old database", len(oldChats))

	// Transform and migrate chats
	log.Println("Migrating chats to new database...")
	moscowLocation, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Fatalf("Failed to load Moscow timezone: %v", err)
	}

	newChats := make([]NewChat, 0, len(oldChats))
	for _, oldChat := range oldChats {
		newChat, err := transformChat(oldChat, moscowLocation)
		if err != nil {
			log.Printf("Failed to transform chat (bot_id=%d, chat_id=%d): %v", oldChat.BotID, oldChat.ChatID, err)
			continue
		}
		newChats = append(newChats, newChat)
	}

	// Insert chats into new database
	if err := upsertNewChats(ctx, newDB, newChats); err != nil {
		log.Fatalf("Failed to upsert new chats: %v", err)
	}

	log.Printf("Successfully migrated %d chats", len(newChats))
}

func fetchOldChats(ctx context.Context, db *ydb.Driver) ([]OldChat, error) {
	var chats []OldChat

	err := db.Table().Do(ctx, func(ctx context.Context, s table.Session) error {
		_, res, err := s.Execute(ctx, table.DefaultTxControl(),
			`SELECT chat_id, bot_id, language_code, state, reminder_offset,
                    reminder_message_id, jamaat, jamaat_message_id, subscribed,
                    subscribed_at, created_at
             FROM chats`,
			nil,
		)
		if err != nil {
			return err
		}
		defer func() { _ = res.Close() }()

		for res.NextResultSet(ctx) {
			for res.NextRow() {
				var chat OldChat
				err := res.ScanNamed(
					named.Required("chat_id", &chat.ChatID),
					named.Required("bot_id", &chat.BotID),
					named.Optional("language_code", &chat.LanguageCode),
					named.Optional("state", &chat.State),
					named.Optional("reminder_offset", &chat.ReminderOffset),
					named.Optional("reminder_message_id", &chat.ReminderMessageID),
					named.Optional("jamaat", &chat.Jamaat),
					named.Optional("jamaat_message_id", &chat.JamaatMessageID),
					named.Optional("subscribed", &chat.Subscribed),
					named.Optional("subscribed_at", &chat.SubscribedAt),
					named.Optional("created_at", &chat.CreatedAt),
				)
				if err != nil {
					return err
				}
				chats = append(chats, chat)
			}
		}

		return res.Err()
	})

	return chats, err
}

func transformChat(oldChat OldChat, moscowLocation *time.Location) (NewChat, error) {
	now := time.Now().In(moscowLocation)

	// Build reminder object
	reminderOffset := int32(20)
	if oldChat.ReminderOffset != nil {
		reminderOffset = *oldChat.ReminderOffset
	}

	reminderMessageID := 0
	if oldChat.ReminderMessageID != nil {
		reminderMessageID = int(*oldChat.ReminderMessageID)
	}

	jamaatMessageID := 0
	if oldChat.JamaatMessageID != nil {
		jamaatMessageID = int(*oldChat.JamaatMessageID)
	}

	jamaat := false
	if oldChat.Jamaat != nil {
		jamaat = *oldChat.Jamaat
	}

	subscribed := false
	if oldChat.Subscribed != nil {
		subscribed = *oldChat.Subscribed
	}

	reminder := Reminder{
		Tomorrow: &ReminderConfig{
			Offset: int64(3 * time.Hour),
			LastAt: now.Format(time.RFC3339),
		},
		Soon: &ReminderConfig{
			Offset:    int64(reminderOffset) * int64(time.Minute),
			MessageID: reminderMessageID,
			LastAt:    now.Format(time.RFC3339),
		},
		Arrive: &ReminderConfig{
			Offset:    0,
			MessageID: jamaatMessageID,
			LastAt:    now.Format(time.RFC3339),
		},
		Jamaat: &JamaatConfig{
			Enabled: subscribed && jamaat,
			Delay: &JamaatDelayConfig{
				Fajr:    int64(10 * time.Minute),
				Shuruq:  int64(10 * time.Minute),
				Dhuhr:   int64(10 * time.Minute),
				Asr:     int64(10 * time.Minute),
				Maghrib: int64(10 * time.Minute),
				Isha:    int64(20 * time.Minute),
			},
		},
	}

	// Marshal reminder to JSON
	reminderJSON, err := json.Marshal(reminder)
	if err != nil {
		return NewChat{}, fmt.Errorf("failed to marshal reminder: %w", err)
	}

	return NewChat{
		ChatID:       oldChat.ChatID,
		BotID:        oldChat.BotID,
		LanguageCode: oldChat.LanguageCode,
		State:        oldChat.State,
		Reminder:     string(reminderJSON),
		Subscribed:   oldChat.Subscribed,
		SubscribedAt: oldChat.SubscribedAt,
		CreatedAt:    oldChat.CreatedAt,
	}, nil
}

func upsertNewChats(ctx context.Context, db *ydb.Driver, chats []NewChat) error {
	return db.Table().Do(ctx, func(ctx context.Context, s table.Session) error {
		for _, chat := range chats {
			err := upsertChat(ctx, s, chat)
			if err != nil {
				log.Printf("Failed to upsert chat (bot_id=%d, chat_id=%d): %v", chat.BotID, chat.ChatID, err)
				return err
			}
		}
		return nil
	})
}

func upsertChat(ctx context.Context, s table.Session, chat NewChat) error {
	query := `
		DECLARE $chat_id AS Int64;
		DECLARE $bot_id AS Int64;
		DECLARE $language_code AS Utf8?;
		DECLARE $state AS Utf8?;
		DECLARE $reminder AS Json;
		DECLARE $subscribed AS Bool?;
		DECLARE $subscribed_at AS Datetime?;
		DECLARE $created_at AS Datetime?;

		UPSERT INTO chats (chat_id, bot_id, language_code, state, reminder, subscribed, subscribed_at, created_at)
		VALUES ($chat_id, $bot_id, $language_code, $state, $reminder, $subscribed, $subscribed_at, $created_at);
	`

	params := table.NewQueryParameters(
		table.ValueParam("$chat_id", types.Int64Value(chat.ChatID)),
		table.ValueParam("$bot_id", types.Int64Value(chat.BotID)),
		table.ValueParam("$reminder", types.JSONValue(chat.Reminder)),
	)

	// Handle optional language_code
	if chat.LanguageCode != nil {
		params.Add(table.ValueParam("$language_code", types.OptionalValue(types.UTF8Value(*chat.LanguageCode))))
	} else {
		params.Add(table.ValueParam("$language_code", types.NullValue(types.TypeUTF8)))
	}

	// Handle optional state
	if chat.State != nil {
		params.Add(table.ValueParam("$state", types.OptionalValue(types.UTF8Value(*chat.State))))
	} else {
		params.Add(table.ValueParam("$state", types.NullValue(types.TypeUTF8)))
	}

	// Handle optional subscribed
	if chat.Subscribed != nil {
		params.Add(table.ValueParam("$subscribed", types.OptionalValue(types.BoolValue(*chat.Subscribed))))
	} else {
		params.Add(table.ValueParam("$subscribed", types.NullValue(types.TypeBool)))
	}

	// Handle optional subscribed_at
	if chat.SubscribedAt != nil {
		params.Add(table.ValueParam("$subscribed_at", types.OptionalValue(types.DatetimeValueFromTime(*chat.SubscribedAt))))
	} else {
		params.Add(table.ValueParam("$subscribed_at", types.NullValue(types.TypeDatetime)))
	}

	// Handle optional created_at
	if chat.CreatedAt != nil {
		params.Add(table.ValueParam("$created_at", types.OptionalValue(types.DatetimeValueFromTime(*chat.CreatedAt))))
	} else {
		params.Add(table.ValueParam("$created_at", types.NullValue(types.TypeDatetime)))
	}

	_, _, err := s.Execute(ctx, table.DefaultTxControl(), query, params)

	return err
}
