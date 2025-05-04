package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/escalopa/prayer-bot/domain"
)

type Storage struct {
	client *s3.S3
}

func NewStorage() (*Storage, error) {
	config := &aws.Config{
		Endpoint:         aws.String(cfg.s3.endpoint),
		Region:           aws.String(cfg.region),
		Credentials:      credentials.NewStaticCredentials(cfg.accessKey, cfg.secretKey, ""),
		S3ForcePathStyle: aws.Bool(true), // required for non-AWS S3 implementations
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("create session: %v", err)
	}

	return &Storage{client: s3.New(sess)}, nil
}

func (s *Storage) Get(ctx context.Context, bucket string, key string) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.GetObjectWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("get object: %v", err)
	}
	defer func(Body io.ReadCloser) { _ = Body.Close() }(result.Body)

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("read object body: %v", err)
	}

	return data, nil
}

const (
	botConfigKey = "bot_config.json"
)

func (s *Storage) LoadBotConfig(ctx context.Context) (map[int32]*domain.BotConfig, error) {
	data, err := s.Get(ctx, cfg.s3.bucket, botConfigKey)
	if err != nil {
		return nil, fmt.Errorf("get botConfig: %v", err)
	}

	var config map[int32]*domain.BotConfig

	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("unmarshal bot config: %v", err)
	}

	return config, nil
}
