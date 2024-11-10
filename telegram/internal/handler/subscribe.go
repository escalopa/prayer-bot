package handler

import (
	objs "github.com/SakoDroid/telego/objects"
	log "github.com/catalystgo/logger/cli"
)

func (h *Handler) Subscribe(u *objs.Update) {
	var (
		chatID = getChatID(u)
		script = h.getChatScript(chatID)
	)

	err := h.uc.Subscribe(h.getChatCtx(chatID), chatID)
	if err != nil {
		log.Errorf("Handler.Subscribe: [%d] => %v", chatID, err)
		h.simpleSend(chatID, script.SubscriptionError)
		return
	}

	h.simpleSend(chatID, script.SubscriptionSuccess)
}

func (h *Handler) Unsubscribe(u *objs.Update) {
	var (
		chatID = getChatID(u)
		script = h.getChatScript(chatID)
	)

	err := h.uc.Unsubscribe(h.getChatCtx(chatID), chatID)
	if err != nil {
		log.Errorf("Handler.Unsubscribe: [%d] => %v", chatID, err)
		h.simpleSend(chatID, script.UnsubscriptionError)
		return
	}

	h.simpleSend(chatID, script.UnsubscriptionSuccess)
}
