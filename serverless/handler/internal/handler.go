package internal

import (
	"context"
	"fmt"
	"strings"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type (
	Queue interface {
		Push(ctx context.Context, message *domain.Payload) error
	}

	Handler struct {
		botID uint8
		queue Queue
	}
)

func NewHandler(botID uint8, queue Queue) *Handler {
	return &Handler{
		botID: botID,
		queue: queue,
	}
}

func (h *Handler) callbackHandler() bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.CallbackQuery == nil || update.CallbackQuery.Data == "" {
			return
		}

		parts := strings.Split(update.CallbackQuery.Data, domain.StageSplitter)
		stage := len(parts) - 1
		command := parts[0]

		if !domain.IsValidCommand(command) {
			fmt.Printf("unexpected command: %s\n", command)
			return
		}

		_, err := b.AnswerCallbackQuery(ctx, &(bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			ShowAlert:       false,
		}))
		if err != nil {
			fmt.Printf("AnswerCallbackQuery: [%v] => %v\n", update.CallbackQuery.Data, err)
			return
		}

		date := update.CallbackQuery.Data[len(domain.DateCommand)+1:] // +1 for the splitter
		h.do(ctx, update.CallbackQuery.Message.Message.Chat.ID, domain.Command(command), stage, date)
	}
}

func (h *Handler) defaultHandler() bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		command := domain.HelpCommand

		if update.Message != nil && domain.IsValidCommand(update.Message.Text) {
			command = domain.Command(update.Message.Text)
		}

		h.do(ctx, update.Message.Chat.ID, command, 0, update.Message)

		// TODO: remove code below (used for testing only)
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Command received",
			ReplyParameters: &models.ReplyParameters{
				MessageID: update.Message.ID,
				ChatID:    update.Message.Chat.ID,
			},
		})
		if err != nil {
			fmt.Printf("SendMessage: %s => %v\n", update.Message.Text, err)
			return
		}
	}
}

func (h *Handler) do(ctx context.Context, chatID int64, command domain.Command, stage int, data interface{}) {
	payload := &domain.Payload{
		BotID:  h.botID,
		ChatID: chatID,

		Command: command,
		Stage:   uint8(stage),
		Data:    data,
	}
	if err := h.queue.Push(ctx, payload); err != nil {
		fmt.Printf("Push: %s:%d => %v\n", command, stage, err)
		return
	}

}

func (h *Handler) Opts() []bot.Option {
	return []bot.Option{
		bot.WithCallbackQueryDataHandler("/", bot.MatchTypePrefix, h.callbackHandler()),
		bot.WithDefaultHandler(h.defaultHandler()),
	}
}
