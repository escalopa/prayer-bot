package handler

import (
	"context"
	"log"
	"strconv"
	"time"

	objs "github.com/SakoDroid/telego/objects"
)

const adminProcessTimeout = 5 * time.Minute

// Respond to a user's feedback or bug report
func (h *Handler) Respond(u *objs.Update) {
	var (
		chatID    = getChatID(u)
		chatIDStr = strconv.Itoa(chatID)
	)

	ch, err := h.bot.AdvancedMode().RegisterChannel(chatIDStr, "message")
	defer h.bot.AdvancedMode().UnRegisterChannel(chatIDStr, "message")

	if err != nil {
		log.Printf("failed to register channel for /respond: %s", err)
		return
	}

	// Check if reply message is provided
	if u.Message.ReplyToMessage == nil {
		h.simpleSend(chatID, respondNoReplyMsg, 0)
		return
	}

	// Read userChatID, messageID, username from the old message that will be replied to
	userChatID, responseMessageID, _, ok := parseUserMessage(u.Message.ReplyToMessage.Text)
	if !ok {
		h.simpleSend(chatID, respondInvalidMsg, 0)
		return
	}

	// Read response message
	messageID := h.simpleSend(chatID, respondStart, 0)
	defer h.deleteMessage(chatID, messageID)

	// Create new command context
	ctx, cancel := context.WithTimeout(h.getChatCtx(chatID), adminProcessTimeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return
	case u = <-*ch:
	}

	if h.isCancelOperation(chatID, u.Message.Text) {
		return
	}

	// Send response message to user
	_, err = h.bot.SendMessage(userChatID, u.Message.Text, "", responseMessageID, false, false)
	if err != nil {
		h.simpleSend(chatID, respondErr, 0)
		log.Printf("failed to respond on user's message on : %s", err)
		return
	}

	h.simpleSend(chatID, respondSuccess, 0)
}

// GetSubscribers returns the number of subscribers to the bot
func (h *Handler) GetSubscribers(u *objs.Update) {
	chatID := getChatID(u)

	ids, err := h.uc.GetSubscribers(h.getChatCtx(chatID))
	if err != nil {
		h.simpleSend(chatID, getSubscribersErr, 0)
		log.Printf("failed to get subscribers on /subs : %s", err)
		return
	}

	h.simpleSend(chatID, strconv.Itoa(len(ids)), 0)
}

// SendAll broadcasts a message to all subscribers
func (h *Handler) SendAll(u *objs.Update) {
	var (
		chatID    = getChatID(u)
		chatIDStr = strconv.Itoa(chatID)
	)

	ch, err := h.bot.AdvancedMode().RegisterChannel(chatIDStr, "message")
	defer h.bot.AdvancedMode().UnRegisterChannel(chatIDStr, "message")
	if err != nil {
		log.Printf("failed to register channel for /sendall: %s", err)
		return
	}

	// Wait for message or timeout after 2 minutes
	messageID := h.simpleSend(chatID, sendAllStart, 0)
	defer h.deleteMessage(chatID, messageID)

	ctx1, cancel1 := context.WithTimeout(h.getChatCtx(chatID), adminProcessTimeout)
	defer cancel1()

	select {
	case <-ctx1.Done():
		return
	case u = <-*ch:
	}

	var (
		broadcastMessageID   = u.Message.MessageId
		broadcastMessageText = u.Message.Text
	)

	if h.isCancelOperation(chatID, broadcastMessageText) {
		return
	}

	// Double check that the owner still wants to send the message
	messageID = h.simpleSend(chatID, sendAllConfirm, 0)
	defer h.deleteMessage(chatID, messageID)

	// Wait for confirmation message or timeout after 5 minutes
	ctx2, cancel2 := context.WithTimeout(h.getChatCtx(chatID), adminProcessTimeout)
	defer cancel2()

	select {
	case <-ctx2.Done():
		return
	case u = <-*ch:
	}

	// Delete the bot message if the user sends the message or if the context times out
	defer h.deleteMessage(chatID, u.Message.MessageId)

	if h.isCancelOperation(chatID, u.Message.Text) {
		return
	}

	chatIDs, err := h.uc.GetSubscribers(h.getChatCtx(chatID))
	if err != nil {
		h.simpleSend(chatID, sendAllErr, 0)
		log.Printf("failed to send message subscribers on /sendall : %s", err)
		return
	}

	// Send message to all subscribers asynchronously
	go func() {
		for _, userChatID := range chatIDs {
			h.simpleSend(userChatID, broadcastMessageText, 0)
		}
		h.simpleSend(chatID, sendAllSuccess, broadcastMessageID)
	}()
}
