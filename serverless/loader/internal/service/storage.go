package service

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Storage struct {
	client *s3.S3
}

func NewStorage() (*Storage, error) {
	var (
		s3Endpoint = os.Getenv("S3_ENDPOINT")

		region    = os.Getenv("REGION")
		accessKey = os.Getenv("ACCESS_KEY")
		secretKey = os.Getenv("SECRET_KEY")
	)

	config := &aws.Config{
		Endpoint:         aws.String(s3Endpoint),
		Region:           aws.String(region),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
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
