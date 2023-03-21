package redis

import (
	"context"
	"strconv"

	"github.com/go-redis/redis/v9"
	"github.com/pkg/errors"
)

type HistoryRepository struct {
	r *redis.Client
}

func NewHistoryRepository(c *redis.Client) *HistoryRepository {
	return &HistoryRepository{r: c}
}

func (h *HistoryRepository) GetPrayerMessageID(ctx context.Context, userID int) (int, error) {
	result, err := h.r.Get(ctx, h.formatPrayerKey(userID)).Result()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get prayer message id from redis")
	}
	id, err := strconv.Atoi(result)
	if err != nil {
		return 0, errors.Wrap(err, "failed to convert prayer message id to int in redis")
	}
	return id, nil
}

func (h *HistoryRepository) StorePrayerMessageID(ctx context.Context, userID int, messageID int) error {
	if userID == 0 {
		return errors.New("prayer message id can't be stored with 0 user id")
	}
	err := h.r.Set(ctx, h.formatPrayerKey(userID), messageID, 0).Err()
	if err != nil {
		return errors.Wrap(err, "failed to store prayer message id in redis")
	}
	return nil
}

func (h *HistoryRepository) formatPrayerKey(userID int) string {
	return "gopray_prayer_message_id:" + strconv.Itoa(userID)
}
