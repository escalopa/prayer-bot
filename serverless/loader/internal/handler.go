package internal

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/escalopa/prayer-bot/domain"
)

const (
	filenameSuffix   = ".csv"
	filenameParts    = 2
	filenameSplitter = "-"
)

type (
	Storage interface {
		Get(ctx context.Context, bucket string, key string) ([]byte, error)
	}

	DB interface {
		SetPrayerDays(ctx context.Context, botID int32, prayerDays []*domain.PrayerDay) error
	}

	Handler struct {
		config  map[int32]*domain.BotConfig
		storage Storage
		db      DB
	}
)

func NewHandler(config map[int32]*domain.BotConfig, storage Storage, db DB) *Handler {
	return &Handler{
		config:  config,
		storage: storage,
		db:      db,
	}
}

func (h Handler) Do(ctx context.Context, bucket string, key string) error {
	if !strings.HasSuffix(key, filenameSuffix) { // ignore non csv files
		return nil
	}

	botID, err := extractBotID(key)
	if err != nil {
		return fmt.Errorf("extract info from filename: %q: %v", key, err)
	}

	_, ok := h.config[botID]
	if !ok {
		return fmt.Errorf("bot config not found for bot_id: %d", botID)
	}

	data, err := h.storage.Get(ctx, bucket, key)
	if err != nil {
		return fmt.Errorf("get file from S3: %q: %v", key, err)
	}

	rows, err := parsePrayerDays(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("load schedule: %q: %v", key, err)
	}

	err = h.db.SetPrayerDays(ctx, botID, rows)
	if err != nil {
		return fmt.Errorf("store prayers: %s: %v", key, err)
	}

	fmt.Printf("processed file: %q bot_id: %d\n", key, botID)
	return nil
}

func extractBotID(filename string) (int32, error) {
	parts := strings.Split(path.Base(filename), filenameSplitter)
	if len(parts) != filenameParts {
		return 0, fmt.Errorf("unexpected filename format: %q", filename)
	}

	botID, err := strconv.ParseUint(parts[0], 10, 8)
	if err != nil {
		return 0, fmt.Errorf("parse bot_id: %s: %v", parts[0], err)
	}

	return int32(botID), nil
}
