package memory

import (
	"context"
	"sync"

	"github.com/escalopa/gopray/telegram/internal/domain"
)

type HistoryRepository struct {
	prayers map[int]int
	mu      sync.RWMutex
}

func NewHistoryRepository() *HistoryRepository {
	return &HistoryRepository{prayers: make(map[int]int)}
}

func (h *HistoryRepository) StorePrayerMessageID(_ context.Context, userID int, messageID int) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.prayers[userID] = messageID

	return nil
}

func (h *HistoryRepository) GetPrayerMessageID(_ context.Context, userID int) (int, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	messageID, ok := h.prayers[userID]
	if !ok || messageID == 0 {
		return 0, domain.ErrNotFound
	}

	return messageID, nil
}
