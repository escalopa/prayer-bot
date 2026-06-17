package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/escalopa/prayer-bot/config"
	"net/http"

	"github.com/escalopa/prayer-bot/dispatcher/internal/handler"
	"github.com/escalopa/prayer-bot/dispatcher/internal/service"
	"github.com/escalopa/prayer-bot/log"
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
		log.Error("dispatcher.entry.unmarshalRequest: failed",
			log.Op("unmarshalRequest"), log.Err(err))
		return newResponse(http.StatusBadRequest, "unmarshal request body")
	}

	botConfig, err := config.Load()
	if err != nil {
		log.Error("dispatcher.entry.loadConfig: failed",
			log.Op("loadConfig"), log.Err(err))
		return newResponse(http.StatusInternalServerError, "load config")
	}

	db, err := service.NewDB(ctx)
	if err != nil {
		log.Error("dispatcher.entry.createDB: failed",
			log.Op("createDB"), log.Err(err))
		return newResponse(http.StatusInternalServerError, "create db")
	}

	h, err := handler.New(botConfig, db)
	if err != nil {
		log.Error("dispatcher.entry.createHandler: failed",
			log.Op("createHandler"), log.Err(err))
		return newResponse(http.StatusInternalServerError, "create handler")
	}

	botID, err := h.Authenticate(request.Headers)
	if err != nil {
		log.Error("dispatcher.entry.authenticate: failed",
			log.Op("authenticate"), log.Err(err))
		return newResponse(http.StatusUnauthorized, "authenticate")
	}

	err = h.Handel(ctx, botID, request.Body)
	if err != nil {
		log.Error("dispatcher.entry.processRequest: handler failed",
			log.Op("processRequest"), log.Err(err))
		return newResponse(http.StatusInternalServerError, "dispatcher cannot process request")
	}

	return newResponse(http.StatusOK, "success")
}
