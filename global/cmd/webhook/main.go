package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	botapi "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/escalopa/prayer-bot/global/internal/config"
	"github.com/escalopa/prayer-bot/global/internal/httpx"
	"github.com/escalopa/prayer-bot/global/internal/location"
	"github.com/escalopa/prayer-bot/global/internal/miniapp"
	"github.com/escalopa/prayer-bot/global/internal/prayertime"
	"github.com/escalopa/prayer-bot/global/internal/reminders"
	"github.com/escalopa/prayer-bot/global/internal/store"
	telegramhandler "github.com/escalopa/prayer-bot/global/internal/telegram"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load("webhook")
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
	calculator := prayertime.New()
	planner := reminders.NewPlanner(storage, calculator)
	resolver := location.NewGoogleMaps(cfg.GoogleMapsAPIKey, cfg.HTTPTimeout)
	handler := telegramhandler.NewHandler(
		telegramBot, storage, resolver, calculator, planner, cfg.OwnerID,
	)
	miniApp := miniapp.NewHandler(cfg.TelegramToken, storage, resolver, calculator, planner, logger, telegramBot)

	mux := http.NewServeMux()
	httpx.HealthMux(mux)
	miniApp.Register(mux)
	mux.HandleFunc("POST /telegram/webhook", func(w http.ResponseWriter, r *http.Request) {
		provided := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
		if len(provided) != len(cfg.WebhookSecret) || subtle.ConstantTimeCompare([]byte(provided), []byte(cfg.WebhookSecret)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		var update models.Update
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, "invalid update", http.StatusBadRequest)
			return
		}
		acquired, err := storage.AcquireUpdate(r.Context(), update.ID)
		if err != nil {
			logger.Error("update acquisition failed", "update_id", update.ID, "error", err)
			http.Error(w, "temporary failure", http.StatusInternalServerError)
			return
		}
		if !acquired {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err := handler.Handle(r.Context(), update); err != nil {
			_ = storage.FailUpdate(r.Context(), update.ID, err)
			logger.Error("update handling failed", "update_id", update.ID, "error", err)
			http.Error(w, "temporary failure", http.StatusInternalServerError)
			return
		}
		if err := storage.CompleteUpdate(r.Context(), update.ID); err != nil {
			logger.Error("update completion failed", "update_id", update.ID, "error", err)
			http.Error(w, "temporary failure", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	logger.Info("webhook service listening", "port", cfg.Port)
	if err := httpx.Serve(cfg.Port, mux); err != nil {
		logger.Error("HTTP server failed", "error", err)
		os.Exit(1)
	}
}
