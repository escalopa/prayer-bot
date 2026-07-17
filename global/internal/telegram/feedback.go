package telegram

import (
	"context"
	"fmt"
	"strings"

	botapi "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/escalopa/prayer-bot/global/internal/i18n"
)

type feedbackSender interface {
	SendMessage(context.Context, *botapi.SendMessageParams) (*models.Message, error)
	CopyMessage(context.Context, *botapi.CopyMessageParams) (*models.MessageID, error)
}

func (h *Handler) requestFeedback(ctx context.Context, chat models.Chat, locale i18n.Locale) error {
	if chat.Type != models.ChatTypePrivate {
		return h.send(ctx, chat.ID, locale.Message("feedback_private"), mainKeyboard(locale))
	}
	return h.send(ctx, chat.ID, locale.Message("feedback_prompt"), &models.ForceReply{
		ForceReply: true, InputFieldPlaceholder: locale.Message("feedback_placeholder"), Selective: true,
	})
}

func (h *Handler) submitFeedback(ctx context.Context, message *models.Message, locale i18n.Locale) error {
	if message.Chat.Type != models.ChatTypePrivate || message.From == nil {
		return nil
	}
	if err := deliverFeedback(ctx, h.bot, h.ownerID, message, locale.Code); err != nil {
		return err
	}
	return h.send(ctx, message.Chat.ID, locale.Message("feedback_sent"), mainKeyboard(locale))
}

func deliverFeedback(ctx context.Context, sender feedbackSender, ownerID int64, message *models.Message, languageCode string) error {
	name := strings.TrimSpace(message.From.FirstName + " " + message.From.LastName)
	if name == "" {
		name = "Telegram user"
	}
	username := "not set"
	if message.From.Username != "" {
		username = "@" + message.From.Username
	}
	header := fmt.Sprintf(
		"<b>New bot feedback</b> 💬\nFrom: <a href=\"tg://user?id=%d\">%s</a>\nUsername: %s\nUser ID: <code>%d</code>\nBot language: <code>%s</code>\n\n<i>Use the button below to contact this user directly. Replying inside the bot chat will not forward your message.</i>",
		message.From.ID, escape(name), escape(username), message.From.ID, escape(languageCode),
	)
	notification, err := sender.SendMessage(ctx, &botapi.SendMessageParams{
		ChatID: ownerID, Text: header, ParseMode: models.ParseModeHTML,
		ReplyMarkup: inlineKeyboard([]models.InlineKeyboardButton{{
			Text: "✉️ Contact user", URL: fmt.Sprintf("tg://user?id=%d", message.From.ID),
		}}),
	})
	if err != nil {
		return fmt.Errorf("send feedback metadata: %w", err)
	}
	if _, err := sender.CopyMessage(ctx, &botapi.CopyMessageParams{
		ChatID: ownerID, FromChatID: message.Chat.ID, MessageID: message.ID,
		ReplyParameters: &models.ReplyParameters{MessageID: notification.ID},
	}); err != nil {
		return fmt.Errorf("copy feedback content: %w", err)
	}
	return nil
}

func isFeedbackReply(message *models.Message) bool {
	reply := message.ReplyToMessage
	return message.Chat.Type == models.ChatTypePrivate && reply != nil && reply.From != nil && reply.From.IsBot && isFeedbackPrompt(reply.Text)
}

func isFeedbackPrompt(text string) bool {
	for _, locale := range i18n.Supported() {
		if text == locale.Message("feedback_prompt") {
			return true
		}
	}
	return false
}
