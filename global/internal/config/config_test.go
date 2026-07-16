package config

import (
	"strings"
	"testing"

	"github.com/escalopa/prayer-bot/global/internal/database"
)

func TestValidWebhookSecret(t *testing.T) {
	for _, value := range []string{"abc-123_DEF", "x"} {
		if !validWebhookSecret(value) {
			t.Fatalf("expected %q to be valid", value)
		}
	}
	for _, value := range []string{"", "contains space", "contains.dot"} {
		if validWebhookSecret(value) {
			t.Fatalf("expected %q to be invalid", value)
		}
	}
}

func TestLoadRejectsUnsupportedDatabaseSchema(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("GLOBAL_DB_SCHEMA", "public")
	t.Setenv("GLOBAL_BOT_TOKEN", "token")

	_, err := Load("send")
	if err == nil || !strings.Contains(err.Error(), "GLOBAL_DB_SCHEMA") {
		t.Fatalf("expected database schema validation error, got %v", err)
	}
}

func TestLoadAcceptsEnvironmentDatabaseSchema(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("GLOBAL_DB_SCHEMA", database.TestingSchema)
	t.Setenv("GLOBAL_BOT_TOKEN", "token")

	cfg, err := Load("send")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.DatabaseSchema != database.TestingSchema {
		t.Fatalf("expected %q, got %q", database.TestingSchema, cfg.DatabaseSchema)
	}
}
