package redis

import (
	"context"

	"github.com/go-redis/redis/v9"
)

func New(url string) (*redis.Client, error) {
	ops, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(ops)
	if err = client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return client, nil
}
