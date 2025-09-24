package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/escalopa/prayer-bot/log"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"golang.org/x/sync/errgroup"
)

type replyType string

const (
	replyTypeBug      replyType = "bug"
	replyTypeFeedback replyType = "feedback"
)

type replyInfo struct {
	Type      replyType `json:"type"`
	ChatID    int64     `json:"chat_id"`
	MessageID int       `json:"message_id"`
	Username  string    `json:"username"`
}

func newReplyInfo(replyType replyType, chatID int64, messageID int, username string) *replyInfo {
	return &replyInfo{
		Type:      replyType,
		ChatID:    chatID,
		MessageID: messageID,
		Username:  username,
	}
}

func (r *replyInfo) JSON() string {
	bytes, _ := json.MarshalIndent(r, "", "\t")
	return string(bytes)
}

type state string

const (
	defaultState state = "default"

	// user state

	bugState      state = "bug"
	feedbackState state = "feedback"

	// admin state

	replyState    state = "reply"
	announceState state = "announce"
)

func (c state) String() string {
	return string(c)
}

func (h *Handler) bugState(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat := getContextChat(ctx)

	_, err := b.ForwardMessage(ctx, &bot.ForwardMessageParams{
		ChatID:     h.cfg[chat.BotID].OwnerID,
		FromChatID: chat.ChatID,
		MessageID:  update.Message.ID,
	})
	if err != nil {
		log.Error("bugState: forward message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	info := newReplyInfo(replyTypeBug, chat.ChatID, update.Message.ID, update.Message.From.Username)
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: h.cfg[chat.BotID].OwnerID,
		Text:   info.JSON(),
	})
	if err != nil {
		log.Error("bugState: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Bug.Success,
	})
	if err != nil {
		log.Error("bugState: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	return nil
}

func (h *Handler) feedbackState(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat := getContextChat(ctx)

	_, err := b.ForwardMessage(ctx, &bot.ForwardMessageParams{
		ChatID:     h.cfg[chat.BotID].OwnerID,
		FromChatID: chat.ChatID,
		MessageID:  update.Message.ID,
	})
	if err != nil {
		log.Error("feedbackState: forward message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	info := newReplyInfo(replyTypeFeedback, chat.ChatID, update.Message.ID, update.Message.From.Username)
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: h.cfg[chat.BotID].OwnerID,
		Text:   info.JSON(),
	})
	if err != nil {
		log.Error("feedbackState: send message to owner", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Feedback.Success,
	})
	if err != nil {
		log.Error("feedbackState: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	return nil
}

func (h *Handler) replyState(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat := getContextChat(ctx)

	if update.Message.ReplyToMessage == nil {
		return fmt.Errorf("replyState: reply to message is nil")
	}

	info := &replyInfo{}
	err := json.Unmarshal([]byte(update.Message.ReplyToMessage.Text), info)
	if err != nil {
		log.Error("replyState: unmarshal reply info", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: info.ChatID,
		Text:   update.Message.Text,
		ReplyParameters: &models.ReplyParameters{
			MessageID:                info.MessageID,
			AllowSendingWithoutReply: true,
		},
		Entities: update.Message.Entities,
	})
	if err != nil {
		log.Error("replyState: reply message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Reply.Success,
	})
	if err != nil {
		log.Error("replyState: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	return nil
}

func (h *Handler) announceState(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat := getContextChat(ctx)

	chats, err := h.db.GetChats(ctx, chat.BotID)
	if err != nil {
		log.Error("announceState: get all chats", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	g := &errgroup.Group{}
	for _, c := range chats {
		c := c
		g.Go(func() error {
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: c.ChatID,
				Text:   update.Message.Text,
			})
			if err != nil {
				log.Error("announceState: send message to user",
					log.Err(err),
					log.BotID(c.ChatID),
					log.ChatID(c.ChatID),
				)
			}
			return nil
		})
	}

	_ = g.Wait()

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Announce.Success,
	})
	if err != nil {
		log.Error("announceState: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	return nil
}
