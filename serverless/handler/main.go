package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/escalopa/prayer-bot/handler/internal"
	"github.com/escalopa/prayer-bot/service"
)

type (
	Request struct {
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
	}
	Response struct {
		StatusCode int    `json:"status_code"`
		Body       string `json:"body"`
	}
)

func newResponse(statusCode int, body string, data ...any) (*Response, error) {
	return &Response{
		StatusCode: statusCode,
		Body:       fmt.Sprintf(body, data...),
	}, nil
}

func Handler(ctx context.Context, requestBytes []byte) (*Response, error) {
	request := &Request{}

	if err := json.Unmarshal(requestBytes, &request); err != nil {
		fmt.Printf("unmarshal: %v", err)
		return newResponse(http.StatusBadRequest, "unmarshal request body")
	}

	storage, err := service.NewStorage()
	if err != nil {
		fmt.Printf("create storage: %v", err)
		return newResponse(http.StatusInternalServerError, "create storage")
	}

	queue, err := service.NewQueue()
	if err != nil {
		fmt.Printf("create queue: %v", err)
		return newResponse(http.StatusInternalServerError, "create queue")
	}

	botConfig, err := storage.LoadBotConfig(ctx)
	if err != nil {
		fmt.Printf("load botConfig: %v", err)
		return newResponse(http.StatusInternalServerError, "load botConfig")
	}

	botID, err := internal.Authenticate(botConfig, request.Headers)
	if err != nil {
		fmt.Printf("unauthorized: %v", err)
		return newResponse(http.StatusUnauthorized, "unauthorized")
	}

	payload := &domain.Payload{
		Type: domain.PayloadTypeHandler,
		Data: &domain.HandlerPayload{
			BotID: botID,
			Data:  request.Body,
		},
	}

	err = queue.Push(ctx, payload)
	if err != nil {
		fmt.Printf("push payload: %v", err)
		return newResponse(http.StatusInternalServerError, "push payload")
	}

	return newResponse(http.StatusOK, "success")
}
