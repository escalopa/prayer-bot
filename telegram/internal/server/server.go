package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	log "github.com/catalystgo/logger/cli"

	"github.com/pkg/errors"
)

type Handler interface {
	Run(ctx context.Context) error
	Stop()
}

type App struct {
	ctx    context.Context
	cancel context.CancelFunc

	handler Handler
}

func New() *App {
	return &App{}
}

func (a *App) Run(ctx context.Context, handler Handler, port string) {
	a.handler = handler
	a.setShutdownCtx(ctx)

	err := a.handler.Run(a.ctx)
	if err != nil {
		panic(err)
	}

	a.health(port)
	a.listenForShutdown()
}

func (a *App) listenForShutdown() {
	<-a.ctx.Done()
	log.Warnf("Shutting down server...")
	a.shutdown()
}

func (a *App) shutdown() {
	a.handler.Stop()
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

	log.Warnf("Server is running on port %s", port)
	err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("http.ListenAndServe: %v", err)
	}
}
