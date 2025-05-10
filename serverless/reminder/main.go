package main

import (
	"context"
	"fmt"

	"github.com/escalopa/prayer-bot/config"
	"github.com/escalopa/prayer-bot/log"
	"github.com/escalopa/prayer-bot/reminder/internal/handler"
	"github.com/escalopa/prayer-bot/reminder/internal/service"
	"golang.org/x/sync/errgroup"
)

func Handler(ctx context.Context) error {
	botConfig, err := config.Load()
	if err != nil {
		log.Error("load config", log.Err(err))
		return fmt.Errorf("load config")
	}

	db, err := service.NewDB(ctx)
	if err != nil {
		log.Error("create db", log.Err(err))
		return fmt.Errorf("create db")
	}

	h, err := handler.New(botConfig, db)
	if err != nil {
		log.Error("create handler", log.Err(err))
		return fmt.Errorf("create handler")
	}

	errG := &errgroup.Group{}
	for botID := range botConfig {
		botID := botID
		errG.Go(func() error {
			err := h.Handel(ctx, botID)
			if err != nil {
				log.Error("reminder cannot process request", log.BotID(botID), log.Err(err))
			}
			return nil
		})
	}

	_ = errG.Wait()
	return nil
}
