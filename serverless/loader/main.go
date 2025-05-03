package main

import (
	"context"
	"fmt"
	"time"

	"github.com/escalopa/prayer-bot/loader/internal"
	"github.com/escalopa/prayer-bot/service"
)

type (
	Event struct {
		Messages []struct {
			EventMetadata struct {
				EventId        string    `json:"event_id"`
				EventType      string    `json:"event_type"`
				CreatedAt      time.Time `json:"created_at"`
				TracingContext struct {
					TraceId      string `json:"trace_id"`
					SpanId       string `json:"span_id"`
					ParentSpanId string `json:"parent_span_id"`
				} `json:"tracing_context"`
				CloudId  string `json:"cloud_id"`
				FolderId string `json:"folder_id"`
			} `json:"event_metadata"`
			Details struct {
				BucketId string `json:"bucket_id"`
				ObjectId string `json:"object_id"`
			} `json:"details"`
		} `json:"messages"`
	}
)

func Handler(ctx context.Context, event *Event) error {
	storage, err := service.NewStorage()
	if err != nil {
		return fmt.Errorf("create storage client: %v", err)
	}

	db, err := service.NewDB(ctx)
	if err != nil {
		return fmt.Errorf("create db connection: %v", err)
	}

	botConfig, err := storage.LoadBotConfig(ctx)
	if err != nil {
		return fmt.Errorf("load bot config: %v", err)
	}

	handler := internal.NewHandler(botConfig, storage, db)

	for _, msg := range event.Messages {
		bucket := msg.Details.BucketId
		key := msg.Details.ObjectId

		err = handler.Process(ctx, bucket, key)
		if err != nil {
			return err
		}
	}

	return nil
}
