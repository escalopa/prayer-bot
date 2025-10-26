package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/table"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/result"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/types"
	yc "github.com/ydb-platform/ydb-go-yc"
)

var (
	writeTx = table.TxControl(
		table.BeginTx(table.WithSerializableReadWrite()),
		table.CommitTx(),
	)
)

type oldChatData struct {
	BotID             int64
	ChatID            int64
	ReminderOffset    int32
	ReminderMessageID int32
	IsGroup           bool
	JamaatOffset      int32
	JamaatMessageID   int32
}

func main() {
	dryRun := flag.Bool("dry-run", false, "Preview changes without writing to database")
	flag.Parse()

	ctx := context.Background()

	// Load Moscow timezone
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Fatalf("Failed to load timezone: %v", err)
	}
	now := time.Now().In(loc)

	// Connect to YDB
	ydbEndpoint := os.Getenv("YDB_ENDPOINT")
	if ydbEndpoint == "" {
		log.Fatal("YDB_ENDPOINT environment variable is not set")
	}

	sdk, err := ydb.Open(ctx, ydbEndpoint,
		yc.WithMetadataCredentials(),
		yc.WithInternalCA(),
	)
	if err != nil {
		log.Fatalf("Failed to connect to YDB: %v", err)
	}
	defer func() { _ = sdk.Close(ctx) }()

	client := sdk.Table()

	// Statistics
	var (
		totalProcessed int
		totalMigrated  int
		totalSkipped   int
	)

	// Query all chats
	query := `
		SELECT bot_id, chat_id, reminder_offset, reminder_message_id, is_group, jamaat_offset, jamaat_message_id
		FROM chats
	`

	err = client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, res, err := s.Execute(ctx,
			table.TxControl(
				table.BeginTx(table.WithOnlineReadOnly()),
				table.CommitTx(),
			),
			query,
			table.NewQueryParameters(),
		)
		if err != nil {
			return fmt.Errorf("execute query: %w", err)
		}
		defer func() { _ = res.Close() }()

		chats := []oldChatData{}
		if res.NextResultSet(ctx) {
			for res.NextRow() {
				var chat oldChatData
				var reminderOffset, reminderMessageID, jamaatOffset, jamaatMessageID *int32
				var isGroup *bool

				err = res.Scan(
					&chat.BotID,
					&chat.ChatID,
					&reminderOffset,
					&reminderMessageID,
					&isGroup,
					&jamaatOffset,
					&jamaatMessageID,
				)
				if err != nil {
					log.Printf("Failed to scan row: %v", err)
					continue
				}

				// Handle nullable fields
				if reminderOffset != nil {
					chat.ReminderOffset = *reminderOffset
				}
				if reminderMessageID != nil {
					chat.ReminderMessageID = *reminderMessageID
				}
				if isGroup != nil {
					chat.IsGroup = *isGroup
				}
				if jamaatOffset != nil {
					chat.JamaatOffset = *jamaatOffset
				}
				if jamaatMessageID != nil {
					chat.JamaatMessageID = *jamaatMessageID
				}

				chats = append(chats, chat)
			}
		}

		log.Printf("Found %d chats to process", len(chats))

		// Process chats in batches
		batchSize := 100
		for i := 0; i < len(chats); i += batchSize {
			end := i + batchSize
			if end > len(chats) {
				end = len(chats)
			}
			batch := chats[i:end]

			for _, chat := range batch {
				totalProcessed++

				// Check if chat has any reminder data
				hasReminderData := chat.ReminderOffset > 0 ||
					chat.ReminderMessageID > 0 ||
					chat.JamaatOffset > 0 ||
					chat.JamaatMessageID > 0

				if !hasReminderData {
					totalSkipped++
					continue
				}

				// Build Reminder object
				reminder := domain.Reminder{
					Today: domain.ReminderConfig{
						Offset:    0, // Disabled by default
						MessageID: 0,
						LastAt:    now.Truncate(24 * time.Hour), // Date only
					},
					Soon: domain.ReminderConfig{
						Offset:    time.Duration(chat.ReminderOffset) * time.Minute,
						MessageID: int(chat.ReminderMessageID),
						LastAt:    now,
					},
					Arrive: domain.ReminderConfig{
						Offset:    0, // Always 0
						MessageID: 0,
						LastAt:    now,
					},
					JamaatDelay: domain.JamaatDelay{
						Fajr:    10 * time.Minute,
						Shuruq:  10 * time.Minute,
						Dhuhr:   10 * time.Minute,
						Asr:     10 * time.Minute,
						Maghrib: 10 * time.Minute,
						Isha:    20 * time.Minute,
					},
				}

				// Serialize to JSON
				reminderJSON, err := json.Marshal(reminder)
				if err != nil {
					log.Printf("Failed to marshal reminder for chat %d/%d: %v", chat.BotID, chat.ChatID, err)
					continue
				}

				if *dryRun {
					log.Printf("[DRY RUN] Would migrate chat %d/%d: %s", chat.BotID, chat.ChatID, string(reminderJSON))
					totalMigrated++
					continue
				}

				// Update database
				updateQuery := `
					DECLARE $bot_id AS Int64;
					DECLARE $chat_id AS Int64;
					DECLARE $reminder AS Json;

					UPDATE chats
					SET reminder = $reminder
					WHERE bot_id = $bot_id AND chat_id = $chat_id;
				`

				params := table.NewQueryParameters(
					table.ValueParam("$bot_id", types.Int64Value(chat.BotID)),
					table.ValueParam("$chat_id", types.Int64Value(chat.ChatID)),
					table.ValueParam("$reminder", types.JSONValue(string(reminderJSON))),
				)

				err = client.Do(ctx, func(ctx context.Context, s table.Session) error {
					_, _, err := s.Execute(ctx, writeTx, updateQuery, params)
					return err
				})

				if err != nil {
					log.Printf("Failed to update chat %d/%d: %v", chat.BotID, chat.ChatID, err)
					continue
				}

				totalMigrated++
				if totalMigrated%10 == 0 {
					log.Printf("Migrated %d chats so far...", totalMigrated)
				}
			}
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Print statistics
	log.Println("==================================================")
	log.Println("Migration completed successfully!")
	log.Printf("Total processed: %d", totalProcessed)
	log.Printf("Total migrated:  %d", totalMigrated)
	log.Printf("Total skipped:   %d", totalSkipped)
	if *dryRun {
		log.Println("*** DRY RUN MODE - No changes were made ***")
	}
}
