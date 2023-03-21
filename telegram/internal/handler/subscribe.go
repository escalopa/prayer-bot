package handler

import (
	"context"
	"fmt"
	"log"

	objs "github.com/SakoDroid/telego/objects"
)

func (h *Handler) notifySubscribers() {
	// store stores the new message id in the database.
	// remove deletes the last message id from the database.
	// I use these functions to avoid code duplication for the three notifications.
	store := func(ctx context.Context, id int, message int, storeFunc func(ctx context.Context, id int, message int) error) {
		err := storeFunc(ctx, id, message)
		if err != nil {
			log.Printf("failed to store message id, Error: %s", err)
		}
	}
	remove := func(ctx context.Context, id int, removeFunc func(ctx context.Context, id int) (int, error)) {
		lastMessageId, err := removeFunc(ctx, id)
		if err != nil {
			log.Printf("failed to remove last message id, Error: %s", err)
		} else {
			h.deleteMessage(id, lastMessageId)
		}
	}

	h.u.Notify(
		func(id int, prayer, time string) {
			// notifySoon
			remove(h.c, id, h.u.GetPrayerMessageID)
			r, err := h.b.SendMessage(id, fmt.Sprintf("<b>%s</b> prayer starts in <b>%s</b> minutes.", prayer, time), "HTML", 0, false, false)
			if err != nil {
				log.Printf("failed to send message, Error: %s", err)
			}
			store(h.c, id, r.Result.MessageId, h.u.StorePrayerMessageID)
		},
		func(id int, prayer string) {
			// notifyNow
			remove(h.c, id, h.u.GetPrayerMessageID)
			r, err := h.b.SendMessage(id, fmt.Sprintf("<b>%s</b> prayer time has arrived.", prayer), "HTML", 0, false, false)
			if err != nil {
				log.Printf("failed to send message, Error: %s", err)
			}
			store(h.c, id, r.Result.MessageId, h.u.StorePrayerMessageID)
		},
		func(id int, time string) {
			// notifyGomaa
			remove(h.c, id, h.u.GetGomaaMessageID)
			message := fmt.Sprintf(
				"Assalamu Alaikum 👋!\nDon't forget today is <b>Gomaa</b>,make sure to attend prayers at the mosque! 🕌, Gomma today is at <b>%s</b>", time)
			r, err := h.b.SendMessage(id, message, "HTML", 0, false, false)
			if err != nil {
				log.Printf("failed to send message, Error: %s", err)
			}
			store(h.c, id, r.Result.MessageId, h.u.StoreGomaaMessageID)
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
