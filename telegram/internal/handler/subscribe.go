package handler

import (
	objs "github.com/SakoDroid/telego/objects"
)

func (h *Handler) Subscribe(u *objs.Update) {
	// TODO: Implement this
	if true {
		h.simpleSend(u.Message.Chat.Id, "This feature is not available yet.", 0)
		return
	}
	err := h.ac.Subscribe(u.Message.Chat.Id)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An unexpected error occurred while subscribing. Please try again later.", 0)
		return
	}
}

func (h *Handler) Unsubscribe(u *objs.Update) {
	// TODO: Implement this
	if true {
		h.simpleSend(u.Message.Chat.Id, "This feature is not available yet.", 0)
		return
	}
	err := h.ac.Unsubscribe(u.Message.Chat.Id)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An unexpected error occurred while unsubscribing. Please try again later.", 0)
		return
	}
}
