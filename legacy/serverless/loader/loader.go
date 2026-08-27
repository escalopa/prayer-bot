package function

import (
	"context"
	"fmt"
	"sync"

	_ "github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
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

func LoaderCloudEvent(parentCtx context.Context, e cloudevents.Event) error {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	var data gcsObjectData
	if err := e.DataAs(&data); err != nil {
		log.Error("loader.gcp.parseEvent: failed",
			log.Op("parseEvent"), log.Err(err))
		return fmt.Errorf("parse event data: %w", err)
	}

	h, err := getLoaderHandler(ctx)
	if err != nil {
		log.Error("loader.gcp.initHandler: failed",
			log.Op("initHandler"), log.Err(err))
		return err
	}

	if err := h.Handle(ctx, data.Bucket, data.Name); err != nil {
		log.Error("loader.gcp.processObject: handler failed",
			log.Op("processObject"),
			log.Err(err),
			log.String("bucket", data.Bucket),
			log.String("key", data.Name),
		)
		return err
	}

	return nil
}
