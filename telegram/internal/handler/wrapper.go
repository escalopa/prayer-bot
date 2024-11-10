package handler

import (
	"context"

	log "github.com/catalystgo/logger/cli"

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
	return func(u *objs.Update) {
		h.renewChatCtx(getChatID(u))
		command(u)
	}
}

// useScript is a wrapper for user commands to load user script if not loaded
func (h *Handler) useScript(command func(u *objs.Update)) func(u *objs.Update) {
	return func(u *objs.Update) {
		chatID := getChatID(u)

		err := h.setScript(chatID)
		if err != nil {
			log.Errorf("Handler.useScript: [%d] => %v", chatID, err)
			h.simpleSend(chatID, unexpectedErrMsg)
			return
		}

		command(u)
	}

}
