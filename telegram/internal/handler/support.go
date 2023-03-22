package handler

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/escalopa/gopray/pkg/core"

	objs "github.com/SakoDroid/telego/objects"
)

func (h *Handler) Start(u *objs.Update) {
	if _, err := h.u.GetLang(h.c, u.Message.Chat.Id); err != nil {
		userLang := u.Message.From.LanguageCode
		if !core.IsValidLang(userLang) {
			userLang = core.DefaultLang()
		}

		err = h.u.SetLang(h.userCtx[u.Message.Chat.Id].ctx, u.Message.Chat.Id, userLang)
		if err != nil {
			log.Printf("failed to set user language on start command, %s", err)
		}
	}
	h.Help(u)
}

func (h *Handler) Help(u *objs.Update) {
	_, err := h.b.SendMessage(u.Message.Chat.Id, `
	Asalamu alaykum, I am kazan prayer's time bot, I can help you know prayer's time anytime to always pray on time 🙏.

	Available commands are below: 👇

	<b>Prayers</b>
	/today - Get prayer's time for today ⏰
	/date - Get prayer's time for a specific date 📅
	/subscribe - Subscribe to daily prayers notification 🔔
	/unsubscribe - Unsubscribe from daily prayers notification 🔕

	<b>Support</b>
	/help - Show this message 📖
	/lang - Set bot language  🌐
	/feedback - Send feedback or idea to the bot developers 📩
	/bug - Report a bug to the bot developers 🐞
	`, "HTML", 0, false, false)
	if err != nil {
		log.Printf("Error: %s, Failed to send help message", err)
	}
}

func (h *Handler) Feedback(u *objs.Update) {
	chatID := strconv.Itoa(u.Message.Chat.Id)
	ch, err := h.b.AdvancedMode().RegisterChannel(chatID, "message")
	defer h.b.AdvancedMode().UnRegisterChannel(chatID, "message")
	if err != nil {
		log.Printf("failed to register channel for /feedback: %s", err)
		return
	}

	messageID := h.simpleSend(u.Message.Chat.Id, "Please send your feedback as text message", 0)

	// Delete the message if the user sends the feedback or if the context times out
	defer h.deleteMessage(u.Message.Chat.Id, messageID)

	// Create new command context
	ctx, cancel := context.WithTimeout(h.userCtx[u.Message.Chat.Id].ctx, 1*time.Minute)
	defer cancel()

	// Wait for user response or timeout
	select {
	case <-ctx.Done():
		return
	case u = <-*ch:
	}

	// Create the feedback message and send it to the bot owner
	message := fmt.Sprintf(`
	Feedback Message... 💬

	<b>User ID:</b> %d
	<b>Username:</b> @%s
	<b>Full Name:</b> %s %s
	<b>Message ID:</b> %d
	<b>Feedback:</b> %s
	`, u.Message.Chat.Id, u.Message.Chat.Username, u.Message.Chat.FirstName, u.Message.Chat.LastName, u.Message.MessageId, u.Message.Text)
	_, err = h.b.SendMessage(h.botOwner, message, "HTML", 0, false, false)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An error occurred while sending your feedback. Please try again later.", 0)
		log.Printf("Error: %s, Failed to send feedback message to bot owner", err)
		return
	}

	message = fmt.Sprintf("Thank you for your feedback %s! 😊", u.Message.Chat.FirstName)
	h.simpleSend(u.Message.Chat.Id, message, 0)
}

func (h *Handler) Bug(u *objs.Update) {
	chatID := strconv.Itoa(u.Message.Chat.Id)
	ch, err := h.b.AdvancedMode().RegisterChannel(chatID, "message")
	defer h.b.AdvancedMode().UnRegisterChannel(chatID, "message")
	if err != nil {
		log.Printf("failed to register channel for /bug: %s", err)
		return
	}

	messageID := h.simpleSend(u.Message.Chat.Id, "Please send your bug report as text message", 0)

	// Delete the message if the user sends the bug or if the context times out
	defer h.deleteMessage(u.Message.Chat.Id, messageID)

	// Create new command context
	ctx, cancel := context.WithTimeout(h.userCtx[u.Message.Chat.Id].ctx, 1*time.Minute)
	defer cancel()

	// Wait for user response or timeout
	select {
	case <-ctx.Done():
		return
	case u = <-*ch:
	}
	text := u.Message.Text

	// Send message to bot owner
	message := fmt.Sprintf(`
	Bug Report... 🐞

	<b>User ID:</b> %d
	<b>Username:</b> @%s
	<b>Full Name:</b> %s %s
	<b>Message ID:</b> %d
	<b>Bug Report:</b> %s
	`, u.Message.Chat.Id, u.Message.Chat.Username, u.Message.Chat.FirstName, u.Message.Chat.LastName, u.Message.MessageId, text)
	_, err = h.b.SendMessage(h.botOwner, message, "HTML", 0, false, false)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An error occurred while sending your bug report. Please try again later.", 0)
		log.Printf("Error: %s, Failed to send bug report to bot owner", err)
		return
	}

	// Send response message to user
	message = fmt.Sprintf("Thank you for your bug report %s!\nWe will fix it 🛠️ ASAP.", u.Message.Chat.FirstName)
	h.simpleSend(u.Message.Chat.Id, message, 0)
}

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
	userID, responeMessageID, fullName, ok := parseUserMessage(u.Message.ReplyToMessage.Text)
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
	message := fmt.Sprintf("Hey %s! 👋, Thanks for contacting us! 🙏\n\n%s", fullName, u.Message.Text)
	_, err = h.b.SendMessage(userID, message, "", responeMessageID, false, false)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "Failed to send response.", 0)
		log.Printf("Error: %s, Failed to repond to user message", err)
		return
	}

	h.simpleSend(u.Message.Chat.Id, "Response sent successfully.", 0)
}

func (h *Handler) GetSubscribers(u *objs.Update) {
	ids, err := h.u.GetSubscribers(h.c)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "Failed to get subscribers.", 0)
		log.Printf("Error: %s, Failed to get subscribers", err)
		return
	}
	h.simpleSend(u.Message.Chat.Id, fmt.Sprintf("Subscribers: %d", len(ids)), 0)
}

func (h *Handler) SendAll(u *objs.Update) {
	// Register channel to receive messages
	chatID := strconv.Itoa(u.Message.Chat.Id)
	ch, err := h.b.AdvancedMode().RegisterChannel(chatID, "message")
	defer h.b.AdvancedMode().UnRegisterChannel(chatID, "message")
	if err != nil {
		log.Printf("Error: %s, Failed to register channel", err)
		return
	}

	// Wait for message or timeout after 2 minutes
	messageID := h.simpleSend(u.Message.Chat.Id, "Send your message, Or /cancel", 0)
	defer h.deleteMessage(u.Message.Chat.Id, messageID)

	ctx1, cancel1 := context.WithTimeout(h.userCtx[u.Message.Chat.Id].ctx, 1*time.Minute)
	defer cancel1()

	select {
	case u = <-*ch:
	case <-ctx1.Done():
		return
	}
	broadcastMessage := u.Message.Text

	if h.cancelOperation(u.Message.Text, "Canceled broadcast.", u.Message.Chat.Id) {
		return
	}

	// Double check that the owner still wants to send the message
	messageID = h.simpleSend(u.Message.Chat.Id, "Use /confirm to send the message, Or /cancel", 0)
	defer h.deleteMessage(u.Message.Chat.Id, messageID)

	// Wait for confirmation message or timeout after 5 minutes
	ctx2, cancel2 := context.WithTimeout(h.userCtx[u.Message.Chat.Id].ctx, 1*time.Minute)
	defer cancel2()

	select {
	case <-ctx2.Done():
		return
	case u = <-*ch:
	}

	// Delete the bot message if the user sends the message or if the context times out
	defer h.deleteMessage(u.Message.Chat.Id, u.Message.MessageId)

	if u.Message.Text != "/confirm" || h.cancelOperation(u.Message.Text, "Canceled broadcast.", u.Message.Chat.Id) {
		return
	}
	// Send message to all subscribers in a goroutine
	go func() {
		// Get all subscribers
		ids, err := h.u.GetSubscribers(h.c)
		if err != nil {
			h.simpleSend(u.Message.Chat.Id, "Failed to send message.", 0)
			log.Printf("Error: %s, Failed to get subscribers", err)
			return
		}
		// Send message to all subscribers
		for _, id := range ids {
			h.simpleSend(id, broadcastMessage, 0)
		}
		h.simpleSend(u.Message.Chat.Id, "Message sent successfully.", 0)
	}()
}

// parseUserMessage parses user feedback or bug report
// @param message - user feedback or bug report
// @return userID - user ID who sent feedback or bug report
// @return messageID - message ID of feedback or bug report in user chat
// @return name - user's name who sent feedback or bug report
// @return ok - true if message is valid
// Note that messageID must be the after userID and name, since we break the loop when we find messageID
func parseUserMessage(message string) (userID, messageID int, name string, ok bool) {
	secondArg := func(s string) (string, bool) {
		ss := strings.Split(s, ":")
		if len(ss) != 2 {
			return "", false
		}
		return strings.TrimSpace(ss[1]), true
	}

	var err error
	var sa string // second argument
	for _, line := range strings.Split(message, "\n") {
		// Parse user ID
		if strings.HasPrefix(strings.TrimSpace(line), "User ID:") {
			sa, ok = secondArg(line)
			if !ok {
				return
			}
			userID, err = strconv.Atoi(sa)
			if err != nil {
				return
			}
		}

		// Parse user full name
		if strings.HasPrefix(strings.TrimSpace(line), "Full Name:") {
			sa, ok = secondArg(line)
			if !ok {
				return
			}
			name = sa
		}

		// Parse user message ID
		if strings.HasPrefix(strings.TrimSpace(line), "Message ID:") {
			sa, ok = secondArg(line)
			if !ok {
				return
			}
			messageID, err = strconv.Atoi(sa)
			if err != nil {
				return
			}
			// We don't need to parse other lines, since we have all the data we need
			break
		}
	}
	if userID == 0 || messageID == 0 || name == "" {
		ok = false
		return
	}
	ok = true
	return
}
