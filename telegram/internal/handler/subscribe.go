package handler

import (
	"fmt"
	"log"

	objs "github.com/SakoDroid/telego/objects"
)

func (h *Handler) notifySubscribers() {
	// store stores the new message id in the database.
	// remove deletes the last message id from the database.
	// I use these functions to avoid code duplication for the three notifications.
	store := func(id int, message int) {
		err := h.u.StorePrayerMessageID(h.c, id, message)
		if err != nil {
			log.Printf("failed to store message id /notify, Error: %s", err)
		}
	}
	remove := func(id int) {
		lastMessageId, err := h.u.GetPrayerMessageID(h.c, id)
		if err != nil {
			log.Printf("failed to remove last message id /notify, Error: %s", err)
		} else {
			h.deleteMessage(id, lastMessageId)
		}
	}

	h.u.Notify(
		func(id int, prayer, time string) {
			// notifySoon
			go remove(id)
			r, err := h.b.SendMessage(id, fmt.Sprintf("<b>%s</b> prayer starts in <b>%s</b> minutes.", prayer, time), "HTML", 0, false, false)
			if err != nil {
				log.Printf("failed to send message on notifySoon, Error: %s", err)
			}
			go store(id, r.Result.MessageId)
		},
		func(id int, prayer string) {
			// notifyNow
			go remove(id)
			r, err := h.b.SendMessage(id, fmt.Sprintf("<b>%s</b> prayer time has arrived.", prayer), "HTML", 0, false, false)
			if err != nil {
				log.Printf("failed to send message on notifyNow, Error: %s", err)
			}
			go store(id, r.Result.MessageId)
		},
		func(id int, time string) {
			// notifyGomaa
			go remove(id)
			message := fmt.Sprintf(
				"Assalamu Alaikum 👋!\nDon't forget today is <b>Gomaa</b>,make sure to attend prayers at the mosque! 🕌, Gomma today is at <b>%s</b>", time)
			r, err := h.b.SendMessage(id, message, "HTML", 0, false, false)
			if err != nil {
				log.Printf("failed to send message on notifyGomaa, Error: %s", err)
			}
			go store(id, r.Result.MessageId)
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
		log.Printf("failed to send subscribe message, Error: %s", err)
		return
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
		log.Printf("failed to send unsubscribe message, Error: %s", err)
		return
	}
}
