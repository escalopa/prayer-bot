package handler

import (
	objs "github.com/SakoDroid/telego/objects"
)

func (h *Handler) NotifyPrayers() {
	h.ac.Notify(func(id int, msg string) {
		h.b.SendMessage(id, msg, "HTML", 0, false, false)
	})
}

func (h *Handler) Subscribe(u *objs.Update) {
	err := h.ac.Subscribe(u.Message.Chat.Id)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An unexpected error occurred while subscribing. Please try again later.", 0)
		return
	}
	h.b.SendMessage(u.Message.Chat.Id, "You have been <b>Subscribed</b> to the daily prayer notifications.", "HTML", 0, false, false)
}

func (h *Handler) Unsubscribe(u *objs.Update) {
	err := h.ac.Unsubscribe(u.Message.Chat.Id)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An unexpected error occurred while unsubscribing. Please try again later.", 0)
		return
	}
	h.b.SendMessage(u.Message.Chat.Id, "You have been <b>Unsubscribed</b> from the daily prayer notifications.", "HTML", 0, false, false)
}
