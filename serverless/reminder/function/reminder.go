package function

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	_ "github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/escalopa/prayer-bot/config"
	"github.com/escalopa/prayer-bot/log"
	"github.com/escalopa/prayer-bot/reminder/internal/handler"
	"github.com/escalopa/prayer-bot/reminder/internal/service"
	"golang.org/x/sync/errgroup"
)

func init() {
	functions.HTTP("ReminderHTTP", ReminderHTTP)
}

const (
	reminderInitTimeout    = 30 * time.Second
	reminderHandlerTimeout = 55 * time.Second
)

var (
	reminderOnce sync.Once
	reminderH    *handler.Handler
	reminderErr  error
)

func getReminderHandler() (*handler.Handler, error) {
	reminderOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), reminderInitTimeout)
		defer cancel()

		botConfig, err := config.Load()
		if err != nil {
			reminderErr = fmt.Errorf("load config: %w", err)
			return
		}

		db, err := service.NewDB(ctx)
		if err != nil {
			reminderErr = err
			return
		}

		reminderH, reminderErr = handler.New(botConfig, db)
	})

	return reminderH, reminderErr
}

func ReminderHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	h, err := getReminderHandler()
	if err != nil {
		log.Error("reminder.gcp.initHandler: failed",
			log.Op("initHandler"), log.Err(err))
		http.Error(w, "init handler", http.StatusInternalServerError)
		return
	}

	botConfig, err := config.Load()
	if err != nil {
		log.Error("reminder.gcp.loadConfig: failed",
			log.Op("loadConfig"), log.Err(err))
		http.Error(w, "load config", http.StatusInternalServerError)
		return
	}

	errG := &errgroup.Group{}
	for botID := range botConfig {
		botID := botID
		errG.Go(func() error {
			handlerCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), reminderHandlerTimeout)
			defer cancel()

			err := h.Handel(handlerCtx, botID)
			if err != nil {
				log.Error("reminder.gcp.processBot: handler failed",
					log.Op("processBot"), log.BotID(botID), log.Err(err))
			}
			return nil
		})
	}

	_ = errG.Wait()
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
