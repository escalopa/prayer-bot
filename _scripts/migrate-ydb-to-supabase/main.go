package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/table"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/result"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/types"
)

var readTx = table.TxControl(
	table.BeginTx(table.WithOnlineReadOnly()),
	table.CommitTx(),
)

type chatRow struct {
	BotID        int64
	ChatID       int64
	LanguageCode *string
	State        *string
	ReminderJSON string
	Subscribed   *bool
	SubscribedAt *time.Time
	CreatedAt    *time.Time
}

type prayerRow struct {
	BotID      int64
	PrayerDate time.Time
	Fajr       *time.Time
	Shuruq     *time.Time
	Dhuhr      *time.Time
	Asr        *time.Time
	Maghrib    *time.Time
	Isha       *time.Time
}

func main() {
	mode := flag.String("mode", "full", "migration mode: full or incremental")
	flag.Parse()

	ctx := context.Background()

	ydbEndpoint := os.Getenv("YDB_ENDPOINT")
	if ydbEndpoint == "" {
		ydbEndpoint = stripGooseParams(os.Getenv("DB_CONNECTION_STRING"))
	}
	if ydbEndpoint == "" {
		ydbEndpoint = stripGooseParams(os.Getenv("GOOSE_DBSTRING_BASE"))
	}
	ydbToken := os.Getenv("YDB_TOKEN")
	if ydbToken == "" {
		ydbToken = os.Getenv("YDB_ACCESS_TOKEN")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("SUPABASE_DB_URL")
	}

	if ydbEndpoint == "" || databaseURL == "" {
		fmt.Fprintln(os.Stderr, "YDB_ENDPOINT and DATABASE_URL (or SUPABASE_DB_URL) are required")
		os.Exit(1)
	}

	ydbClient, err := ydb.Open(ctx, ydbEndpoint, ydb.WithAccessTokenCredentials(ydbToken))
	if err != nil {
		fmt.Fprintf(os.Stderr, "open ydb: %v\n", err)
		os.Exit(1)
	}
	defer ydbClient.Close(ctx)

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open postgres: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	chats, err := exportChats(ctx, ydbClient.Table())
	if err != nil {
		fmt.Fprintf(os.Stderr, "export chats: %v\n", err)
		os.Exit(1)
	}

	prayers, err := exportPrayers(ctx, ydbClient.Table())
	if err != nil {
		fmt.Fprintf(os.Stderr, "export prayers: %v\n", err)
		os.Exit(1)
	}

	if err := upsertChats(ctx, pool, chats); err != nil {
		fmt.Fprintf(os.Stderr, "upsert chats: %v\n", err)
		os.Exit(1)
	}

	if err := upsertPrayers(ctx, pool, prayers); err != nil {
		fmt.Fprintf(os.Stderr, "upsert prayers: %v\n", err)
		os.Exit(1)
	}

	if *mode == "full" {
		if err := verifyCounts(ctx, ydbClient.Table(), pool, len(chats), len(prayers)); err != nil {
			fmt.Fprintf(os.Stderr, "verify counts: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("migration complete: chats=%d prayers=%d mode=%s\n", len(chats), len(prayers), *mode)
}

func exportChats(ctx context.Context, client table.Client) ([]chatRow, error) {
	query := `
		SELECT bot_id, chat_id, language_code, state, reminder, subscribed, subscribed_at, created_at
		FROM chats;
	`

	var rows []chatRow
	err := client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, res, err := s.Execute(ctx, readTx, query, table.NewQueryParameters())
		if err != nil {
			return err
		}
		defer res.Close()

		for res.NextResultSet(ctx) {
			for res.NextRow() {
				var row chatRow
				if err := res.ScanWithDefaults(
					&row.BotID,
					&row.ChatID,
					&row.LanguageCode,
					&row.State,
					&row.ReminderJSON,
					&row.Subscribed,
					&row.SubscribedAt,
					&row.CreatedAt,
				); err != nil {
					return err
				}
				rows = append(rows, row)
			}
		}
		return nil
	})

	return rows, err
}

func exportPrayers(ctx context.Context, client table.Client) ([]prayerRow, error) {
	query := `
		SELECT bot_id, prayer_date, fajr, shuruq, dhuhr, asr, maghrib, isha
		FROM prayers;
	`

	var rows []prayerRow
	err := client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, res, err := s.Execute(ctx, readTx, query, table.NewQueryParameters())
		if err != nil {
			return err
		}
		defer res.Close()

		for res.NextResultSet(ctx) {
			for res.NextRow() {
				var row prayerRow
				if err := res.ScanWithDefaults(
					&row.BotID,
					&row.PrayerDate,
					&row.Fajr,
					&row.Shuruq,
					&row.Dhuhr,
					&row.Asr,
					&row.Maghrib,
					&row.Isha,
				); err != nil {
					return err
				}
				rows = append(rows, row)
			}
		}
		return nil
	})

	return rows, err
}

func upsertChats(ctx context.Context, pool *pgxpool.Pool, rows []chatRow) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, row := range rows {
		if !json.Valid([]byte(row.ReminderJSON)) {
			empty, _ := json.Marshal(&domain.Reminder{})
			row.ReminderJSON = string(empty)
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO chats (bot_id, chat_id, language_code, state, reminder, subscribed, subscribed_at, created_at)
			VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8)
			ON CONFLICT (bot_id, chat_id) DO UPDATE SET
				language_code = EXCLUDED.language_code,
				state = EXCLUDED.state,
				reminder = EXCLUDED.reminder,
				subscribed = EXCLUDED.subscribed,
				subscribed_at = EXCLUDED.subscribed_at,
				created_at = COALESCE(chats.created_at, EXCLUDED.created_at)
		`,
			row.BotID,
			row.ChatID,
			row.LanguageCode,
			row.State,
			row.ReminderJSON,
			row.Subscribed,
			row.SubscribedAt,
			row.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func upsertPrayers(ctx context.Context, pool *pgxpool.Pool, rows []prayerRow) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, row := range rows {
		_, err := tx.Exec(ctx, `
			INSERT INTO prayers (bot_id, prayer_date, fajr, shuruq, dhuhr, asr, maghrib, isha)
			VALUES ($1, $2::date, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (bot_id, prayer_date) DO UPDATE SET
				fajr = EXCLUDED.fajr,
				shuruq = EXCLUDED.shuruq,
				dhuhr = EXCLUDED.dhuhr,
				asr = EXCLUDED.asr,
				maghrib = EXCLUDED.maghrib,
				isha = EXCLUDED.isha
		`,
			row.BotID,
			row.PrayerDate,
			row.Fajr,
			row.Shuruq,
			row.Dhuhr,
			row.Asr,
			row.Maghrib,
			row.Isha,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func verifyCounts(ctx context.Context, client table.Client, pool *pgxpool.Pool, wantChats, wantPrayers int) error {
	var ydbChats, ydbPrayers int
	err := client.Do(ctx, func(ctx context.Context, s table.Session) error {
		_, res, err := s.Execute(ctx, readTx, `
			SELECT COUNT(*) AS chats FROM chats;
			SELECT COUNT(*) AS prayers FROM prayers;
		`, table.NewQueryParameters())
		if err != nil {
			return err
		}
		defer func(res result.Result) { _ = res.Close() }(res)

		if res.NextResultSet(ctx) && res.NextRow() {
			if err := res.ScanWithDefaults(&ydbChats); err != nil {
				return err
			}
		}
		if res.NextResultSet(ctx) && res.NextRow() {
			if err := res.ScanWithDefaults(&ydbPrayers); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	var pgChats, pgPrayers int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM chats`).Scan(&pgChats); err != nil {
		return err
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM prayers`).Scan(&pgPrayers); err != nil {
		return err
	}

	if pgChats != ydbChats || pgChats != wantChats {
		return fmt.Errorf("chat count mismatch: ydb=%d exported=%d postgres=%d", ydbChats, wantChats, pgChats)
	}
	if pgPrayers != ydbPrayers || pgPrayers != wantPrayers {
		return fmt.Errorf("prayer count mismatch: ydb=%d exported=%d postgres=%d", ydbPrayers, wantPrayers, pgPrayers)
	}

	return nil
}

func stripGooseParams(dsn string) string {
	dsn = strings.TrimSpace(dsn)
	if idx := strings.Index(dsn, "&"); idx >= 0 {
		return dsn[:idx]
	}
	return dsn
}
