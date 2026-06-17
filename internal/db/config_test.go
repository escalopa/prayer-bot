package db

import (
	"testing"
)

func TestLoadConfigDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SUPABASE_DB_URL", "postgres://example")

	cfg := LoadConfig()
	if cfg.DatabaseURL != "postgres://example" {
		t.Fatalf("unexpected database url %q", cfg.DatabaseURL)
	}
}

func TestLoadConfigPrefersDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://primary")
	t.Setenv("SUPABASE_DB_URL", "postgres://fallback")

	cfg := LoadConfig()
	if cfg.DatabaseURL != "postgres://primary" {
		t.Fatalf("unexpected database url %q", cfg.DatabaseURL)
	}
}
