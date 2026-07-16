package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/escalopa/prayer-bot/global/internal/database"
)

func main() {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		fatal("DATABASE_URL is required")
	}
	databaseSchema := strings.TrimSpace(os.Getenv("GLOBAL_DB_SCHEMA"))
	if err := database.ValidateSchema(databaseSchema); err != nil {
		fatal("GLOBAL_DB_SCHEMA must select the testing or production global schema")
	}
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		fatal("database connection setup failed")
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		fatal("database connection setup failed")
	}
	defer pool.Close()
	query := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", pgx.Identifier{databaseSchema}.Sanitize())
	if _, err := pool.Exec(context.Background(), query); err != nil {
		fatal("global database schema bootstrap failed")
	}
}

func fatal(message string) {
	_, _ = fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
