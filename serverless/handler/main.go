package main

import (
	"context"
	"fmt"

	"github.com/escalopa/prayer-bot/handler/internal"
	"github.com/escalopa/prayer-bot/service"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type (
	Storage interface {
		LoadBotConfig(ctx context.Context) (map[uint8]*service.BotConfig, error)
	}

	Queue interface {
		Push(ctx context.Context, event models.Update) error
	}
)

func Handler(ctx context.Context, request []byte) error {
	update, headers, err := internal.ParseRequest(request)
	if err != nil {
		return fmt.Errorf("parse request: %v", err)
	}

	var (
		storage Storage
	)

	storage, err = service.NewStorage()
	if err != nil {
		return fmt.Errorf("create S3 client: %w", err)
	}

	botConfig, err := storage.LoadBotConfig(ctx)
	if err != nil {
		return fmt.Errorf("load bot config: %v", err)
	}

	botID, token, err := internal.Authenticate(botConfig, headers)
	if err != nil {
		fmt.Printf("authenticate: %v\n", err)
		return nil // hide error from user
	}
	_ = botID // TODO: use it

	opts := []bot.Option{
		bot.WithDefaultHandler(handler),
	}

	b, err := bot.New(token, opts...)
	if err != nil {
		return fmt.Errorf("create bot: %v", err)
	}

	b.ProcessUpdate(ctx, update)
	return nil
}

func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message != nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   update.Message.Text,
		})
	}
}
