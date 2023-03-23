package redis

import (
	"context"
	"strconv"

	"github.com/pkg/errors"

	"github.com/go-redis/redis/v9"
)

const sk = "gopraySubscribers" // subscribers list key

type SubscriberRepository struct {
	r *redis.Client
}

func NewSubscriberRepository(r *redis.Client) *SubscriberRepository {
	return &SubscriberRepository{r: r}
}

func (s *SubscriberRepository) StoreSubscriber(ctx context.Context, id int) error {
	err := s.r.SAdd(ctx, sk, id).Err()
	if err != nil {
		return errors.Wrap(err, "failed to store subscriber in redis")
	}
	return nil
}

func (s *SubscriberRepository) RemoveSubscribe(ctx context.Context, id int) error {
	err := s.r.SRem(ctx, sk, id).Err()
	if err != nil {
		return errors.Wrap(err, "failed to remove subscriber from redis")
	}
	return nil
}

func (s *SubscriberRepository) GetSubscribers(ctx context.Context) ([]int, error) {
	sIds, err := s.r.SMembers(ctx, sk).Result()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get subscribers from redis")
	}
	// Convert string ids to int
	ids := make([]int, len(sIds))
	for i, sId := range sIds {
		ids[i], err = strconv.Atoi(sId)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert subscriber id to int")
		}
	}
	return ids, nil
}
