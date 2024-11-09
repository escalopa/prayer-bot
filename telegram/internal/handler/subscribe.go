package handler

import (
	objs "github.com/SakoDroid/telego/objects"
	log "github.com/sirupsen/logrus"
)

func (h *Handler) Subscribe(u *objs.Update) {
	chatID := getChatID(u)

	err := h.uc.Subscribe(h.getChatCtx(chatID), chatID)
	if err != nil {
		h.simpleSend(chatID, h.chatScript[chatID].SubscriptionError, 0)
		return
	}
	_, err = h.bot.SendMessage(chatID, h.chatScript[chatID].SubscriptionSuccess, "HTML", 0, false, false)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warn("failed to send subscribe message")
		return
	}
}

func (h *Handler) Unsubscribe(u *objs.Update) {
	chatID := getChatID(u)

	err := h.uc.Unsubscribe(h.getChatCtx(chatID), chatID)
	if err != nil {
		h.simpleSend(chatID, h.chatScript[chatID].UnsubscriptionError, 0)
		return
	}

	_, err = h.bot.SendMessage(chatID, h.chatScript[chatID].UnsubscriptionSuccess, "HTML", 0, false, false)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warn("failed to send unsubscribe message")
		return
	}
}
