package redis

import (
	"context"
	"strconv"

	"github.com/go-redis/redis/v9"
)

const sk = "gopraySubscribers" // subscribers list key

type SubscriberRepository struct {
	r *redis.Client
}

func NewSubscriberRepository(r *redis.Client) *SubscriberRepository {
	return &SubscriberRepository{r: r}
}

func (s *SubscriberRepository) StoreSubscriber(id int) error {
	return s.r.SAdd(context.TODO(), sk, id).Err()
}

func (s *SubscriberRepository) RemoveSubscribe(id int) error {
	return s.r.SRem(context.TODO(), sk, id).Err()
}

func (s *SubscriberRepository) GetSubscribers() ([]int, error) {
	sIds, err := s.r.SMembers(context.TODO(), sk).Result()
	if err != nil {
		return nil, err
	}
	// Convert string ids to int
	ids := make([]int, len(sIds))
	for i, sId := range sIds {
		ids[i], err = strconv.Atoi(sId)
		if err != nil {
			return nil, err
		}
	}
	return ids, nil
}
