package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	log "github.com/catalystgo/logger/cli"

	"github.com/escalopa/gopray/telegram/internal/domain"

	objs "github.com/SakoDroid/telego/objects"
)

func (h *Handler) Start(u *objs.Update) {
	chatID := getChatID(u)
	if _, err := h.uc.GetLang(h.ctx, chatID); err == nil {
		h.Help(u)
		return
	}

	ctx := h.getChatCtx(chatID)

	// Check that the user language is valid & supported.
	userLang := u.Message.From.LanguageCode
	if !domain.IsValidLang(userLang) {
		userLang = domain.DefaultLang().Short
	}

	err := h.uc.SetLang(ctx, chatID, userLang)
	if err != nil {
		log.Errorf("Handler.Start: [%d] => %v", chatID, err)
	}

	// Get & set the user script.
	script, err := h.uc.GetScript(ctx, userLang)
	if err != nil {
		log.Errorf("Handler.Start: [%d] => %v", chatID, err)
	}
	h.setChatScript(chatID, script)

	h.Help(u)
}

func (h *Handler) Help(u *objs.Update) {
	chatID := getChatID(u)
	_, err := h.bot.SendMessage(chatID, h.getChatScript(chatID).Help, "HTML", 0, false, false)
	if err != nil {
		log.Errorf("Handler.Help: [%d] => %v", u.Message.Chat.Id, err)
	}
}

func (h *Handler) Feedback(u *objs.Update) {
	var (
		chatID    = getChatID(u)
		chatIDStr = strconv.Itoa(chatID)

		ctx    = h.getChatCtx(chatID)
		script = h.getChatScript(chatID)
	)

	ch, closer, err := h.registerChannel(chatIDStr)
	if err != nil {
		log.Errorf("Handler.Feedback: register channel [%d] => %v", chatID, err)
		h.simpleSend(chatID, script.FeedbackFail)
		return
	}
	defer closer()

	messageID := h.simpleSend(chatID, script.FeedbackStart)
	defer h.deleteMessage(chatID, messageID)

	// Create new command context
	ctx, cancel := context.WithTimeout(ctx, inputTimeout)
	defer cancel()

	// Wait for user response or timeout
	select {
	case <-ctx.Done():
		return
	case u = <-ch:
	}

	// Create the feedback message and send it to the bot owner
	message := fmt.Sprintf(feedbackSendMsg, chatID,
		u.Message.Chat.Username,
		u.Message.Chat.FirstName,
		u.Message.Chat.LastName,
		u.Message.MessageId,
		u.Message.Text,
	)

	_, err = h.bot.SendMessage(h.botOwner, message, "HTML", 0, false, false)
	if err != nil {
		log.Errorf("Handler.Feedback: [%d] => %v", chatID, err)
		h.simpleSend(u.Message.Chat.Id, script.FeedbackFail)
		return
	}

	message = fmt.Sprintf(script.FeedbackSuccess, u.Message.Chat.FirstName)
	h.simpleSend(chatID, message)
}

func (h *Handler) Bug(u *objs.Update) {
	var (
		chatID    = getChatID(u)
		chatIDStr = strconv.Itoa(chatID)

		script = h.getChatScript(chatID)
	)

	ch, closer, err := h.registerChannel(chatIDStr)
	if err != nil {
		log.Errorf("Handler.Bug: register channel [%d] => %v", chatID, err)
		h.simpleSend(chatID, script.BugReportFail)
		return
	}
	defer closer()

	// Delete the message if the user sends the bug or if the context times out
	messageID := h.simpleSend(chatID, script.BugReportStart)
	defer h.deleteMessage(chatID, messageID)

	// Create new command context
	ctx, cancel := context.WithTimeout(h.getChatCtx(chatID), 1*time.Minute)
	defer cancel()

	// Wait for user response or timeout
	select {
	case <-ctx.Done():
		return
	case u = <-ch:
	}
	text := u.Message.Text

	// Send message to bot owner
	message := fmt.Sprintf(bugSendMsg,
		chatID,
		u.Message.Chat.Username,
		u.Message.Chat.FirstName,
		u.Message.Chat.LastName,
		u.Message.MessageId,
		text,
	)
	_, err = h.bot.SendMessage(h.botOwner, message, "HTML", 0, false, false)
	if err != nil {
		log.Errorf("Handler.Bug: [%d] => %v", chatID, err)
		h.simpleSend(chatID, script.BugReportFail)
		return
	}

	// Send response message to user
	message = fmt.Sprintf(script.BugReportSuccess, u.Message.Chat.FirstName)
	h.simpleSend(chatID, message)
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
