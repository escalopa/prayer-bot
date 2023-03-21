package memory

import (
	"context"
	"errors"
	"fmt"
)

type HistoryRepository struct {
	prayers map[int]int
	gomaa   map[int]int
}

func NewHistoryRepository() *HistoryRepository {
	return &HistoryRepository{
		prayers: make(map[int]int),
		gomaa:   make(map[int]int),
	}
}

// Default prayers

func (h *HistoryRepository) GetPrayerMessageID(ctx context.Context, userID int) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	messageID, ok := h.prayers[userID]
	if !ok {
		return 0, fmt.Errorf("message id for user %d not found", userID)
	}
	return messageID, nil
}

func (h *HistoryRepository) StorePrayerMessageID(ctx context.Context, userID int, messageID int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if userID == 0 {
		return errors.New("user id cannot be 0")
	}
	h.prayers[userID] = messageID
	return nil
}

// Gomaa

func (h *HistoryRepository) GetGomaaMessageID(ctx context.Context, userID int) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	messageID, ok := h.gomaa[userID]
	if !ok {
		return 0, fmt.Errorf("message id for user %d not found", userID)
	}
	return messageID, nil
}

func (h *HistoryRepository) StoreGomaaMessageID(ctx context.Context, userID int, messageID int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if userID == 0 {
		return errors.New("user id cannot be 0")
	}
	h.gomaa[userID] = messageID
	return nil
}
