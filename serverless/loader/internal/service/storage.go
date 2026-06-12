package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"google.golang.org/api/iterator"
)

type Storage struct {
	backend string
	s3      *s3.Client
	gcs     *storage.Client
}

func NewStorage() (*Storage, error) {
	backend := strings.ToLower(strings.TrimSpace(os.Getenv("STORAGE_BACKEND")))
	if backend == "" {
		if os.Getenv("GCS_BUCKET") != "" || os.Getenv("STORAGE_PROVIDER") == "gcs" {
			backend = "gcs"
		} else {
			backend = "s3"
		}
	}

	switch backend {
	case "gcs":
		ctx := context.Background()
		client, err := storage.NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("create gcs client: %w", err)
		}
		return &Storage{backend: "gcs", gcs: client}, nil
	default:
		return newS3Storage()
	}
}

func newS3Storage() (*Storage, error) {
	endpoint := os.Getenv("S3_ENDPOINT")
	region := os.Getenv("REGION")
	accessKey := os.Getenv("ACCESS_KEY")
	secretKey := os.Getenv("SECRET_KEY")

	cfg := aws.Config{
		Region: region,
		Credentials: credentials.NewStaticCredentialsProvider(
			accessKey,
			secretKey,
			"",
		),
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	return &Storage{backend: "s3", s3: client}, nil
}

func (s *Storage) Get(ctx context.Context, bucket string, key string) ([]byte, error) {
	switch s.backend {
	case "gcs":
		reader, err := s.gcs.Bucket(bucket).Object(key).NewReader(ctx)
		if err != nil {
			return nil, fmt.Errorf("get gcs object: %w", err)
		}
		defer reader.Close()

		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("read gcs object body: %w", err)
		}
		return data, nil
	default:
		result, err := s.s3.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
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
}

// ListObjects lists object names in a bucket (used by bucket migration script).
func (s *Storage) ListObjects(ctx context.Context, bucket string) ([]string, error) {
	switch s.backend {
	case "gcs":
		var keys []string
		it := s.gcs.Bucket(bucket).Objects(ctx, nil)
		for {
			attrs, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("list gcs objects: %w", err)
			}
			keys = append(keys, attrs.Name)
		}
		return keys, nil
	default:
		var keys []string
		paginator := s3.NewListObjectsV2Paginator(s.s3, &s3.ListObjectsV2Input{
			Bucket: aws.String(bucket),
		})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("list s3 objects: %w", err)
			}
			for _, obj := range page.Contents {
				if obj.Key != nil {
					keys = append(keys, *obj.Key)
				}
			}
		}
		return keys, nil
	}
}
