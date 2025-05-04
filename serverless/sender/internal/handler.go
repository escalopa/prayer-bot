package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type (
	DB interface {
		CreateChat(ctx context.Context, botID int32, chatID int64, languageCode string, notifyOffset int32, state string) error
		GetChat(ctx context.Context, botID int32, chatID int64) (chat *domain.Chat, _ error)

		SetLanguageCode(ctx context.Context, botID int32, chatID int64, languageCode string) error
		SetSubscribed(ctx context.Context, botID int32, chatID int64, subscribed bool) error
		SetNotifyOffset(ctx context.Context, botID int32, chatID int64, notifyOffset int32) error
		SetNotifyMessageID(ctx context.Context, botID int32, chatID int64, notifyMessageID int32) error
		SetState(ctx context.Context, botID int32, chatID int64, state string) error

		GetPrayerDay(ctx context.Context, botID int32, date time.Time) (*domain.PrayerDay, error)

		GetStats(ctx context.Context, botID int32) (*domain.Stats, error)
	}

	Handler struct {
		bots   map[int32]*bot.Bot
		botsMu sync.Mutex

		config map[int32]*domain.BotConfig
		db     DB
	}
)

func NewHandler(config map[int32]*domain.BotConfig, db DB) *Handler {
	return &Handler{
		bots:   make(map[int32]*bot.Bot),
		config: config,
		db:     db,
	}
}

func (h *Handler) getBot(botID int32) (*bot.Bot, error) {
	h.botsMu.Lock()
	defer h.botsMu.Unlock()

	b, ok := h.bots[botID]
	if ok {
		return b, nil
	}

	botConfig, ok := h.config[botID]
	if !ok {
		return nil, fmt.Errorf("bot config not found")
	}

	b, err := bot.New(botConfig.Token, h.opts()...)
	if err != nil {
		return nil, fmt.Errorf("create bot: %v", err)
	}

	h.bots[botID] = b
	return b, nil
}

func (h *Handler) Do(ctx context.Context, body string) error {
	payload := &domain.Payload{}
	if err := payload.Unmarshal([]byte(body)); err != nil {
		return fmt.Errorf("unmarshal payload: %v", err)
	}

	switch payload.Type {
	case domain.PayloadTypeHandler:
		return h.handel(ctx, payload.Data)
	case domain.PayloadTypeNotifier:
		return h.notify(ctx, payload.Data)
	default:
		return fmt.Errorf("unknown payload type: %s", payload.Type)
	}
}

func (h *Handler) handel(ctx context.Context, data interface{}) error {
	payload, err := domain.Unmarshal[domain.HandlerPayload](data)
	if err != nil {
		return err
	}

	b, err := h.getBot(payload.BotID)
	if err != nil {
		return fmt.Errorf("get bot: %v", err)
	}

	var update models.Update
	err = json.Unmarshal([]byte(payload.Data), &update)
	if err != nil {
		return fmt.Errorf("unmarshal update: %v", err)
	}

	ctx = setContextBotID(ctx, payload.BotID)
	b.ProcessUpdate(ctx, &update)
	return nil
}

func (h *Handler) notify(ctx context.Context, data interface{}) error {
	payload, err := domain.Unmarshal[domain.NotifierPayload](data)
	if err != nil {
		return err
	}

	b, err := h.getBot(payload.BotID)
	if err != nil {
		return fmt.Errorf("get bot: %v", err)
	}

	return h.notifyBot(ctx, b, payload)
}

func (h *Handler) opts() []bot.Option {
	return []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
	}
}
