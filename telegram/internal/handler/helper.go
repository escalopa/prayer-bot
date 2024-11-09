package handler

import (
	"context"

	objs "github.com/SakoDroid/telego/objects"
	"github.com/escalopa/gopray/telegram/internal/domain"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// simpleSend sends a simple message to the chat with the given chatID & text and replyTo.
func (h *Handler) simpleSend(chatID int, text string, replyTo int) (messageID int) {
	r, err := h.bot.SendMessage(chatID, text, "", replyTo, false, false)
	if err != nil {
		log.Printf("failed to send message on simpleSend: %s", err)
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
		log.Printf("failed to delete message: %s", err)
		return
	}
}

// replace is a helper function to delete the last message id and store the new one's id in the database.
func (h *Handler) replace(chatID int, messageID int) {
	lastMessageId, err := h.uc.GetPrayerMessageID(h.ctx, chatID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		log.WithFields(log.Fields{"error": err}).Warn("failed to remove last messageID chatID /notify")
	}

	// Delete the last messageID in chatID if exists.
	h.deleteMessage(chatID, lastMessageId)

	// Store the new messageID chatID.
	err = h.uc.StorePrayerMessageID(h.ctx, chatID, messageID)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warn("failed to replace messageID chatID /notify")
	}
}

// isCancelOperation checks if the message is /cancel and sends a response.
// Returns true if the message is /cancel.
func (h *Handler) isCancelOperation(chatID int, message string) bool {
	if message == "/cancel" {
		h.simpleSend(chatID, operationCanceled, 0)
		return true
	}
	return false
}

func (h *Handler) getChatCtx(chatID int) context.Context {
	h.chatCtxMu.RLock()
	defer h.chatCtxMu.RUnlock()

	userCtx, _ := h.chatCtx[chatID]
	return userCtx.ctx
}

func (h *Handler) renewChatCtx(chatID int) {
	h.chatCtxMu.Lock()
	defer h.chatCtxMu.Unlock()

	// Create new context for user
	ctx, cancel := context.WithCancel(h.ctx)

	// Cancel previous context if exists
	if uc, ok := h.chatCtx[chatID]; ok {
		uc.cancel()
	}

	// Set new context
	h.chatCtx[chatID] = userContext{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (h *Handler) getChatScript(chatID int) *domain.Script {
	h.chatScriptMu.RLock()
	defer h.chatScriptMu.RUnlock()

	script, _ := h.chatScript[chatID]
	return script
}

func (h *Handler) setChatScript(chatID int, script *domain.Script) {
	h.chatScriptMu.Lock()
	defer h.chatScriptMu.Unlock()

	h.chatScript[chatID] = script
}

func getChatID(u *objs.Update) int {
	return u.Message.Chat.Id
}

func isConfirmOperation(text string) bool {
	return text == "/confirm"
}
