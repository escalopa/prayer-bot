package service

import (
	"os"
)

var cfg = struct {
	sqs struct {
		url      string
		region   string
		endpoint string
	}

	s3 struct {
		bucket   string
		endpoint string
	}

	ydb struct {
		endpoint string
	}

	region    string
	accessKey string
	secretKey string
}{}

func init() {
	cfg.sqs.url = os.Getenv("SQS_URL")
	cfg.sqs.region = os.Getenv("SQS_REGION")
	cfg.sqs.endpoint = "https://message-queue.api.cloud.yandex.net"

	cfg.s3.bucket = os.Getenv("S3_BUCKET")
	cfg.s3.endpoint = "https://storage.yandexcloud.net"

	cfg.ydb.endpoint = os.Getenv("YDB_ENDPOINT")

	cfg.region = os.Getenv("REGION")
	cfg.accessKey = os.Getenv("ACCESS_KEY")
	cfg.secretKey = os.Getenv("SECRET_KEY")
}
