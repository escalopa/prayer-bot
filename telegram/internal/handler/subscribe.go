package handler

import (
	"log"

	objs "github.com/SakoDroid/telego/objects"
)

func (h *Handler) NotifySubscribers() {
	h.ac.Notify(func(id int, msg string) {
		_, err := h.b.SendMessage(id, msg, "HTML", 0, false, false)
		if err != nil {
			log.Printf("Err: %s, Failed to send subscription message to: %d", err, id)
		}
	})
}

func (h *Handler) Subscribe(u *objs.Update) {
	err := h.ac.Subscribe(u.Message.Chat.Id)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An error occurred while subscribing. Please try again later.", 0)
		return
	}
	_, err = h.b.SendMessage(u.Message.Chat.Id, "You have been <b>Subscribed</b> to the daily prayer notifications. 🔔", "HTML", 0, false, false)
	if err != nil {
		log.Printf("Error: %s, Failed to send subscribe message", err)
	}
}

func (h *Handler) Unsubscribe(u *objs.Update) {
	err := h.ac.Unsubscribe(u.Message.Chat.Id)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An error occurred while unsubscribing. Please try again later.", 0)
		return
	}
	_, err = h.b.SendMessage(u.Message.Chat.Id, "You have been <b>Unsubscribed</b> from the daily prayer notifications. 🔕", "HTML", 0, false, false)
	if err != nil {
		log.Printf("Error: %s, Failed to send unsubscribe message", err)
	}
}
