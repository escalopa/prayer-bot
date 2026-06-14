//go:build gcp

package loader

import (
	"context"
	"fmt"
	"sync"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/escalopa/prayer-bot/config"
	"github.com/escalopa/prayer-bot/loader/internal/handler"
	"github.com/escalopa/prayer-bot/loader/internal/service"
	"github.com/escalopa/prayer-bot/log"
)

func init() {
	functions.CloudEvent("LoaderCloudEvent", LoaderCloudEvent)
}

type gcsObjectData struct {
	Bucket string `json:"bucket"`
	Name   string `json:"name"`
}

var (
	loaderOnce sync.Once
	loaderH    *handler.Handler
	loaderErr  error
)

func getLoaderHandler(ctx context.Context) (*handler.Handler, error) {
	loaderOnce.Do(func() {
		botConfig, err := config.Load()
		if err != nil {
			loaderErr = fmt.Errorf("load config: %w", err)
			return
		}

		storage, err := service.NewStorage()
		if err != nil {
			loaderErr = err
			return
		}

		db, err := service.NewDB(ctx)
		if err != nil {
			loaderErr = err
			return
		}

		loaderH = handler.New(botConfig, storage, db)
	})

	return loaderH, loaderErr
}

func LoaderCloudEvent(ctx context.Context, e cloudevents.Event) error {
	var data gcsObjectData
	if err := e.DataAs(&data); err != nil {
		log.Error("parse cloud event data", log.Err(err))
		return fmt.Errorf("parse event data: %w", err)
	}

	h, err := getLoaderHandler(ctx)
	if err != nil {
		log.Error("init loader handler", log.Err(err))
		return err
	}

	if err := h.Handel(ctx, data.Bucket, data.Name); err != nil {
		log.Error("loader cannot process request",
			log.Err(err),
			log.String("bucket", data.Bucket),
			log.String("key", data.Name),
		)
		return err
	}

	return nil
}
