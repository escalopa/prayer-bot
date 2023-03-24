package handler

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	objs "github.com/SakoDroid/telego/objects"
)

// Respond to a user's feedback or bug report
func (h *Handler) Respond(u *objs.Update) {
	chatID := strconv.Itoa(u.Message.Chat.Id)
	ch, err := h.b.AdvancedMode().RegisterChannel(chatID, "message")
	defer h.b.AdvancedMode().UnRegisterChannel(chatID, "message")
	if err != nil {
		log.Printf("failed to register channel for /respond: %s", err)
		return
	}

	// Check if reply message is provided
	if u.Message.ReplyToMessage == nil {
		h.simpleSend(u.Message.Chat.Id, "No reply message provided, /respond", 0)
		return
	}

	// Read userID, messageID, username from the old message that will be replied to
	userID, responseMessageID, _, ok := parseUserMessage(u.Message.ReplyToMessage.Text)
	if !ok {
		h.simpleSend(u.Message.Chat.Id, "Invalid message.", 0)
		return
	}

	// Read response message
	messageID := h.simpleSend(u.Message.Chat.Id, "Send your response message, Or /cancel", 0)
	defer h.deleteMessage(u.Message.Chat.Id, messageID)

	// Create new command context
	ctx, cancel := context.WithTimeout(h.userCtx[u.Message.Chat.Id].ctx, 1*time.Minute)
	defer cancel()

	select {
	case <-ctx.Done():
		return
	case u = <-*ch:
	}

	if h.cancelOperation(u.Message.Text, "Canceled response.", u.Message.Chat.Id) {
		return
	}

	// Send response message to user
	_, err = h.b.SendMessage(userID, u.Message.Text, "", responseMessageID, false, false)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "Failed to send response.", 0)
		log.Printf("failed to respond on user's message on : %s", err)
		return
	}

	h.simpleSend(u.Message.Chat.Id, "Response sent successfully.", 0)
}

// GetSubscribers returns the number of subscribers to the bot
func (h *Handler) GetSubscribers(u *objs.Update) {
	ids, err := h.u.GetSubscribers(h.c)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "Failed to get subscribers.", 0)
		log.Printf("failed to get subscribers on /subs : %s", err)
		return
	}
	h.simpleSend(u.Message.Chat.Id, fmt.Sprintf("Subscribers: %d", len(ids)), 0)
}

// SendAll broadcasts a message to all subscribers
func (h *Handler) SendAll(u *objs.Update) {
	// Register channel to receive messages
	chatID := u.Message.Chat.Id
	ch, err := h.b.AdvancedMode().RegisterChannel(strconv.Itoa(u.Message.Chat.Id), "message")
	defer h.b.AdvancedMode().UnRegisterChannel(strconv.Itoa(u.Message.Chat.Id), "message")
	if err != nil {
		log.Printf("failed to register channel for /sendall: %s", err)
		return
	}

	// Wait for message or timeout after 2 minutes
	messageID := h.simpleSend(chatID, "Send your message, Or /cancel", 0)
	defer h.deleteMessage(chatID, messageID)

	ctx1, cancel1 := context.WithTimeout(h.userCtx[chatID].ctx, 1*time.Minute)
	defer cancel1()

	select {
	case u = <-*ch:
	case <-ctx1.Done():
		return
	}
	broadcastMessage := u.Message.Text

	if h.cancelOperation(u.Message.Text, "Canceled broadcast.", chatID) {
		return
	}

	// Double check that the owner still wants to send the message
	messageID = h.simpleSend(chatID, "Use /confirm to send the message, Or /cancel", 0)
	defer h.deleteMessage(chatID, messageID)

	// Wait for confirmation message or timeout after 5 minutes
	ctx2, cancel2 := context.WithTimeout(h.userCtx[chatID].ctx, 1*time.Minute)
	defer cancel2()

	select {
	case <-ctx2.Done():
		return
	case u = <-*ch:
	}

	// Delete the bot message if the user sends the message or if the context times out
	defer h.deleteMessage(chatID, u.Message.MessageId)

	if u.Message.Text != "/confirm" || h.cancelOperation(u.Message.Text, "Canceled broadcast.", chatID) {
		return
	}
	// Send message to all subscribers in a goroutine
	go func() {
		// Get all subscribers
		ids, err := h.u.GetSubscribers(h.c)
		if err != nil {
			h.simpleSend(chatID, "Failed to send message.", 0)
			log.Printf("failed to send message subscribers on /sendall : %s", err)
			return
		}
		// Send message to all subscribers
		for _, id := range ids {
			h.simpleSend(id, broadcastMessage, 0)
		}
		h.simpleSend(chatID, "Message sent successfully.", 0)
	}()
}
