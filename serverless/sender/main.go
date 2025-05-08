package main

import (
	"context"
	"fmt"
	"time"

	"github.com/escalopa/prayer-bot/sender/internal"
	"github.com/escalopa/prayer-bot/service"
)

type Event struct {
	Messages []struct {
		EventMetadata struct {
			EventID   string    `json:"event_id"`
			EventType string    `json:"event_type"`
			CreatedAt time.Time `json:"created_at"`
			CloudID   string    `json:"cloud_id"`
			FolderID  string    `json:"folder_id"`
		} `json:"event_metadata"`
		Details struct {
			QueueID string `json:"queue_id"`
			Message struct {
				MessageID  string `json:"message_id"`
				Md5OfBody  string `json:"md5_of_body"`
				Body       string `json:"body"`
				Attributes struct {
					SentTimestamp string `json:"SentTimestamp"`
				} `json:"attributes"`
				MessageAttributes struct {
					MessageAttributeKey struct {
						DataType    string `json:"data_type"`
						StringValue string `json:"string_value"`
					} `json:"messageAttributeKey"`
				} `json:"message_attributes"`
				Md5OfMessageAttributes string `json:"md5_of_message_attributes"`
			} `json:"message"`
		} `json:"details"`
	} `json:"messages"`
}

func Handler(ctx context.Context, event *Event) error {
	storage, err := service.NewStorage()
	if err != nil {
		return fmt.Errorf("create storage: %v", err)
	}
	fmt.Println("storage created")

	db, err := service.NewDB(ctx)
	if err != nil {
		return fmt.Errorf("create db: %v", err)
	}

	fmt.Println("db created")

	botConfig, err := storage.LoadBotConfig(ctx)
	if err != nil {
		return fmt.Errorf("load botConfig: %v", err)
	}
	fmt.Println("botConfig loaded")

	handler, err := internal.NewHandler(botConfig, db)
	if err != nil {
		return fmt.Errorf("create handler: %v", err)
	}

	for _, msg := range event.Messages {
		body := msg.Details.Message.Body
		err = handler.Do(ctx, body)
		if err != nil {
			return err
		}
	}

	return nil
}
