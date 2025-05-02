package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
)

const (
	s3Endpoint = "https://storage.yandexcloud.net"
)

type BotConfig struct {
	BotID    uint8  `json:"bot_id"`
	Location string `json:"location"`
	Token    string `json:"token"`
}

func (bc *BotConfig) String() string {
	return fmt.Sprintf("BotConfig{bot_id: %d, location: %s, token: %s}", bc.BotID, bc.Location, mask(bc.Token))
}

var cfg = struct {
	Region    string
	AccessKey string
	SecretKey string

	YDBEndpoint string
	S3Bucket    string
}{}

func init() {
	botIDsStr := strings.Split(os.Getenv("BOT_IDS"), ",")
	botIDs := make([]int, len(botIDsStr))
	for i, idStr := range botIDsStr {
		id, err := strconv.Atoi(strings.TrimSpace(idStr))
		if err != nil {
			log.Fatalf("parse bot_id %q at index %d: %v", idStr, i, err)
		}
		botIDs[i] = id
	}

	cfg.YDBEndpoint = os.Getenv("YDB_ENDPOINT")
	cfg.S3Bucket = os.Getenv("S3_BUCKET")

	cfg.Region = os.Getenv("REGION")
	cfg.AccessKey = os.Getenv("ACCESS_KEY")
	cfg.SecretKey = os.Getenv("SECRET_KEY")
}

type credentials struct{}

func (c *credentials) Retrieve(_ context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID:     cfg.AccessKey,
		SecretAccessKey: cfg.SecretKey,
	}, nil
}

func mask(token string) string {
	tokenLen := len(token)
	if tokenLen >= 8 {
		return fmt.Sprintf("%s%s%s",
			token[:4],                       // first 4 chars
			strings.Repeat("*", tokenLen-8), // middle chars
			token[tokenLen-4:])              // last 4 chars
	} else {
		return ""
	}
}
