package handler

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	objs "github.com/SakoDroid/telego/objects"
)

const (
	botOwnerID = 1385434843
)

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
		return
	}

	h.simpleSend(u.Message.Chat.Id, "Please send your feedback as text message", 0)
	u = <-*ch
	text := u.Message.Text

	message := fmt.Sprintf(`
	Feedback Message... 💬

	<b>User ID:</b> %d
	<b>Username:</b> @%s
	<b>Full Name:</b> %s %s
	<b>Message ID:</b> %d
	<b>Feedback:</b> %s
	`, u.Message.Chat.Id, u.Message.Chat.Username, u.Message.Chat.FirstName, u.Message.Chat.LastName, u.Message.MessageId, text)
	_, err = h.b.SendMessage(botOwnerID, message, "HTML", 0, false, false)
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
		return
	}

	h.simpleSend(u.Message.Chat.Id, "Please send your bug report as text message", 0)
	u = <-*ch
	text := u.Message.Text

	message := fmt.Sprintf(`
	Bug Report... 🐞

	<b>User ID:</b> %d
	<b>Username:</b> @%s
	<b>Full Name:</b> %s %s
	<b>Message ID:</b> %d
	<b>Bug Report:</b> %s
	`, u.Message.Chat.Id, u.Message.Chat.Username, u.Message.Chat.FirstName, u.Message.Chat.LastName, u.Message.MessageId, text)
	_, err = h.b.SendMessage(botOwnerID, message, "HTML", 0, false, false)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An error occurred while sending your bug report. Please try again later.", 0)
		log.Printf("Error: %s, Failed to send bug report to bot owner", err)
		return
	}

	message = fmt.Sprintf("Thank you for your bug report %s!\nWe will fix it 🛠️ ASAP.", u.Message.Chat.FirstName)
	h.simpleSend(u.Message.Chat.Id, message, 0)

}

func (h *Handler) Respond(u *objs.Update) {
	// Only bot owner can use this command
	if u.Message.Chat.Id != botOwnerID {
		return
	}

	chatID := strconv.Itoa(u.Message.Chat.Id)
	ch, err := h.b.AdvancedMode().RegisterChannel(chatID, "message")
	defer h.b.AdvancedMode().UnRegisterChannel(chatID, "message")
	if err != nil {
		return
	}

	// Check if reply message is provided
	if u.Message.ReplyToMessage == nil {
		h.simpleSend(u.Message.Chat.Id, "No reply message provided, /respond", 0)
		return
	}

	// Read userID, messageID, username from the old message that will be replied to
	userID, messageID, fullName, ok := parseUserMessage(u.Message.ReplyToMessage.Text)
	if !ok {
		h.simpleSend(u.Message.Chat.Id, "Invalid message.", 0)
		return
	}

	// Read response message
	h.simpleSend(u.Message.Chat.Id, "Send your response message, Or /cancel", 0)
	u = <-*ch
	response := u.Message.Text
	if h.cancelOperation(response, "Canceled response.", u.Message.Chat.Id) {
		return
	}

	message := fmt.Sprintf("Hey %s! 👋, Thanks for contacting us! 🙏\n\n%s", fullName, response)
	_, err = h.b.SendMessage(userID, message, "", messageID, false, false)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "Failed to send response.", 0)
		log.Printf("Error: %s, Failed to repond to user message", err)
		return
	}

	h.simpleSend(u.Message.Chat.Id, "Response sent successfully.", 0)

}

func (h *Handler) GetSubscribers(u *objs.Update) {
	// Only bot owner can use this command
	if u.Message.Chat.Id != botOwnerID {
		return
	}
	ids, err := h.u.GetSubscribers(h.c)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "Failed to get subscribers.", 0)
		log.Printf("Error: %s, Failed to get subscribers", err)
		return
	}
	h.simpleSend(u.Message.Chat.Id, fmt.Sprintf("Subscribers: %d", len(ids)), 0)
}

func (h *Handler) SendAll(u *objs.Update) {
	// Only bot owner can use this command
	if u.Message.Chat.Id != botOwnerID {
		return
	}
	// Register channel to receive messages
	chatID := strconv.Itoa(u.Message.Chat.Id)
	ch, err := h.b.AdvancedMode().RegisterChannel(chatID, "message")
	defer h.b.AdvancedMode().UnRegisterChannel(chatID, "message")
	if err != nil {
		log.Printf("Error: %s, Failed to register channel", err)
		return
	}

	// Wait for message or timeout after 5 minutes
	h.simpleSend(u.Message.Chat.Id, "Send your message, Or /cancel", 0)
	ctx1, cancel1 := context.WithTimeout(h.c, 5*time.Minute)
	defer cancel1()
	select {
	case u = <-*ch:
	case <-ctx1.Done():
		return
	}
	message := u.Message.Text
	if h.cancelOperation(message, "Canceled broadcast.", u.Message.Chat.Id) {
		return
	}

	// Double check that the owner still wants to send the message
	// Wait for confirmation message or timeout after 5 minutes
	h.simpleSend(u.Message.Chat.Id, "Use /confirm to send the message, Or /cancel", 0)
	ctx2, cancel2 := context.WithTimeout(h.c, 5*time.Minute)
	defer cancel2()
	select {
	case u = <-*ch:
	case <-ctx2.Done():
		return
	}
	confirm := u.Message.Text
	if confirm != "/confirm" || h.cancelOperation(confirm, "Canceled broadcast.", u.Message.Chat.Id) {
		return
	}
	ids, err := h.u.GetSubscribers(h.c)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "Failed to get subscribers.", 0)
		log.Printf("Error: %s, Failed to get subscribers", err)
		return
	}
	// Send message to all subscribers in a goroutine
	go func() {
		for _, id := range ids {
			h.simpleSend(id, message, 0)
		}
	}()
	h.simpleSend(u.Message.Chat.Id, "Message sent successfully.", 0)
}

// parseUserMessage parses user feedback or bug report
// @param message - user feedback or bug report
// @return userID - user ID who sent feedback or bug report
// @return messageID - message ID of feedback or bug report in user chat
// @return name - user name who sent feedback or bug report
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
