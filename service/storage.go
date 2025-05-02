package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Storage struct {
	client *s3.Client
}

func NewStorage() *Storage {
	client := s3.New(s3.Options{
		Credentials:  &credentials{},
		Region:       cfg.Region,
		BaseEndpoint: aws.String(s3Endpoint),
	})
	return &Storage{client: client}
}

func (s *Storage) Get(ctx context.Context, bucket string, key string) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}
	defer func(Body io.ReadCloser) { _ = Body.Close() }(result.Body)

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("read object body: %w", err)
	}

	return data, nil
}

const (
	botConfigKey = "bot_config.json"
)

func (s *Storage) LoadBotConfig(ctx context.Context, bucket string) (map[uint8]*BotConfig, error) {
	data, err := s.Get(ctx, bucket, botConfigKey)
	if err != nil {
		return nil, fmt.Errorf("load bot config: %w", err)
	}

	var config map[uint8]*BotConfig

	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("unmarshal bot config: %w", err)
	}

	return config, nil
}
