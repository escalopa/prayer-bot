package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/database"
)

type Config struct {
	Port                     string
	DatabaseURL              string
	DatabaseSchema           string
	TelegramToken            string
	WebhookSecret            string
	OwnerID                  int64
	GoogleMapsAPIKey         string
	MiniAppURL               string
	GCPProjectID             string
	GCPRegion                string
	CloudTasksQueue          string
	SenderURL                string
	TaskCallerServiceAccount string
	DispatchBatchSize        int
	HTTPTimeout              time.Duration
}

func Load(service string) (Config, error) {
	cfg := Config{
		Port:                     envOr("PORT", "8080"),
		DatabaseURL:              strings.TrimSpace(os.Getenv("DATABASE_URL")),
		DatabaseSchema:           strings.TrimSpace(os.Getenv("GLOBAL_DB_SCHEMA")),
		TelegramToken:            strings.TrimSpace(os.Getenv("GLOBAL_BOT_TOKEN")),
		WebhookSecret:            strings.TrimSpace(os.Getenv("GLOBAL_WEBHOOK_SECRET")),
		GoogleMapsAPIKey:         strings.TrimSpace(os.Getenv("GOOGLE_MAPS_API_KEY")),
		MiniAppURL:               strings.TrimSpace(os.Getenv("MINI_APP_URL")),
		GCPProjectID:             strings.TrimSpace(os.Getenv("GCP_PROJECT_ID")),
		GCPRegion:                envOr("GCP_REGION", "europe-west1"),
		CloudTasksQueue:          envOr("CLOUD_TASKS_QUEUE", "global-prayer-notifications"),
		SenderURL:                strings.TrimSpace(os.Getenv("GLOBAL_SENDER_URL")),
		TaskCallerServiceAccount: strings.TrimSpace(os.Getenv("TASK_CALLER_SERVICE_ACCOUNT")),
		DispatchBatchSize:        envInt("DISPATCH_BATCH_SIZE", 100),
		HTTPTimeout:              time.Duration(envInt("HTTP_TIMEOUT_SECONDS", 10)) * time.Second,
	}

	if raw := strings.TrimSpace(os.Getenv("GLOBAL_OWNER_ID")); raw != "" {
		ownerID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("parse GLOBAL_OWNER_ID: %w", err)
		}
		cfg.OwnerID = ownerID
	}

	switch service {
	case "webhook":
		if cfg.DatabaseURL == "" || cfg.DatabaseSchema == "" || cfg.TelegramToken == "" || cfg.WebhookSecret == "" || cfg.GoogleMapsAPIKey == "" || cfg.OwnerID == 0 {
			return Config{}, fmt.Errorf("webhook requires DATABASE_URL, GLOBAL_DB_SCHEMA, GLOBAL_BOT_TOKEN, GLOBAL_WEBHOOK_SECRET, GLOBAL_OWNER_ID, and GOOGLE_MAPS_API_KEY")
		}
		if !validWebhookSecret(cfg.WebhookSecret) {
			return Config{}, fmt.Errorf("GLOBAL_WEBHOOK_SECRET must be 1-256 characters using only letters, numbers, underscore, or hyphen")
		}
	case "dispatch":
		if cfg.DatabaseURL == "" || cfg.DatabaseSchema == "" || cfg.GCPProjectID == "" || cfg.SenderURL == "" || cfg.TaskCallerServiceAccount == "" {
			return Config{}, fmt.Errorf("dispatch requires DATABASE_URL, GLOBAL_DB_SCHEMA, GCP_PROJECT_ID, GLOBAL_SENDER_URL, and TASK_CALLER_SERVICE_ACCOUNT")
		}
	case "send":
		if cfg.DatabaseURL == "" || cfg.DatabaseSchema == "" || cfg.TelegramToken == "" {
			return Config{}, fmt.Errorf("send requires DATABASE_URL, GLOBAL_DB_SCHEMA, and GLOBAL_BOT_TOKEN")
		}
	case "botprofile":
		if cfg.TelegramToken == "" || cfg.WebhookSecret == "" || cfg.MiniAppURL == "" {
			return Config{}, fmt.Errorf("botprofile requires GLOBAL_BOT_TOKEN, GLOBAL_WEBHOOK_SECRET, and MINI_APP_URL")
		}
		if !validWebhookSecret(cfg.WebhookSecret) {
			return Config{}, fmt.Errorf("GLOBAL_WEBHOOK_SECRET must be 1-256 characters using only letters, numbers, underscore, or hyphen")
		}
		if !validMiniAppURL(cfg.MiniAppURL) {
			return Config{}, fmt.Errorf("MINI_APP_URL must be an HTTPS URL")
		}
	default:
		return Config{}, fmt.Errorf("unknown service %q", service)
	}
	if cfg.DatabaseSchema != "" {
		if err := database.ValidateSchema(cfg.DatabaseSchema); err != nil {
			return Config{}, fmt.Errorf("GLOBAL_DB_SCHEMA: %w", err)
		}
	}
	return cfg, nil
}

func validMiniAppURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil && parsed.Fragment == ""
}

func validWebhookSecret(value string) bool {
	if len(value) == 0 || len(value) > 256 {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || character == '_' || character == '-' {
			continue
		}
		return false
	}
	return true
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envInt(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
