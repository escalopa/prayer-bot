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

// contextWrapper is a wrapper for user commands to create a new context for each user
// and cancel the previous context if exists
func (h *Handler) contextWrapper(command func(u *objs.Update)) func(update *objs.Update) {
	return func(update *objs.Update) {
		// Create new context for user
		newCtx, cancel := context.WithCancel(h.c)

		// Cancel previous context if exists
		if uc, ok := h.userCtx[update.Message.Chat.Id]; ok {
			uc.cancel()
		}

		// Set new context
		h.userCtx[update.Message.Chat.Id] = userContext{
			ctx:    newCtx,
			cancel: cancel,
		}

		// Call user command
		command(update)
	}
}

// scriptWrapper is a wrapper for user commands to load user script if not loaded
func (h *Handler) scriptWrapper(command func(u *objs.Update)) func(u *objs.Update) {
	return func(u *objs.Update) {
		err := h.setScript(u.Message.Chat.Id)
		if err != nil {
			log.Printf("failed to set script on scriptWrapper: %v", err)
			h.simpleSend(u.Message.Chat.Id, "unexpected error, Use /bug to report the error if it remains", 0)
			return
		}
		command(u)
	}

}
