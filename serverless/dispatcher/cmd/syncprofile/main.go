// Command syncprofile applies Telegram bot name, descriptions, and commands for each bot
// in the config file. Intended to run from CI after deploy (see .github/workflows/deploy.yaml).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/escalopa/prayer-bot/dispatcher/internal/botprofile"
	"github.com/escalopa/prayer-bot/domain"
	"github.com/go-telegram/bot"
)

const (
	syncAttempts = 4
	syncBackoff  = 500 * time.Millisecond
)

func main() {
	configPath := flag.String("config", "", "path to bot config JSON (default: $APP_CONFIG_PATH or config.json)")
	flag.Parse()

	path := *configPath
	if path == "" {
		path = os.Getenv("APP_CONFIG_PATH")
	}
	if path == "" {
		path = "config.json"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read config: %v\n", err)
		os.Exit(1)
	}

	var cfg map[int64]*domain.BotConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "parse config: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	exit := 0
	for id, botCfg := range cfg {
		if botCfg == nil || botCfg.Token == "" {
			fmt.Fprintf(os.Stderr, "skip bot_id %d: missing token\n", id)
			exit = 1
			continue
		}
		b, err := bot.New(botCfg.Token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bot.New bot_id %d: %v\n", id, err)
			exit = 1
			continue
		}
		if err := botprofile.SyncWithRetry(ctx, b, botCfg.OwnerID, syncAttempts, syncBackoff); err != nil {
			if errors.Is(err, botprofile.ErrRateLimited) {
				fmt.Fprintf(os.Stderr, "warning: Telegram profile sync rate limited for bot_id %d; skipping until a future deployment: %v\n", id, err)
				continue
			}
			// Profile sync is best-effort: a transient Telegram/network failure
			// (e.g. an empty API response) must not fail the deploy. Genuine
			// errors (bad token, invalid params) still do.
			if errors.Is(err, botprofile.ErrTransient) {
				fmt.Fprintf(os.Stderr, "warning: profile sync skipped for bot_id %d after %d attempts: %v\n", id, syncAttempts, err)
				continue
			}
			fmt.Fprintf(os.Stderr, "botprofile.Sync bot_id %d: %v\n", id, err)
			exit = 1
			continue
		}
		fmt.Printf("telegram profile synced for bot_id %d\n", id)
	}
	os.Exit(exit)
}
