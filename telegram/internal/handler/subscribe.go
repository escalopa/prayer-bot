package handler

import (
	"fmt"
	"log"

	objs "github.com/SakoDroid/telego/objects"
)

func (h *Handler) notifySubscribers() {
	// replace is a helper function to delete the last message id and store the new one's id in the database.
	replace := func(id int, message int) {
		// Delete the last message id.
		if err := h.c.Err(); err != nil {
			return
		}
		lastMessageId, err := h.u.GetPrayerMessageID(h.c, id)
		if err != nil {
			log.Printf("failed to remove last message id /notify, Error: %s", err)
		} else {
			h.deleteMessage(id, lastMessageId)
		}
		// Store the new message id.
		if err = h.c.Err(); err != nil {
			return
		}
		err = h.u.StorePrayerMessageID(h.c, id, message)
		if err != nil {
			log.Printf("failed to replace message id /notify, Error: %s", err)
		}
	}

	h.u.Notify(
		// notifySoon
		func(id int, prayer, time string) {
			if err := h.setScript(id); err != nil {
				log.Printf("failed to send notify message, user language not set on prayer soon /notify: %v", err)
				return
			}
			message := fmt.Sprintf(h.userScript[id].PrayerSoon, h.userScript[id].GetPrayerByName(prayer), time)
			r, err := h.b.SendMessage(id, message, "HTML", 0, false, false)
			if err != nil {
				log.Printf("failed to send message on notifySoon, Error: %s", err)
				return
			}
			go replace(id, r.Result.MessageId)
		},

		// notifyNow
		func(id int, prayer string) {
			if err := h.setScript(id); err != nil {
				log.Printf("failed to send notify message, user language not set on prayer arrived /notify: %v", err)
				return
			}
			message := fmt.Sprintf(h.userScript[id].PrayerArrived, h.userScript[id].GetPrayerByName(prayer))
			r, err := h.b.SendMessage(id, message, "HTML", 0, false, false)
			if err != nil {
				log.Printf("failed to send message on notifyNow, Error: %s", err)
				return
			}
			go replace(id, r.Result.MessageId)
		},

		// notifyGomaa
		func(id int, time string) {
			if err := h.setScript(id); err != nil {
				log.Printf("failed to send notify message, user language not set on gomaa /notify: %v", err)
				return
			}
			message := fmt.Sprintf(h.userScript[id].GomaaDay, time)
			r, err := h.b.SendMessage(id, message, "HTML", 0, false, false)
			if err != nil {
				log.Printf("failed to send message on notifyGomaa, Error: %s", err)
				return
			}
			go replace(id, r.Result.MessageId)
		},
	)
}

func (h *Handler) Subscribe(u *objs.Update) {
	chatID := u.Message.Chat.Id
	err := h.u.Subscribe(h.c, chatID)
	if err != nil {
		h.simpleSend(chatID, h.userScript[chatID].SubscriptionError, 0)
		return
	}
	_, err = h.b.SendMessage(chatID, h.userScript[chatID].SubscriptionSuccess, "HTML", 0, false, false)
	if err != nil {
		log.Printf("failed to send subscribe message, Error: %s", err)
		return
	}
}

func (h *Handler) Unsubscribe(u *objs.Update) {
	chatID := u.Message.Chat.Id
	err := h.u.Unsubscribe(h.c, chatID)
	if err != nil {
		h.simpleSend(chatID, h.userScript[chatID].UnsubscriptionError, 0)
		return
	}
	_, err = h.b.SendMessage(chatID, h.userScript[chatID].UnsubscriptionSuccess, "HTML", 0, false, false)
	if err != nil {
		log.Printf("failed to send unsubscribe message, Error: %s", err)
		return
	}
}
