package service

import (
	"os"
)

const (
	s3Endpoint  = "https://storage.yandexcloud.net"
	sqsEndpoint = "https://message-queue.yandexcloud.net"
)

var cfg = struct {
	region    string
	accessKey string
	secretKey string

	ydb    string
	queue  string
	bucket string
}{}

func init() {
	cfg.ydb = os.Getenv("YDB_ENDPOINT")
	cfg.queue = os.Getenv("QUEUE_URL")
	cfg.bucket = os.Getenv("S3_BUCKET")

	cfg.region = os.Getenv("REGION")
	cfg.accessKey = os.Getenv("ACCESS_KEY")
	cfg.secretKey = os.Getenv("SECRET_KEY")
}
