package db

import (
	"os"
)

type Config struct {
	DatabaseURL string
}

func LoadConfig() Config {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("SUPABASE_DB_URL")
	}

	return Config{
		DatabaseURL: databaseURL,
	}
}
