package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	botapi "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

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
		AllowedUpdates: []string{models.AllowedUpdateMessage},
	}); err != nil {
		fatal(fmt.Errorf("set webhook failed"))
	}
	if _, err := client.SetMyCommands(ctx, &botapi.SetMyCommandsParams{Commands: []models.BotCommand{
		{Command: "location", Description: "Set or replace prayer location"},
		{Command: "today", Description: "Show today's prayer times"},
		{Command: "tomorrow", Description: "Show tomorrow's prayer times"},
		{Command: "next", Description: "Show the next prayer"},
		{Command: "settings", Description: "Show calculation settings"},
		{Command: "remind", Description: "Enable or disable reminders"},
		{Command: "privacy", Description: "See stored data and deletion"},
		{Command: "help", Description: "Show all commands"},
	}}); err != nil {
		fatal(fmt.Errorf("set commands failed"))
	}
}

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
