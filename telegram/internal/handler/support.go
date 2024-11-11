package handler

import (
	"fmt"
	"strconv"
	"strings"

	objs "github.com/SakoDroid/telego/objects"
	log "github.com/catalystgo/logger/cli"
	"github.com/escalopa/gopray/telegram/internal/domain"
)

// Start is the first command that the user will see when they start the bot.
// It will set the user's language and show the help message.
func (h *Handler) Start(u *objs.Update) {
	var (
		chatID = getChatID(u)
		ctx    = h.getChatCtx(chatID)
	)

	if _, err := h.uc.GetLang(h.ctx, chatID); err == nil {
		h.Help(u)
		return
	}

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
		chatID = getChatID(u)

		ctx    = h.getChatCtx(chatID)
		script = h.getChatScript(chatID)
	)

	ch, closer, err := h.registerChannel(chatID)
	if err != nil {
		log.Errorf("Handler.Feedback: register channel [%d] => %v", chatID, err)
		h.simpleSend(chatID, script.FeedbackFail)
		return
	}
	defer closer()

	messageID := h.simpleSend(chatID, script.FeedbackStart)
	defer h.deleteMessage(chatID, messageID)

	u = h.readInput(ctx, ch)
	if u == nil {
		return
	}

	// Create the feedback message and send it to the bot owner
	messageToBotOwner := fmt.Sprintf(feedbackSendMsg,
		chatID,
		u.Message.MessageId,
		u.Message.Chat.Username,
	)

	h.sendToOwner(
		chatID,
		u.Message.MessageId,
		messageToBotOwner,
		fmt.Sprintf(script.FeedbackSuccess, u.Message.Chat.FirstName),
		script.FeedbackFail,
	)

}

func (h *Handler) Bug(u *objs.Update) {
	var (
		chatID = getChatID(u)

		ctx    = h.getChatCtx(chatID)
		script = h.getChatScript(chatID)
	)

	ch, closer, err := h.registerChannel(chatID)
	if err != nil {
		log.Errorf("Handler.Bug: register channel [%d] => %v", chatID, err)
		h.simpleSend(chatID, script.BugReportFail)
		return
	}
	defer closer()

	// Delete the message if the user sends the bug or if the context times out
	messageID := h.simpleSend(chatID, script.BugReportStart)
	defer h.deleteMessage(chatID, messageID)

	u = h.readInput(ctx, ch)
	if u == nil {
		return
	}

	messageToBotOwner := fmt.Sprintf(bugSendMsg,
		chatID,
		u.Message.MessageId,
		u.Message.Chat.Username,
	)

	h.sendToOwner(
		chatID,
		u.Message.MessageId,
		messageToBotOwner,
		fmt.Sprintf(script.BugReportSuccess, u.Message.Chat.FirstName),
		script.BugReportFail,
	)
}

// sendToOwner sends a message from bot's user to bot's owner
//
//	chatID - user's ChatID
//	messageID - user's message that need to be sent to the owner
//	messageDataToOwner - data about user's message that is used by /respond cmd
//	messageOnSuccess - text to send to user on success
//	messageOnFail - text to send to user on fail
func (h *Handler) sendToOwner(
	chatID int,
	messageID int,
	messageDataToOwner string,
	messageOnSuccess string,
	messageOnFail string,
) {
	forward := h.bot.ForwardMessage(messageID, false, false)
	res, err := forward.ForwardFromUserToUser(h.botOwner, chatID)
	if err != nil {
		log.Errorf("Handler.sendToOwner: [%d] => %v", chatID, err)
		h.simpleSend(chatID, messageOnFail)
		return
	}

	_, err = h.bot.SendMessage(h.botOwner, messageDataToOwner, "HTML", res.Result.MessageId, false, false)
	if err != nil {
		log.Errorf("Handler.sendToOwner: [%d] => %v", chatID, err)
		h.simpleSend(chatID, messageOnFail)
		return
	}

	h.simpleSend(chatID, messageOnSuccess)
}

// parseUserMessage parses user's feedback or bug report
func parseUserMessage(message string) (chatID int, messageID int, ok bool) {
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
		if strings.HasPrefix(strings.TrimSpace(line), "ChatID:") {
			sa, ok = secondArg(line)
			if !ok {
				return
			}
			chatID, err = strconv.Atoi(sa)
			if err != nil {
				return
			}
		}

		// Parse user message ID
		if strings.HasPrefix(strings.TrimSpace(line), "MessageID:") {
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

	if chatID == 0 || messageID == 0 {
		ok = false
		return
	}
	ok = true
	return
}
