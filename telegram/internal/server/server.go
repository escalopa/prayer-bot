package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
)

type Handler interface {
	Run(ctx context.Context) error
	Stop()
}

type App struct {
	ctx    context.Context
	cancel context.CancelFunc

	handlers []Handler
}

func New() *App {
	return &App{}
}

func (a *App) AddHandler(handler Handler) {
	a.handlers = append(a.handlers, handler)
}

func (a *App) Run(ctx context.Context, port string) {
	a.setShutdownCtx(ctx)
	for _, handler := range a.handlers {
		err := handler.Run(ctx)
		if err != nil {
			panic(err) // TODO: FIX THIS
		}
	}
	a.health(port)
	a.listenForShutdown()
}

func (a *App) listenForShutdown() {
	<-a.ctx.Done()
	a.shutdown()
}

func (a *App) shutdown() {
	for _, handler := range a.handlers {
		handler.Stop()
	}
}

func (a *App) setShutdownCtx(ctx context.Context) {
	a.ctx, a.cancel = signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
}

func (a *App) health(port string) {
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	log.Printf("starting server on port: %s", port)
	err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err) // TODO: FIX THIS
	}
}
