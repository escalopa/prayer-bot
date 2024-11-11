package handler

import (
	"strconv"

	objs "github.com/SakoDroid/telego/objects"
	log "github.com/catalystgo/logger/cli"
)

// Respond to a user's feedback or bug report
func (h *Handler) Respond(u *objs.Update) {
	var (
		chatID = getChatID(u)
		ctx    = h.getChatCtx(chatID)
	)

	ch, closer, err := h.registerChannel(chatID)
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

	// Read userChatID, messageID, from the old message that will be replied to
	userChatID, userMessageID, ok := parseUserMessage(u.Message.ReplyToMessage.Text)
	if !ok {
		h.simpleSend(chatID, respondInvalidMsg)
		return
	}

	// Read response message
	messageID := h.simpleSend(chatID, respondStart)
	defer h.deleteMessage(chatID, messageID)

	u = h.readInput(ctx, ch)
	if u == nil {
		return
	}

	if h.isCancelOperation(chatID, u.Message.Text) {
		return
	}

	h.reply(userChatID, u.Message.Text, userMessageID)
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
		chatID = getChatID(u)
		ctx    = h.getChatCtx(chatID)
	)

	ch, closer, err := h.registerChannel(chatID)
	if err != nil {
		log.Errorf("Handler.SendAll: register channel [%d] => %v", chatID, err)
		return
	}
	defer closer()

	// Wait for message or timeout after 2 minutes
	messageID := h.simpleSend(chatID, sendAllStart)
	defer h.deleteMessage(chatID, messageID)

	u = h.readInput(ctx, ch)
	if u == nil {
		return
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

	u = h.readInput(ctx, ch)
	if u == nil {
		return
	}

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
			if userChatID == chatID { // Skip the owner
				continue
			}
			h.simpleSend(userChatID, broadcastMessageText)
		}
		h.reply(chatID, sendAllSuccess, broadcastMessageID) // Notify the owner
	}()
}
