package db

import (
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("DB_PRIMARY", "")
	t.Setenv("DUAL_WRITE", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SUPABASE_DB_URL", "")

	cfg := LoadConfig()
	if cfg.Primary != "ydb" {
		t.Fatalf("expected primary ydb, got %q", cfg.Primary)
	}
	if cfg.DualWrite {
		t.Fatal("expected dual write disabled by default")
	}
}

func TestLoadConfigPostgresDualWrite(t *testing.T) {
	t.Setenv("DB_PRIMARY", "postgres")
	t.Setenv("DUAL_WRITE", "true")
	t.Setenv("SUPABASE_DB_URL", "postgres://example")

	cfg := LoadConfig()
	if cfg.Primary != "postgres" {
		t.Fatalf("expected primary postgres, got %q", cfg.Primary)
	}
	if !cfg.DualWrite {
		t.Fatal("expected dual write enabled")
	}
	if cfg.DatabaseURL != "postgres://example" {
		t.Fatalf("unexpected database url %q", cfg.DatabaseURL)
	}
}
