package main

import (
	"context"
	"fmt"
	"time"

	"github.com/escalopa/prayer-bot/config"
	"github.com/escalopa/prayer-bot/loader/internal/handler"
	"github.com/escalopa/prayer-bot/loader/internal/service"
	"github.com/escalopa/prayer-bot/log"
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
	botConfig, err := config.Load()
	if err != nil {
		log.Error("loader.entry.loadConfig: failed",
			log.Op("loadConfig"), log.Err(err))
		return fmt.Errorf("load config")
	}

	storage, err := service.NewStorage()
	if err != nil {
		log.Error("loader.entry.createStorage: failed",
			log.Op("createStorage"), log.Err(err))
		return fmt.Errorf("create storage")
	}

	db, err := service.NewDB(ctx)
	if err != nil {
		log.Error("loader.entry.createDB: failed",
			log.Op("createDB"), log.Err(err))
		return fmt.Errorf("create db")
	}

	h := handler.New(botConfig, storage, db)

	for _, msg := range event.Messages {
		bucket := msg.Details.BucketID
		key := msg.Details.ObjectID

		err = h.Handel(ctx, bucket, key)
		if err != nil {
			log.Error("loader.entry.processObject: handler failed",
				log.Op("processObject"),
				log.Err(err),
				log.String("bucket", bucket),
				log.String("key", key),
			)
			return err
		}
	}

	return nil
}
