package handler

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/escalopa/prayer-bot/log"

	"github.com/escalopa/prayer-bot/domain"
)

const (
	filenameSuffix = ".csv"
)

type (
	Storage interface {
		Get(ctx context.Context, bucket string, key string) ([]byte, error)
	}

	DB interface {
		SetPrayerDays(ctx context.Context, botID int64, prayerDays []*domain.PrayerDay) error
	}

	Handler struct {
		config  map[int64]*domain.BotConfig
		storage Storage
		db      DB
	}
)

func New(config map[int64]*domain.BotConfig, storage Storage, db DB) *Handler {
	return &Handler{
		config:  config,
		storage: storage,
		db:      db,
	}
}

func (h Handler) Handel(ctx context.Context, bucket string, key string) error {
	if !strings.HasSuffix(key, filenameSuffix) { // ignore non csv files
		return nil
	}

	botID, err := extractBotID(key)
	if err != nil {
		return fmt.Errorf("extract info from filename: %v", err)
	}

	cfg, ok := h.config[botID]
	if !ok {
		return fmt.Errorf("bot config not found")
	}

	data, err := h.storage.Get(ctx, bucket, key)
	if err != nil {
		return fmt.Errorf("get file from storage: %v", err)
	}

	rows, err := parsePrayerDays(bytes.NewReader(data), cfg.Location.V())
	if err != nil {
		return fmt.Errorf("load schedule: %v", err)
	}

	err = h.db.SetPrayerDays(ctx, botID, rows)
	if err != nil {
		return fmt.Errorf("store prayers: %v", err)
	}

	log.Info("set prayers done", log.BotID(botID), log.String("key", key))
	return nil
}

func extractBotID(key string) (int64, error) {
	botIDStr := strings.TrimSuffix(path.Base(key), filenameSuffix)
	botID, err := strconv.ParseInt(botIDStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse bot_id: %v", err)
	}
	return botID, nil
}
