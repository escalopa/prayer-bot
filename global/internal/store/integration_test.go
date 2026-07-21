package store

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/escalopa/prayer-bot/global/internal/database"
	"github.com/escalopa/prayer-bot/global/internal/domain"
)

// Integration tests run only when TEST_DATABASE_URL points at a disposable
// PostgreSQL database. They exercise real SQL through pgx in QueryExecModeExec,
// which is the only place the JSONB-as-text encoding and the transactional
// outbox can be verified end-to-end. `make test` skips them automatically.
//
//	TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable' go test ./internal/store/
//
// WARNING: the harness drops and recreates the global_bot_testing schema, so
// never point TEST_DATABASE_URL at a database with real data.
func openTestStore(t *testing.T) *Store {
	t.Helper()
	url := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run store integration tests")
	}
	ctx := context.Background()
	applyMigrations(t, ctx, url, database.TestingSchema)

	storage, err := Open(ctx, url, database.TestingSchema)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(storage.Close)
	return storage
}

// applyMigrations recreates the target schema and runs every goose "Up" section
// in order. It uses the simple protocol so multi-statement DDL executes in one
// call, and substitutes ${GLOBAL_DB_SCHEMA} the way goose ENVSUB would.
func applyMigrations(t *testing.T, ctx context.Context, url, schema string) {
	t.Helper()
	config, err := pgx.ParseConfig(url)
	if err != nil {
		t.Fatalf("parse migration url: %v", err)
	}
	config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		t.Fatalf("connect for migrations: %v", err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, "DROP SCHEMA IF EXISTS "+pgx.Identifier{schema}.Sanitize()+" CASCADE"); err != nil {
		t.Fatalf("reset schema: %v", err)
	}

	dir := filepath.Join("..", "..", "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	var files []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read migration %s: %v", name, err)
		}
		up := goosUpSection(string(raw))
		up = strings.ReplaceAll(up, "${GLOBAL_DB_SCHEMA}", schema)
		if strings.TrimSpace(up) == "" {
			continue
		}
		if _, err := conn.Exec(ctx, up); err != nil {
			t.Fatalf("apply migration %s: %v", name, err)
		}
	}
}

func goosUpSection(content string) string {
	var (
		builder   strings.Builder
		capturing bool
	)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "-- +goose Up") {
			capturing = true
			continue
		}
		if strings.HasPrefix(trimmed, "-- +goose Down") {
			break
		}
		if !capturing || strings.HasPrefix(trimmed, "-- +goose") {
			continue
		}
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	return builder.String()
}

func seedChat(t *testing.T, storage *Store, chatID int64) {
	t.Helper()
	if err := storage.UpsertChat(context.Background(), domain.Chat{
		TelegramChatID: chatID, Type: "private", LanguageCode: "en",
	}); err != nil {
		t.Fatalf("seed chat: %v", err)
	}
}

// TestIntegrationProfileRoundTripPreservesJSONBAdjustments guards the exact
// production incident: under the Supabase transaction pooler, adjustments passed
// as []byte failed with SQLSTATE 22P02. This asserts the JSON-text encoding
// persists and reads back correctly.
func TestIntegrationProfileRoundTripPreservesJSONBAdjustments(t *testing.T) {
	storage := openTestStore(t)
	ctx := context.Background()
	seedChat(t, storage, 3)

	profile := domain.PrayerProfile{
		ChatID: 3, Latitude: 30.044, Longitude: 31.236, Timezone: "Africa/Cairo",
		Method: domain.MethodEgyptian, Madhab: domain.MadhabShafii,
		HighLatitudeRule: domain.HighLatitudeAngleBased,
		Adjustments:      domain.Adjustments{Fajr: 2, Dhuhr: 3, Isha: -1},
		HijriAdjustment:  1,
	}
	saved, err := storage.UpsertProfile(ctx, profile)
	if err != nil {
		t.Fatalf("upsert profile: %v", err)
	}
	if saved.Version < 1 {
		t.Fatalf("expected a positive version, got %d", saved.Version)
	}

	got, err := storage.Profile(ctx, 3)
	if err != nil {
		t.Fatalf("read profile: %v", err)
	}
	if got.Adjustments != profile.Adjustments {
		t.Fatalf("adjustments round-trip mismatch: got %+v want %+v", got.Adjustments, profile.Adjustments)
	}
	if got.Method != domain.MethodEgyptian || got.Timezone != "Africa/Cairo" || got.HijriAdjustment != 1 {
		t.Fatalf("profile fields did not round-trip: %+v", got)
	}
}

// TestIntegrationClaimDueWritesOutboxWithJSONPayload verifies the transactional
// outbox: a due schedule is claimed, a JSON-text delivery payload is written and
// decodes cleanly, and the delivery lease is single-owner.
func TestIntegrationClaimDueWritesOutboxWithJSONPayload(t *testing.T) {
	storage := openTestStore(t)
	ctx := context.Background()
	seedChat(t, storage, 7)

	profile, err := storage.UpsertProfile(ctx, domain.PrayerProfile{
		ChatID: 7, Latitude: 51.507, Longitude: -0.128, Timezone: "Europe/London",
		Method: domain.MethodMWL, Madhab: domain.MadhabShafii,
		HighLatitudeRule: domain.HighLatitudeMiddleNight,
	})
	if err != nil {
		t.Fatalf("upsert profile: %v", err)
	}
	if err := storage.SetWeeklyRule(ctx, 7, domain.ReminderWeeklyKahf, true); err != nil {
		t.Fatalf("enable weekly rule: %v", err)
	}
	rules, err := storage.EnabledRules(ctx, 7)
	if err != nil || len(rules) != 1 {
		t.Fatalf("expected exactly one enabled rule, got %d (%v)", len(rules), err)
	}

	due := time.Now().Add(-time.Minute)
	schedule, err := storage.UpsertSchedule(ctx, domain.ReminderSchedule{
		RuleID: rules[0].ID, ChatID: 7, ProfileVersion: profile.Version,
		LocalDate: "2026-07-24", PrayerAt: due, NextRunAt: due,
	})
	if err != nil {
		t.Fatalf("upsert schedule: %v", err)
	}

	count, err := storage.ClaimDue(ctx, time.Now(), 10)
	if err != nil {
		t.Fatalf("claim due: %v", err)
	}
	if count != 1 {
		t.Fatalf("claimed %d schedules, want 1", count)
	}

	items, err := storage.PendingOutbox(ctx, 10)
	if err != nil {
		t.Fatalf("pending outbox: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("outbox has %d items, want 1", len(items))
	}
	var task domain.DeliveryTask
	if err := json.Unmarshal(items[0].Payload, &task); err != nil {
		t.Fatalf("outbox payload is not valid JSON: %v", err)
	}
	if task.ScheduleID != schedule.ID || task.ChatID != 7 || task.RuleID != rules[0].ID {
		t.Fatalf("unexpected delivery task: %+v", task)
	}

	acquired, err := storage.AcquireDelivery(ctx, task)
	if err != nil || !acquired {
		t.Fatalf("first AcquireDelivery = (%v, %v), want (true, nil)", acquired, err)
	}
	again, err := storage.AcquireDelivery(ctx, task)
	if err != nil {
		t.Fatalf("second AcquireDelivery error: %v", err)
	}
	if again {
		t.Fatal("a held delivery lease must not be acquired twice")
	}
}
