package main

import (
	"context"
	"flag"
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

func main() {
	flag.Parse()

	ctx := context.Background()

	sourceBucket := os.Getenv("YC_BUCKET")
	if sourceBucket == "" {
		sourceBucket = fmt.Sprintf("prayer-bot-bucket-%s", os.Getenv("ENVIRONMENT"))
	}
	targetBucket := os.Getenv("GCS_BUCKET")
	if targetBucket == "" {
		fmt.Fprintln(os.Stderr, "GCS_BUCKET is required")
		os.Exit(1)
	}

	s3Client, err := newS3Client()
	if err != nil {
		fmt.Fprintf(os.Stderr, "create s3 client: %v\n", err)
		os.Exit(1)
	}

	keys, err := listYCObjects(ctx, s3Client, sourceBucket)
	if err != nil {
		fmt.Fprintf(os.Stderr, "list source objects: %v\n", err)
		os.Exit(1)
	}

	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create gcs client: %v\n", err)
		os.Exit(1)
	}
	defer gcsClient.Close()

	for _, key := range keys {
		if err := copyObject(ctx, s3Client, gcsClient, sourceBucket, targetBucket, key); err != nil {
			fmt.Fprintf(os.Stderr, "copy %s: %v\n", key, err)
			os.Exit(1)
		}
	}

	targetKeys, err := listGCSObjects(ctx, gcsClient, targetBucket)
	if err != nil {
		fmt.Fprintf(os.Stderr, "list target objects: %v\n", err)
		os.Exit(1)
	}

	if len(targetKeys) != len(keys) {
		fmt.Fprintf(os.Stderr, "object count mismatch: source=%d target=%d\n", len(keys), len(targetKeys))
		os.Exit(1)
	}

	fmt.Printf("bucket migration complete: %d objects copied to %s\n", len(keys), targetBucket)
}

func newS3Client() (*s3.Client, error) {
	endpoint := os.Getenv("S3_ENDPOINT")
	region := os.Getenv("REGION")

	cfg := aws.Config{
		Region: region,
		Credentials: credentials.NewStaticCredentialsProvider(
			os.Getenv("AWS_ACCESS_KEY_ID"),
			os.Getenv("AWS_SECRET_ACCESS_KEY"),
			"",
		),
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	}), nil
}

func listYCObjects(ctx context.Context, client *s3.Client, bucket string) ([]string, error) {
	var keys []string
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, obj := range page.Contents {
			if obj.Key != nil && !strings.HasSuffix(*obj.Key, "/") {
				keys = append(keys, *obj.Key)
			}
		}
	}
	return keys, nil
}

func listGCSObjects(ctx context.Context, client *storage.Client, bucket string) ([]string, error) {
	var keys []string
	it := client.Bucket(bucket).Objects(ctx, nil)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		keys = append(keys, attrs.Name)
	}
	return keys, nil
}

func copyObject(
	ctx context.Context,
	s3Client *s3.Client,
	gcsClient *storage.Client,
	sourceBucket, targetBucket, key string,
) error {
	out, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(sourceBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return err
	}

	writer := gcsClient.Bucket(targetBucket).Object(key).NewWriter(ctx)
	if out.ContentType != nil {
		writer.ContentType = *out.ContentType
	}
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}
