package main

import (
	"context"
	"fmt"
	"time"

	"github.com/escalopa/prayer-bot/notifier/internal"
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

	handler := internal.NewHandler(db, queue)

	botLocation := make(map[int32]*time.Location, len(botConfig))
	for botID, config := range botConfig {
		location, err := time.LoadLocation(config.Location)
		if err != nil {
			return fmt.Errorf("load timezone location: %q bot_id %d: %v", config.Location, botID, err)
		}
		botLocation[botID] = location
	}

	errG := &errgroup.Group{}
	for botID, loc := range botLocation {
		botID, loc := botID, loc
		errG.Go(func() error {
			return handler.Do(ctx, botID, loc)
		})
	}

	err = errG.Wait()
	if err != nil {
		return fmt.Errorf("run: %v", err)
	}

	return nil
}
