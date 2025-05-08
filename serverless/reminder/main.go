package main

import (
	"context"
	"fmt"

	"github.com/escalopa/prayer-bot/reminder/internal"
	"github.com/escalopa/prayer-bot/service"
	"golang.org/x/sync/errgroup"
)

func Handler(ctx context.Context) error {
	storage, err := service.NewStorage()
	if err != nil {
		return fmt.Errorf("create storage: %v", err)
	}

	db, err := service.NewDB(ctx)
	if err != nil {
		return fmt.Errorf("create db: %v", err)
	}

	queue, err := service.NewQueue()
	if err != nil {
		return fmt.Errorf("create queue: %v", err)
	}

	botConfig, err := storage.LoadBotConfig(ctx)
	if err != nil {
		return fmt.Errorf("load botConfig: %v", err)
	}

	handler := internal.NewHandler(botConfig, db, queue)

	errG := &errgroup.Group{}
	for botID := range botConfig {
		botID := botID
		errG.Go(func() error {
			if err := handler.Do(ctx, botID); err != nil {
				return fmt.Errorf("handler do: %v", err)
			}
			return nil
		})
	}

	_ = errG.Wait()
	return nil
}
