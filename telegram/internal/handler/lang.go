package handler

import (
	objs "github.com/SakoDroid/telego/objects"
)

func (h *Handler) SetLang(u *objs.Update) {
	// TODO: Implement SetLang
	if true {
		h.simpleSend(u.Message.Chat.Id, "This feature is not available yet.", 0)
		return
	}
	err := h.ac.SetLang(u.Message.Chat.Id, u.Message.Text)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An error occurred while setting language. Please try again later.", 0)
		return
	}
	h.simpleSend(u.Message.Chat.Id, "Language set successfully.", 0)
}
