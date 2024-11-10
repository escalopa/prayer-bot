package redis

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	"github.com/go-redis/redis/v9"
)

const listKey = "subscribers" // subscribers list key

type SubscriberRepository struct {
	client  *redis.Client
	listKey string
}

func NewSubscriberRepository(client *redis.Client, prefix string) *SubscriberRepository {
	return &SubscriberRepository{
		client:  client,
		listKey: fmt.Sprintf("%s:%s", prefix, listKey),
	}
}

func (s *SubscriberRepository) StoreSubscriber(ctx context.Context, id int) error {
	err := s.client.SAdd(ctx, s.listKey, id).Err()
	if err != nil {
		return errors.Errorf("StoreSubscriber: %v", err)
	}
	return nil
}

func (s *SubscriberRepository) RemoveSubscribe(ctx context.Context, id int) error {
	err := s.client.SRem(ctx, s.listKey, id).Err()
	if err != nil {
		return errors.Errorf("RemoveSubscribe: %v", err)
	}
	return nil
}

func (s *SubscriberRepository) GetSubscribers(ctx context.Context) ([]int, error) {
	setIDs, err := s.client.SMembers(ctx, s.listKey).Result()
	if err != nil {
		return nil, errors.Errorf("GetSubscribers: %v", err)
	}

	// Convert string ids to int
	ids := make([]int, len(setIDs))
	for i, id := range setIDs {
		ids[i], err = strconv.Atoi(id)
		if err != nil {
			return nil, errors.Errorf("GetSubscribers: %v", err)
		}
	}
	return ids, nil
}
