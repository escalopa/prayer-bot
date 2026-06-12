package db

import (
	"os"
	"strings"
)

type Config struct {
	Primary     string
	DualWrite   bool
	DatabaseURL string
	YDBEndpoint string
	YDBToken    string
}

func LoadConfig() Config {
	primary := strings.ToLower(strings.TrimSpace(os.Getenv("DB_PRIMARY")))
	if primary == "" {
		primary = "ydb"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("SUPABASE_DB_URL")
	}

	dualWrite := strings.EqualFold(os.Getenv("DUAL_WRITE"), "true") ||
		os.Getenv("DUAL_WRITE") == "1"

	return Config{
		Primary:     primary,
		DualWrite:   dualWrite,
		DatabaseURL: databaseURL,
		YDBEndpoint: os.Getenv("YDB_ENDPOINT"),
		YDBToken:    os.Getenv("YDB_TOKEN"),
	}
}
