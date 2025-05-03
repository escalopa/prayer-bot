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
		return newResponse(http.StatusBadRequest, "unmarshal request body: %v", err)
	}

	storage, err := service.NewStorage()
	if err != nil {
		return newResponse(http.StatusInternalServerError, "create storage: %v", err)
	}

	botConfig, err := storage.LoadBotConfig(ctx)
	if err != nil {
		return newResponse(http.StatusInternalServerError, "load bot config: %v", err)
	}

	botID, err := internal.Authenticate(botConfig, request.Headers)
	if err != nil {
		return newResponse(http.StatusUnauthorized, "authenticate: %v", err)
	}

	queue, err := service.NewQueue()
	if err != nil {
		return newResponse(http.StatusInternalServerError, "create queue: %v", err)
	}

	payload := &domain.Payload{BotID: botID, Data: request.Body}

	err = queue.Push(ctx, payload)
	if err != nil {
		return newResponse(http.StatusInternalServerError, fmt.Sprintf("push payload: %v", err))
	}

	return newResponse(http.StatusOK, "success")
}
