package handler

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/escalopa/gopray/telegram/internal/domain"

	objs "github.com/SakoDroid/telego/objects"
)

func (h *Handler) Start(u *objs.Update) {
	chatID := u.Message.Chat.Id
	if _, err := h.uc.GetLang(h.ctx, chatID); err != nil {
		// Check that the user language is valid & supported.
		userLang := u.Message.From.LanguageCode
		if !domain.IsValidLang(userLang) {
			userLang = domain.DefaultLang().Short
		}
		// Set the user language in the database.
		go func() {
			err = h.uc.SetLang(h.chatCtx[chatID].ctx, chatID, userLang)
			if err != nil {
				log.Printf("failed to set user language on /start: %s", err)
			}
		}()
		// Get & set the user script.
		script, err := h.uc.GetScript(h.chatCtx[chatID].ctx, userLang)
		if err != nil {
			log.Printf("failed to get script for %s: %v", userLang, err)
		}
		h.chatScript[chatID] = script
	}
	h.Help(u)
}

func (h *Handler) Help(u *objs.Update) {
	_, err := h.bot.SendMessage(u.Message.Chat.Id, h.chatScript[u.Message.Chat.Id].Help, "HTML", 0, false, false)
	if err != nil {
		log.Printf("failed to send message on /help: %s", err)
	}
}

func (h *Handler) Feedback(u *objs.Update) {
	chatID := u.Message.Chat.Id
	ch, err := h.bot.AdvancedMode().RegisterChannel(strconv.Itoa(u.Message.Chat.Id), "message")
	defer h.bot.AdvancedMode().UnRegisterChannel(strconv.Itoa(u.Message.Chat.Id), "message")
	if err != nil {
		log.Printf("failed to register channel for /feedback: %s", err)
		return
	}

	messageID := h.simpleSend(chatID, h.chatScript[chatID].FeedbackStart, 0)

	// Delete the message if the user sends the feedback or if the context times out
	defer h.deleteMessage(chatID, messageID)

	// Create new command context
	ctx, cancel := context.WithTimeout(h.chatCtx[chatID].ctx, 1*time.Minute)
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

	<bot>User ID:</bot> %d
	<bot>Username:</bot> @%s
	<bot>Full Name:</bot> %s %s
	<bot>Message ID:</bot> %d
	<bot>Feedback:</bot> %s
	`, u.Message.Chat.Id, u.Message.Chat.Username, u.Message.Chat.FirstName, u.Message.Chat.LastName, u.Message.MessageId, u.Message.Text)
	_, err = h.bot.SendMessage(h.botOwner, message, "HTML", 0, false, false)
	if err != nil {
		log.Printf("failed to send feedback message /feedback: %s", err)
		h.simpleSend(u.Message.Chat.Id, h.chatScript[chatID].FeedbackFail, 0)
		return
	}

	message = fmt.Sprintf(h.chatScript[chatID].FeedbackSuccess, u.Message.Chat.FirstName)
	h.simpleSend(u.Message.Chat.Id, message, 0)
}

func (h *Handler) Bug(u *objs.Update) {
	chatID := u.Message.Chat.Id
	ch, err := h.bot.AdvancedMode().RegisterChannel(strconv.Itoa(u.Message.Chat.Id), "message")
	defer h.bot.AdvancedMode().UnRegisterChannel(strconv.Itoa(u.Message.Chat.Id), "message")
	if err != nil {
		log.Printf("failed to register channel for /bug: %s", err)
		return
	}

	// Delete the message if the user sends the bug or if the context times out
	messageID := h.simpleSend(chatID, h.chatScript[chatID].BugReportStart, 0)
	defer h.deleteMessage(chatID, messageID)

	// Create new command context
	ctx, cancel := context.WithTimeout(h.chatCtx[chatID].ctx, 1*time.Minute)
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

	<bot>User ID:</bot> %d
	<bot>Username:</bot> @%s
	<bot>Full Name:</bot> %s %s
	<bot>Message ID:</bot> %d
	<bot>Bug Report:</bot> %s
	`, chatID, u.Message.Chat.Username, u.Message.Chat.FirstName, u.Message.Chat.LastName, u.Message.MessageId, text)
	_, err = h.bot.SendMessage(h.botOwner, message, "HTML", 0, false, false)
	if err != nil {
		h.simpleSend(chatID, h.chatScript[chatID].BugReportFail, 0)
		log.Printf("failed to send bug report message /bug: %s", err)
		return
	}

	// Send response message to user
	message = fmt.Sprintf(h.chatScript[chatID].BugReportSuccess, u.Message.Chat.FirstName)
	h.simpleSend(chatID, message, 0)
}

// parseUserMessage parses user's feedback or bug report
func parseUserMessage(message string) (chatID int, messageID int, name string, ok bool) {
	secondArg := func(s string) (string, bool) {
		ss := strings.Split(s, ":")
		if len(ss) != 2 {
			return "", false
		}
		return strings.TrimSpace(ss[1]), true
	}

	var (
		err error
		sa  string // second argument
	)

	for _, line := range strings.Split(message, "\n") {
		// Parse user ID
		if strings.HasPrefix(strings.TrimSpace(line), "User ID:") {
			sa, ok = secondArg(line)
			if !ok {
				return
			}
			chatID, err = strconv.Atoi(sa)
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
	if chatID == 0 || messageID == 0 || name == "" {
		ok = false
		return
	}
	ok = true
	return
}
