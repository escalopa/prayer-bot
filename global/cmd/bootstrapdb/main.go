package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		fatal("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		fatal("database connection setup failed")
	}
	defer pool.Close()
	if _, err := pool.Exec(context.Background(), `CREATE SCHEMA IF NOT EXISTS global_bot`); err != nil {
		fatal("global_bot schema bootstrap failed")
	}
}

func fatal(message string) {
	_, _ = fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
