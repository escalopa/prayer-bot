package function

import (
	"context"
	"fmt"
	"sync"
	"time"

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

const loaderInitTimeout = 30 * time.Second

var (
	loaderOnce sync.Once
	loaderH    *handler.Handler
	loaderErr  error
)

func getLoaderHandler() (*handler.Handler, error) {
	loaderOnce.Do(func() {
		initCtx, cancel := context.WithTimeout(context.Background(), loaderInitTimeout)
		defer cancel()

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

		db, err := service.NewDB(initCtx)
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
		log.Error("loader.gcp.parseEvent: failed",
			log.Op("parseEvent"), log.Err(err))
		return fmt.Errorf("parse event data: %w", err)
	}

	h, err := getLoaderHandler()
	if err != nil {
		log.Error("loader.gcp.initHandler: failed",
			log.Op("initHandler"), log.Err(err))
		return err
	}

	if err := h.Handel(ctx, data.Bucket, data.Name); err != nil {
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
