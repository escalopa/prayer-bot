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
)

type Storage struct {
	client *s3.S3
}

func NewStorage() (*Storage, error) {
	config := &aws.Config{
		Endpoint:         aws.String(s3Endpoint),
		Region:           aws.String(cfg.region),
		Credentials:      credentials.NewStaticCredentials(cfg.accessKey, cfg.secretKey, ""),
		S3ForcePathStyle: aws.Bool(true), // required for non-AWS S3 implementations
	}

	// create a new session with the custom configuration
	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	client := s3.New(sess, config)

	return &Storage{client: client}, nil
}

func (s *Storage) Get(ctx context.Context, bucket string, key string) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.GetObjectWithContext(ctx, input)
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

func (s *Storage) LoadBotConfig(ctx context.Context) (map[uint8]*BotConfig, error) {
	data, err := s.Get(ctx, cfg.bucket, botConfigKey)
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
