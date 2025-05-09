package main

import (
	"context"
	"fmt"
	"time"

	"github.com/escalopa/prayer-bot/log"

	"github.com/escalopa/prayer-bot/loader/internal"
	"github.com/escalopa/prayer-bot/service"
)

type (
	Event struct {
		Messages []struct {
			EventMetadata struct {
				EventID        string    `json:"event_id"`
				EventType      string    `json:"event_type"`
				CreatedAt      time.Time `json:"created_at"`
				TracingContext struct {
					TraceID      string `json:"trace_id"`
					SpanID       string `json:"span_id"`
					ParentSpanID string `json:"parent_span_id"`
				} `json:"tracing_context"`
				CloudID  string `json:"cloud_id"`
				FolderID string `json:"folder_id"`
			} `json:"event_metadata"`
			Details struct {
				BucketID string `json:"bucket_id"`
				ObjectID string `json:"object_id"`
			} `json:"details"`
		} `json:"messages"`
	}
)

func Handler(ctx context.Context, event *Event) error {
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

	botConfig, err := storage.LoadBotConfig(ctx)
	if err != nil {
		log.Error("load botConfig", log.Err(err))
		return fmt.Errorf("load botConfig")
	}

	handler := internal.NewHandler(botConfig, storage, db)

	for _, msg := range event.Messages {
		bucket := msg.Details.BucketID
		key := msg.Details.ObjectID

		err = handler.Do(ctx, bucket, key)
		if err != nil {
			log.Error("loader cannot process request",
				log.Err(err),
				log.String("bucket", bucket),
				log.String("key", key),
			)
			return err
		}
	}

	return nil
}
