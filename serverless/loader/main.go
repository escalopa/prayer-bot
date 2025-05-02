package main

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/escalopa/prayer-bot/loader/internal"
	"github.com/escalopa/prayer-bot/service"
)

type (
	Storage interface {
		Get(ctx context.Context, bucket string, key string) ([]byte, error)
		LoadBotConfig(ctx context.Context) (map[uint8]*service.BotConfig, error)
	}

	DB interface {
		StorePrayers(ctx context.Context, botID uint8, rows []*domain.PrayerTimes) error
	}

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

const (
	filenameSuffix = ".csv"
)

func Handler(ctx context.Context, event *Event) error {
	fmt.Printf("event received: %v\n", event)

	var (
		storage Storage
		db      DB
		err     error
	)

	storage, err = service.NewStorage()
	if err != nil {
		return fmt.Errorf("create storage client: %w", err)
	}

	db, err = service.NewDB(ctx)
	if err != nil {
		return fmt.Errorf("create db connection: %w", err)
	}

	botConfig, err := storage.LoadBotConfig(ctx)
	if err != nil {
		return fmt.Errorf("load bot config: %w", err)
	}

	for _, msg := range event.Messages {
		bucket := msg.Details.BucketId
		key := msg.Details.ObjectId

		// ignore non csv files
		if !strings.HasSuffix(key, filenameSuffix) {
			fmt.Printf("ignore file: %s\n", key)
			continue
		}

		fmt.Printf("processing file: %s\n", key)

		botID, err := internal.ExtractBotID(key)
		if err != nil {
			return fmt.Errorf("extract info from filename: %s => %w", key, err)
		}

		_, ok := botConfig[botID]
		if !ok {
			return fmt.Errorf("bot config not found for bot_id: %d", botID)
		}

		data, err := storage.Get(ctx, bucket, key)
		if err != nil {
			return fmt.Errorf("get file from S3: %s => %w", key, err)
		}

		rows, err := internal.ParsePrayers(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("load schedule: %s => %w", key, err)
		}

		err = db.StorePrayers(ctx, botID, rows)
		if err != nil {
			return fmt.Errorf("store prayers: %s => %w", key, err)
		}

		fmt.Printf("processed file for bot_id: %d\n", botID)
	}

	return nil
}
