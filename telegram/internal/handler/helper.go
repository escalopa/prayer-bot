package handler

import (
	"context"
	"strconv"
	"time"

	objs "github.com/SakoDroid/telego/objects"
	log "github.com/catalystgo/logger/cli"
	"github.com/escalopa/gopray/telegram/internal/domain"
	"github.com/pkg/errors"
)

const (
	inputTimeout = 10 * time.Minute
)

// simpleSend sends a simple text message to the chatID
func (h *Handler) simpleSend(chatID int, text string) (messageID int) {
	return h.reply(chatID, text, 0)
}

// reply sends a text message to the chatID with the given replyTo messageID.
func (h *Handler) reply(chatID int, text string, replyTo int) (messageID int) {
	r, err := h.bot.SendMessage(chatID, text, "", replyTo, false, false)
	if err != nil {
		log.Errorf("Handler.SendMessage: [%d] [%d] [%s] => %v", chatID, replyTo, text, err)
		return 0
	}
	return r.Result.MessageId
}

// deleteMessage deletes the message with the given chatID & messageID.
// If error occurs, it will be logged.
func (h *Handler) deleteMessage(chatID, messageID int) {
	if messageID == 0 {
		return
	}

	editor := h.bot.GetMsgEditor(chatID)
	_, err := editor.DeleteMessage(messageID)
	if err != nil {
		log.Errorf("Handler.deleteMessage: [%d] [%d] => %v", chatID, messageID, err)
		return
	}
}

// replace is a helper function to delete the last message id and store the new one's id in the database.
func (h *Handler) replace(chatID int, messageID int) {
	lastMessageId, err := h.uc.GetPrayerMessageID(h.ctx, chatID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		log.Errorf("Handler.replace: [%d] => %v", chatID, err)
	}

	// Delete the last messageID in chatID if exists.
	h.deleteMessage(chatID, lastMessageId)

	// Store the new messageID chatID.
	err = h.uc.StorePrayerMessageID(h.ctx, chatID, messageID)
	if err != nil {
		log.Errorf("Handler.replace: [%d] => %v", chatID, err)
	}
}

func (h *Handler) registerChannel(chatID int) (chan *objs.Update, func(), error) {
	chatIDStr := strconv.Itoa(chatID)
	ch, err := h.bot.AdvancedMode().RegisterChannel(chatIDStr, "message")
	closer := func() {
		if ch == nil {
			return
		}
		h.bot.AdvancedMode().UnRegisterChannel(chatIDStr, "message")
	}
	return *ch, closer, err
}

// isCancelOperation checks if the message is /cancel and sends a response.
// Returns true if the message is /cancel.
func (h *Handler) isCancelOperation(chatID int, message string) bool {
	if message == cmdCancel {
		h.simpleSend(chatID, operationCanceled)
		return true
	}
	return false
}

func (h *Handler) getChatCtx(chatID int) context.Context {
	h.internal.chatCtxMu.RLock()
	defer h.internal.chatCtxMu.RUnlock()

	userCtx, _ := h.internal.chatCtx[chatID]
	return userCtx.ctx
}

func (h *Handler) renewChatCtx(chatID int) {
	h.internal.chatCtxMu.Lock()
	defer h.internal.chatCtxMu.Unlock()

	// Create new context for user
	ctx, cancel := context.WithCancel(h.ctx)

	// Cancel previous context if exists
	if uc, ok := h.internal.chatCtx[chatID]; ok {
		uc.cancel()
	}

	// Set new context
	h.internal.chatCtx[chatID] = userContext{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (h *Handler) getChatScript(chatID int) *domain.Script {
	h.internal.chatScriptMu.RLock()
	defer h.internal.chatScriptMu.RUnlock()

	script, _ := h.internal.chatScript[chatID]
	return script
}

func (h *Handler) setChatScript(chatID int, script *domain.Script) {
	h.internal.chatScriptMu.Lock()
	defer h.internal.chatScriptMu.Unlock()

	h.internal.chatScript[chatID] = script
}

func getChatID(u *objs.Update) int {
	return u.Message.Chat.Id
}
