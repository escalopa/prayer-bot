package main

import (
	"context"
	"fmt"

	"github.com/escalopa/prayer-bot/log"

	"github.com/escalopa/prayer-bot/reminder/internal"
	"github.com/escalopa/prayer-bot/service"
	"golang.org/x/sync/errgroup"
)

func Handler(ctx context.Context) error {
	storage, err := service.NewStorage()
	if err != nil {
		log.Error("create storage", log.Err(err))
		return fmt.Errorf("create storage")
	}

	db, err := service.NewDB(ctx)
	if err != nil {
		log.Error("create db", log.Err(err))
		return fmt.Errorf("create db")
	}

	queue, err := service.NewQueue()
	if err != nil {
		log.Error("create queue", log.Err(err))
		return fmt.Errorf("create queue")
	}

	botConfig, err := storage.LoadBotConfig(ctx)
	if err != nil {
		log.Error("load botConfig", log.Err(err))
		return fmt.Errorf("load botConfig")
	}

	handler := internal.NewHandler(botConfig, db, queue)

	errG := &errgroup.Group{}
	for botID := range botConfig {
		botID := botID
		errG.Go(func() error {
			err := handler.Do(ctx, botID)
			if err != nil {
				log.Error("reminder cannot process request", log.BotID(botID), log.Err(err))
			}
			return nil
		})
	}

	_ = errG.Wait()
	return nil
}
