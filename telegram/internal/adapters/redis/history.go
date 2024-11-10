package redis

import (
	"context"
	"fmt"
	"strconv"

	"github.com/escalopa/gopray/telegram/internal/domain"
	"github.com/go-redis/redis/v9"
	"github.com/pkg/errors"
)

type HistoryRepository struct {
	client *redis.Client
	prefix string
}

func NewHistoryRepository(client *redis.Client, prefix string) *HistoryRepository {
	return &HistoryRepository{
		client: client,
		prefix: prefix,
	}
}

func (h *HistoryRepository) StorePrayerMessageID(ctx context.Context, chatID int, messageID int) error {
	err := h.client.Set(ctx, h.formatPrayerKey(chatID), messageID, 0).Err()
	if err != nil {
		return errors.Errorf("StorePrayerMessageID: %v", err)
	}
	return nil
}

func (h *HistoryRepository) GetPrayerMessageID(ctx context.Context, chatID int) (int, error) {
	result, err := h.client.Get(ctx, h.formatPrayerKey(chatID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, domain.ErrNotFound
		}
		return 0, errors.Errorf("GetPrayerMessageID: %v", err)
	}
	id, err := strconv.Atoi(result)
	if err != nil {
		return 0, errors.Errorf("GetPrayerMessageID: %v", err)
	}
	return id, nil
}

func (h *HistoryRepository) formatPrayerKey(chatID int) string {
	return fmt.Sprintf("%s:prayer_message_id:%d", h.prefix, chatID)
}
