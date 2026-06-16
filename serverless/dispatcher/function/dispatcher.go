package function

import (
	"context"
	"io"
	"net/http"
	"sync"

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

var (
	dispatcherOnce sync.Once
	dispatcherH    *handler.Handler
	dispatcherErr  error
)

func getDispatcherHandler(ctx context.Context) (*handler.Handler, error) {
	dispatcherOnce.Do(func() {
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

	h, err := getDispatcherHandler(r.Context())
	if err != nil {
		log.Error("init dispatcher handler", log.Err(err))
		http.Error(w, "init handler", http.StatusInternalServerError)
		return
	}

	botID, err := h.Authenticate(headerMap(r.Header))
	if err != nil {
		log.Error("authenticate", log.Err(err))
		http.Error(w, "authenticate", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("read request body", log.Err(err))
		http.Error(w, "read request body", http.StatusBadRequest)
		return
	}

	if err := h.Handel(r.Context(), botID, string(body)); err != nil {
		log.Error("dispatcher cannot process request", log.Err(err))
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
