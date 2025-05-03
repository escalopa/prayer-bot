package main

import (
	"context"
	"fmt"

	"github.com/escalopa/prayer-bot/handler/internal"
	"github.com/escalopa/prayer-bot/service"
	"github.com/go-telegram/bot"
)

type Response struct {
	StatusCode int    `json:"status_code"`
	Body       string `json:"body"`
}

func Handler(ctx context.Context, request []byte) (*Response, error) {
	update, headers, err := internal.ParseRequest(request)
	if err != nil {
		return nil, fmt.Errorf("parse request: %v", err)
	}

	storage, err := service.NewStorage()
	if err != nil {
		return nil, fmt.Errorf("create S3 client: %v", err)
	}

	queue, err := service.NewQueue()
	if err != nil {
		return nil, fmt.Errorf("create queue: %v", err)
	}

	botConfig, err := storage.LoadBotConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load bot config: %v", err)
	}

	botID, token, err := internal.Authenticate(botConfig, headers)
	if err != nil {
		return nil, fmt.Errorf("authenticate: %v\n", err)
	}

	handler := internal.NewHandler(botID, queue)

	b, err := bot.New(token, handler.Opts()...)
	if err != nil {
		return nil, fmt.Errorf("create bot: %v", err)
	}

	b.ProcessUpdate(ctx, update)

	return &Response{StatusCode: 200, Body: "OK"}, nil
}
