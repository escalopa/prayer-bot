package service

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/escalopa/prayer-bot/domain"
)

type Queue struct {
	client *sqs.SQS
}

func NewQueue() (*Queue, error) {
	config := &aws.Config{
		Endpoint:    aws.String(cfg.sqs.endpoint),
		Region:      aws.String(cfg.sqs.region),
		Credentials: credentials.NewStaticCredentials(cfg.accessKey, cfg.secretKey, ""),
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("create session: %v", err)
	}

	client := sqs.New(sess)

	return &Queue{client: client}, nil
}

func (q *Queue) Push(ctx context.Context, payload *domain.HandlePayload) error {
	b, err := payload.Marshal()
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(cfg.sqs.url),
		MessageBody: aws.String(string(b)),
	}

	_, err = q.client.SendMessageWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("send message: %v", err)
	}

	return nil
}
