package function

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	_ "github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/escalopa/prayer-bot/config"
	"github.com/escalopa/prayer-bot/dispatcher/internal/handler"
	"github.com/escalopa/prayer-bot/dispatcher/internal/service"
	"github.com/escalopa/prayer-bot/log"
)

func init() {
	functions.HTTP("DispatcherHTTP", DispatcherHTTP)
}

const (
	dispatcherInitTimeout    = 30 * time.Second
	dispatcherHandlerTimeout = 55 * time.Second
)

var (
	dispatcherOnce sync.Once
	dispatcherH    *handler.Handler
	dispatcherErr  error
)

func getDispatcherHandler() (*handler.Handler, error) {
	dispatcherOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), dispatcherInitTimeout)
		defer cancel()

		botConfig, err := config.Load()
		if err != nil {
			dispatcherErr = err
			return
		}

		db, err := service.NewDB(ctx)
		if err != nil {
			dispatcherErr = err
			return
		}

		dispatcherH, dispatcherErr = handler.New(botConfig, db)
	})

	return dispatcherH, dispatcherErr
}

func DispatcherHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	h, err := getDispatcherHandler()
	if err != nil {
		log.Error("dispatcher.gcp.initHandler: failed",
			log.Op("initHandler"), log.Err(err))
		http.Error(w, "init handler", http.StatusInternalServerError)
		return
	}

	botID, err := h.Authenticate(headerMap(r.Header))
	if err != nil {
		log.Error("dispatcher.gcp.authenticate: failed",
			log.Op("authenticate"), log.Err(err))
		http.Error(w, "authenticate", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("dispatcher.gcp.readBody: failed",
			log.Op("readBody"), log.Err(err))
		http.Error(w, "read request body", http.StatusBadRequest)
		return
	}

	handlerCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), dispatcherHandlerTimeout)
	defer cancel()

	if err := h.Handel(handlerCtx, botID, string(body)); err != nil {
		log.Error("dispatcher.gcp.processRequest: handler failed",
			log.Op("processRequest"), log.Err(err))
		http.Error(w, "process request", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("success"))
}

func headerMap(header http.Header) map[string]string {
	out := make(map[string]string, len(header))
	for key, values := range header {
		if len(values) > 0 {
			out[key] = values[0]
		}
	}
	return out
}
