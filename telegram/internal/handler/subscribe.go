package handler

import (
	"fmt"
	"log"

	objs "github.com/SakoDroid/telego/objects"
)

func (h *Handler) notifySubscribers() {
	h.u.Notify(
		func(id int, prayer, time string) {
			// notifySoon
			_, err := h.b.SendMessage(id, fmt.Sprintf("<b>%s</b> prayer starts in <b>%s</b> minutes.", prayer, time), "HTML", 0, false, false)
			if err != nil {
				log.Printf("failed to send message, Error: %s", err)
			}
		},
		func(id int, prayer string) {
			// notifyNow
			_, err := h.b.SendMessage(id, fmt.Sprintf("<b>%s</b> prayer time has arrived.", prayer), "HTML", 0, false, false)
			if err != nil {
				log.Printf("failed to send message, Error: %s", err)
			}
		},
		func(id int, time string) {
			// notifyGomaa
			message := fmt.Sprintf(
				"Assalamu Alaikum 👋!\nDon't forget today is <b>Gomaa</b>,make sure to attend prayers at the mosque! 🕌, Gomma today is at <b>%s</b>", time)
			_, err := h.b.SendMessage(id, message, "HTML", 0, false, false)
			if err != nil {
				log.Printf("failed to send message, Error: %s", err)
			}
		},
	)
}

func (h *Handler) Subscribe(u *objs.Update) {
	err := h.u.Subscribe(h.c, u.Message.Chat.Id)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An error occurred while subscribing. Please try again later.", 0)
		return
	}
	_, err = h.b.SendMessage(u.Message.Chat.Id, "You have been <b>Subscribed</b> to the daily prayers notifications. 🔔", "HTML", 0, false, false)
	if err != nil {
		log.Printf("Error: %s, Failed to send subscribe message", err)
	}
}

func (h *Handler) Unsubscribe(u *objs.Update) {
	err := h.u.Unsubscribe(h.c, u.Message.Chat.Id)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An error occurred while unsubscribing. Please try again later.", 0)
		return
	}
	_, err = h.b.SendMessage(u.Message.Chat.Id, "You have been <b>Unsubscribed</b> from the daily prayers notifications. 🔕", "HTML", 0, false, false)
	if err != nil {
		log.Printf("Error: %s, Failed to send unsubscribe message", err)
	}
}
