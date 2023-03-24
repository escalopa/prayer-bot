package handler

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/escalopa/gopray/pkg/language"

	objs "github.com/SakoDroid/telego/objects"
)

func (h *Handler) Start(u *objs.Update) {
	chatID := u.Message.Chat.Id
	if _, err := h.u.GetLang(h.c, chatID); err != nil {
		// Check that the user language is valid & supported.
		userLang := u.Message.From.LanguageCode
		if !language.IsValidLang(userLang) {
			userLang = language.DefaultLang().Short
		}
		// Set the user language in the database.
		go func() {
			err = h.u.SetLang(h.userCtx[chatID].ctx, chatID, userLang)
			if err != nil {
				log.Printf("failed to set user language on /start: %s", err)
			}
		}()
		// Get & set the user script.
		script, err := h.u.GetScript(h.userCtx[chatID].ctx, userLang)
		if err != nil {
			log.Printf("failed to get script for %s: %v", userLang, err)
		}
		h.userScript[chatID] = script
	}
	h.Help(u)
}

func (h *Handler) Help(u *objs.Update) {
	_, err := h.b.SendMessage(u.Message.Chat.Id, h.userScript[u.Message.Chat.Id].Help, "HTML", 0, false, false)
	if err != nil {
		log.Printf("failed to send message on /help: %s", err)
	}
}

func (h *Handler) Feedback(u *objs.Update) {
	chatID := u.Message.Chat.Id
	ch, err := h.b.AdvancedMode().RegisterChannel(strconv.Itoa(u.Message.Chat.Id), "message")
	defer h.b.AdvancedMode().UnRegisterChannel(strconv.Itoa(u.Message.Chat.Id), "message")
	if err != nil {
		log.Printf("failed to register channel for /feedback: %s", err)
		return
	}

	messageID := h.simpleSend(chatID, h.userScript[chatID].FeedbackStart, 0)

	// Delete the message if the user sends the feedback or if the context times out
	defer h.deleteMessage(chatID, messageID)

	// Create new command context
	ctx, cancel := context.WithTimeout(h.userCtx[chatID].ctx, 1*time.Minute)
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
		log.Printf("failed to send feedback message /feedback: %s", err)
		h.simpleSend(u.Message.Chat.Id, h.userScript[chatID].FeedbackFail, 0)
		return
	}

	message = fmt.Sprintf(h.userScript[chatID].FeedbackSuccess, u.Message.Chat.FirstName)
	h.simpleSend(u.Message.Chat.Id, message, 0)
}

func (h *Handler) Bug(u *objs.Update) {
	chatID := u.Message.Chat.Id
	ch, err := h.b.AdvancedMode().RegisterChannel(strconv.Itoa(u.Message.Chat.Id), "message")
	defer h.b.AdvancedMode().UnRegisterChannel(strconv.Itoa(u.Message.Chat.Id), "message")
	if err != nil {
		log.Printf("failed to register channel for /bug: %s", err)
		return
	}

	// Delete the message if the user sends the bug or if the context times out
	messageID := h.simpleSend(chatID, h.userScript[chatID].BugReportStart, 0)
	defer h.deleteMessage(chatID, messageID)

	// Create new command context
	ctx, cancel := context.WithTimeout(h.userCtx[chatID].ctx, 1*time.Minute)
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
	`, chatID, u.Message.Chat.Username, u.Message.Chat.FirstName, u.Message.Chat.LastName, u.Message.MessageId, text)
	_, err = h.b.SendMessage(h.botOwner, message, "HTML", 0, false, false)
	if err != nil {
		h.simpleSend(chatID, h.userScript[chatID].BugReportFail, 0)
		log.Printf("failed to send bug report message /bug: %s", err)
		return
	}

	// Send response message to user
	message = fmt.Sprintf(h.userScript[chatID].BugReportSuccess, u.Message.Chat.FirstName)
	h.simpleSend(chatID, message, 0)
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
