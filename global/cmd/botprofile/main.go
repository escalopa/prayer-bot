package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	botapi "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	profile "github.com/escalopa/prayer-bot/global/internal/botprofile"
	"github.com/escalopa/prayer-bot/global/internal/config"
)

func main() {
	cfg, err := config.Load("botprofile")
	if err != nil {
		fatal(err)
	}
	webhookURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WEBHOOK_URL")), "/")
	if webhookURL == "" {
		fatal(fmt.Errorf("WEBHOOK_URL is required"))
	}
	client, err := botapi.New(cfg.TelegramToken, botapi.WithSkipGetMe())
	if err != nil {
		fatal(err)
	}
	ctx := context.Background()
	if _, err := client.SetWebhook(ctx, &botapi.SetWebhookParams{
		URL: webhookURL + "/telegram/webhook", SecretToken: cfg.WebhookSecret,
		AllowedUpdates: []string{models.AllowedUpdateMessage, models.AllowedUpdateCallbackQuery},
	}); err != nil {
		fatal(fmt.Errorf("set webhook failed: %w", err))
	}
	if err := profile.Sync(ctx, client, cfg.TelegramToken, cfg.MiniAppURL); err != nil {
		if retryAfter, limited := profile.RateLimitRetryAfter(err); limited {
			fmt.Printf("PROFILE_SYNC_SKIPPED_RETRY_AFTER=%d\n", retryAfter)
			fmt.Fprintf(os.Stderr, "warning: Telegram profile sync rate limited; skipping profile updates and retrying on a future deployment: %v\n", err)
			return
		}
		fatal(fmt.Errorf("sync bot profile failed: %w", err))
	}
	fmt.Println("PROFILE_SYNC_STATUS=synchronized")
}

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
