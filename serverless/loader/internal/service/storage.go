package service

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
)

type Storage struct {
	client *storage.Client
}

func NewStorage() (*Storage, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create gcs client: %w", err)
	}
	return &Storage{client: client}, nil
}

func (s *Storage) Get(ctx context.Context, bucket string, key string) ([]byte, error) {
	reader, err := s.client.Bucket(bucket).Object(key).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("get gcs object: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read gcs object body: %w", err)
	}
	return data, nil
}
