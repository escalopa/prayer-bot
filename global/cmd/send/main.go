package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	botapi "github.com/go-telegram/bot"

	"github.com/escalopa/prayer-bot/global/internal/config"
	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/httpx"
	"github.com/escalopa/prayer-bot/global/internal/prayertime"
	"github.com/escalopa/prayer-bot/global/internal/reminders"
	"github.com/escalopa/prayer-bot/global/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load("send")
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
	telegramBot, err := botapi.New(cfg.TelegramToken, botapi.WithSkipGetMe())
	if err != nil {
		logger.Error("Telegram client initialization failed", "error", err)
		os.Exit(1)
	}
	planner := reminders.NewPlanner(storage, prayertime.New())
	sender := reminders.NewSender(storage, planner, telegramBot)

	mux := http.NewServeMux()
	httpx.HealthMux(mux)
	mux.HandleFunc("POST /tasks/send", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		var task domain.DeliveryTask
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			http.Error(w, "invalid task", http.StatusBadRequest)
			return
		}
		if err := sender.Process(r.Context(), task); err != nil {
			logger.Error("notification delivery failed", "delivery_key", task.DeliveryKey, "error", err)
			http.Error(w, "temporary failure", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /tasks/delete", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		var task domain.MessageDeletionTask
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			http.Error(w, "invalid task", http.StatusBadRequest)
			return
		}
		if err := sender.Delete(r.Context(), task); err != nil {
			logger.Error("notification cleanup failed", "deletion_key", task.DeletionKey, "error", err)
			http.Error(w, "temporary failure", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	logger.Info("sender service listening", "port", cfg.Port)
	if err := httpx.Serve(cfg.Port, mux); err != nil {
		logger.Error("HTTP server failed", "error", err)
		os.Exit(1)
	}
}
