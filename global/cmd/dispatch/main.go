package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/adapter/out/metals"
	"github.com/escalopa/prayer-bot/global/internal/adapter/out/store"
	"github.com/escalopa/prayer-bot/global/internal/config"
	"github.com/escalopa/prayer-bot/global/internal/core/reminders"
	"github.com/escalopa/prayer-bot/global/internal/httpx"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load("dispatch")
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}
	storage, err := store.Open(context.Background(), cfg.DatabaseURL, cfg.DatabaseSchema)
	if err != nil {
		logger.Error("database connection failed")
		os.Exit(1)
	}
	defer storage.Close()
	enqueuer, err := reminders.NewCloudTasksEnqueuer(context.Background(), cfg.GCPProjectID, cfg.GCPRegion,
		cfg.CloudTasksQueue, cfg.SenderURL, cfg.TaskCallerServiceAccount)
	if err != nil {
		logger.Error("Cloud Tasks client initialization failed", "error", err)
		os.Exit(1)
	}
	defer enqueuer.Close()
	dispatcher := reminders.NewDispatcher(storage, enqueuer, cfg.DispatchBatchSize)
	metalsClient := metals.NewClient(cfg.HTTPTimeout)

	mux := http.NewServeMux()
	httpx.HealthMux(mux)
	mux.HandleFunc("POST /dispatch", func(w http.ResponseWriter, r *http.Request) {
		count, err := dispatcher.Run(r.Context(), time.Now())
		if err != nil {
			logger.Error("reminder dispatch failed", "error", err)
			http.Error(w, "temporary failure", http.StatusInternalServerError)
			return
		}
		logger.Info("reminders dispatched", "count", count)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /maintenance", func(w http.ResponseWriter, r *http.Request) {
		// The niSab price refresh is best-effort: a failure logs a warning and
		// keeps the previously cached prices, and must not fail the request or
		// block retention cleanup.
		if prices, err := metalsClient.Fetch(r.Context()); err != nil {
			logger.Warn("metal price refresh failed; keeping cached prices", "error", err)
		} else if err := storage.UpsertMetalPrices(r.Context(), prices); err != nil {
			logger.Warn("metal price persist failed; keeping cached prices", "error", err)
		} else {
			logger.Info("metal prices refreshed")
		}
		count, err := storage.Cleanup(r.Context(), time.Now(), 1000)
		if err != nil {
			logger.Error("retention cleanup failed", "error", err)
			http.Error(w, "temporary failure", http.StatusInternalServerError)
			return
		}
		logger.Info("retention cleanup completed", "deleted", count)
		w.WriteHeader(http.StatusNoContent)
	})
	logger.Info("dispatch service listening", "port", cfg.Port)
	if err := httpx.Serve(cfg.Port, mux); err != nil {
		logger.Error("HTTP server failed", "error", err)
		os.Exit(1)
	}
}
