package internal

import (
	"context"
	"fmt"

	"github.com/escalopa/prayer-bot/domain"
)

const (
	telegramBotAPISecretTokenHeader = "X-Telegram-Bot-Api-Secret-Token"
)

type (
	Queue interface {
		Enqueue(ctx context.Context, payload *domain.Payload) error
	}

	Handler struct {
		cfg map[int32]*domain.BotConfig
		q   Queue
	}
)

func NewHandler(cfg map[int32]*domain.BotConfig, queue Queue) *Handler {
	return &Handler{
		cfg: cfg,
		q:   queue,
	}
}

func (h *Handler) Do(ctx context.Context, body string, headers map[string]string) error {
	botID, err := h.authenticate(headers)
	if err != nil {
		return fmt.Errorf("authenticate: %v", err)
	}

	payload := &domain.Payload{
		Type: domain.PayloadTypeDispatcher,
		Data: &domain.DispatcherPayload{
			BotID: botID,
			Data:  body,
		},
	}

	err = h.q.Enqueue(ctx, payload)
	if err != nil {
		return fmt.Errorf("enqueue: %v", err)
	}

	return nil
}

func (h *Handler) authenticate(headers map[string]string) (int32, error) {
	secretToken := headers[telegramBotAPISecretTokenHeader]
	if secretToken == "" {
		return 0, fmt.Errorf("empty secret token header")
	}

	for _, botConfig := range h.cfg {
		if botConfig.Secret == secretToken {
			return botConfig.BotID, nil
		}
	}

	return 0, fmt.Errorf("secret token mismatch")
}
