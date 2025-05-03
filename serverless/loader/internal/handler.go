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
	filenameSuffix = ".csv"
	filenameParts  = 2
)

type (
	Storage interface {
		Get(ctx context.Context, bucket string, key string) ([]byte, error)
		LoadBotConfig(ctx context.Context) (map[uint8]*domain.BotConfig, error)
	}

	DB interface {
		StorePrayers(ctx context.Context, botID uint8, rows []*domain.PrayerTimes) error
	}

	Handler struct {
		config  map[uint8]*domain.BotConfig
		storage Storage
		db      DB
	}
)

func NewHandler(config map[uint8]*domain.BotConfig, storage Storage, db DB) *Handler {
	return &Handler{
		config:  config,
		storage: storage,
		db:      db,
	}
}

func (h Handler) Process(ctx context.Context, bucket string, key string) error {
	// ignore non csv files
	if !strings.HasSuffix(key, filenameSuffix) {
		fmt.Printf("ignore file: %s\n", key)
		return nil
	}

	fmt.Printf("processing file: %s\n", key)

	botID, err := extractBotID(key)
	if err != nil {
		return fmt.Errorf("extract info from filename: %s => %v", key, err)
	}

	_, ok := h.config[botID]
	if !ok {
		return fmt.Errorf("bot config not found for bot_id: %d", botID)
	}

	data, err := h.storage.Get(ctx, bucket, key)
	if err != nil {
		return fmt.Errorf("get file from S3: %s => %v", key, err)
	}

	rows, err := parsePrayers(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("load schedule: %s => %v", key, err)
	}

	err = h.db.StorePrayers(ctx, botID, rows)
	if err != nil {
		return fmt.Errorf("store prayers: %s => %v", key, err)
	}

	fmt.Printf("processed file for bot_id: %d\n", botID)
	return nil
}

// extractBotID extracts the bot ID from the filename.
// example filename: "BOT_ID-CITY_NAME.csv"
func extractBotID(filename string) (uint8, error) {
	parts := strings.Split(path.Base(filename), "-")
	if len(parts) != filenameParts {
		return 0, fmt.Errorf("unexpected filename format: %s", filename)
	}

	botID, err := strconv.ParseUint(parts[0], 10, 8)
	if err != nil {
		return 0, fmt.Errorf("parse bot_id: %s => %v", parts[0], err)
	}

	return uint8(botID), nil
}
