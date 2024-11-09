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

func (hr *HistoryRepository) StorePrayerMessageID(_ context.Context, userID int, messageID int) error {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	hr.prayers[userID] = messageID

	return nil
}

func (hr *HistoryRepository) GetPrayerMessageID(_ context.Context, userID int) (int, error) {
	hr.mu.RLock()
	defer hr.mu.RUnlock()

	messageID, ok := hr.prayers[userID]
	if !ok || messageID == 0 {
		return 0, domain.ErrNotFound
	}

	return messageID, nil
}
