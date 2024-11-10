package handler

import (
	"context"
	objs "github.com/SakoDroid/telego/objects"
	log "github.com/catalystgo/logger/cli"
	"strconv"
)

// Respond to a user's feedback or bug report
func (h *Handler) Respond(u *objs.Update) {
	var (
		chatID    = getChatID(u)
		chatIDStr = strconv.Itoa(chatID)
	)

	ch, closer, err := h.registerChannel(chatIDStr)
	if err != nil {
		log.Errorf("Handler.Respond: register channel [%d] => %v", chatID, err)
		return
	}
	defer closer()

	// Check if reply message is provided
	if u.Message.ReplyToMessage == nil {
		h.simpleSend(chatID, respondNoReplyMsg)
		return
	}

	// Read userChatID, messageID, username from the old message that will be replied to
	userChatID, responseMessageID, _, ok := parseUserMessage(u.Message.ReplyToMessage.Text)
	if !ok {
		h.simpleSend(chatID, respondInvalidMsg)
		return
	}

	// Read response message
	messageID := h.simpleSend(chatID, respondStart)
	defer h.deleteMessage(chatID, messageID)

	// Create new command context
	ctx, cancel := context.WithTimeout(h.getChatCtx(chatID), inputTimeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return
	case u = <-ch:
	}

	if h.isCancelOperation(chatID, u.Message.Text) {
		return
	}

	h.reply(userChatID, u.Message.Text, responseMessageID)
	h.simpleSend(chatID, respondSuccess)
}

// GetSubscribers returns the number of subscribers to the bot
func (h *Handler) GetSubscribers(u *objs.Update) {
	chatID := getChatID(u)

	ids, err := h.uc.GetSubscribers(h.getChatCtx(chatID))
	if err != nil {
		log.Errorf("Handler.GetSubscribers: [%d] => %v", chatID, err)
		h.simpleSend(chatID, getSubscribersErr)
		return
	}

	h.simpleSend(chatID, strconv.Itoa(len(ids)))
}

// SendAll broadcasts a message to all subscribers
func (h *Handler) SendAll(u *objs.Update) {
	var (
		chatID    = getChatID(u)
		chatIDStr = strconv.Itoa(chatID)
	)

	ch, closer, err := h.registerChannel(chatIDStr)
	if err != nil {
		log.Errorf("Handler.SendAll: register channel [%d] => %v", chatID, err)
		return
	}
	defer closer()

	// Wait for message or timeout after 2 minutes
	messageID := h.simpleSend(chatID, sendAllStart)
	defer h.deleteMessage(chatID, messageID)

	ctx1, cancel1 := context.WithTimeout(h.getChatCtx(chatID), inputTimeout)
	defer cancel1()

	select {
	case <-ctx1.Done():
		return
	case u = <-ch:
	}

	var (
		broadcastMessageID   = u.Message.MessageId
		broadcastMessageText = u.Message.Text
	)

	if h.isCancelOperation(chatID, broadcastMessageText) {
		return
	}

	// Double check that the owner still wants to send the message
	messageID = h.simpleSend(chatID, sendAllConfirm)
	defer h.deleteMessage(chatID, messageID)

	// Wait for confirmation message or timeout after 5 minutes
	ctx2, cancel2 := context.WithTimeout(h.getChatCtx(chatID), inputTimeout)
	defer cancel2()

	select {
	case <-ctx2.Done():
		return
	case u = <-ch:
	}

	// Delete the bot message if the user sends the message or if the context times out
	defer h.deleteMessage(chatID, u.Message.MessageId)

	if h.isCancelOperation(chatID, u.Message.Text) {
		return
	}

	chatIDs, err := h.uc.GetSubscribers(h.getChatCtx(chatID))
	if err != nil {
		log.Errorf("Handler.SendAll: [%d] => %v", chatID, err)
		h.simpleSend(chatID, sendAllErr)
		return
	}

	// Send message to all subscribers asynchronously
	go func() {
		for _, userChatID := range chatIDs {
			if userChatID == chatID {
				continue
			}
			h.simpleSend(userChatID, broadcastMessageText)
		}
		h.reply(chatID, sendAllSuccess, broadcastMessageID)
	}()
}
