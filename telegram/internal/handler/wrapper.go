package handler

import (
	"context"
	"log"

	objs "github.com/SakoDroid/telego/objects"
)

type userContext struct {
	ctx    context.Context
	cancel func()
}

// useAdmin is a wrapper for useAdmin commands to check if the user is the bot owner
func (h *Handler) useAdmin(command func(u *objs.Update)) func(u *objs.Update) {
	return func(u *objs.Update) {
		if u.Message.From.Id == h.botOwner {
			command(u)
		} else {
			h.Help(u)
		}
	}
}

// useContext is a wrapper for user commands to create a new context for each user
// and cancel the previous context if exists
func (h *Handler) useContext(command func(u *objs.Update)) func(update *objs.Update) {
	return func(update *objs.Update) {
		// Create new context for user
		newCtx, cancel := context.WithCancel(h.ctx)

		// Cancel previous context if exists
		if uc, ok := h.chatCtx[update.Message.Chat.Id]; ok {
			uc.cancel()
		}

		// Set new context
		h.chatCtx[update.Message.Chat.Id] = userContext{
			ctx:    newCtx,
			cancel: cancel,
		}

		// Call user command
		command(update)
	}
}

// useScript is a wrapper for user commands to load user script if not loaded
func (h *Handler) useScript(command func(u *objs.Update)) func(u *objs.Update) {
	return func(u *objs.Update) {
		chatID := getChatID(u)

		err := h.setScript(chatID)
		if err != nil {
			log.Printf("Handler.useScript: %v", err)
			h.simpleSend(chatID, unexpectedErrMsg, 0)
			return
		}

		command(u)
	}

}
