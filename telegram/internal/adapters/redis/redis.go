package redis

import (
	"context"
	"log"

	"github.com/go-redis/redis/v9"
)

func New(url string) *redis.Client {
	ops, err := redis.ParseURL(url)
	if err != nil {
		log.Fatal(err)
	}
	client := redis.NewClient(ops)
	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Fatal(err)
	}
	return client
}
