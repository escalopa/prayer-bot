package handler

import (
	"fmt"

	objs "github.com/SakoDroid/telego/objects"
	log "github.com/sirupsen/logrus"
)

func (h *Handler) notifySubscribers() {
	h.u.Notify(
		// notifySoon
		func(id int, prayer, time string) {
			if err := h.setScript(id); err != nil {
				log.WithFields(log.Fields{"error": err}).Warn("failed to send notify message, user language not set on prayer soon /notify")
				return
			}
			message := fmt.Sprintf(h.userScript[id].PrayerSoon, h.userScript[id].GetPrayerByName(prayer), time)
			r, err := h.b.SendMessage(id, message, "HTML", 0, false, false)
			if err != nil {
				log.WithFields(log.Fields{"error": err}).Warn("failed to send message on notifySoon")
				return
			}
			go h.replace(id, r.Result.MessageId)
		},

		// notifyNow
		func(id int, prayer string) {
			if err := h.setScript(id); err != nil {
				log.WithFields(log.Fields{"error": err}).Warn("failed to send notify message, user language not set on prayer arrived /notify")
				return
			}
			message := fmt.Sprintf(h.userScript[id].PrayerArrived, h.userScript[id].GetPrayerByName(prayer))
			r, err := h.b.SendMessage(id, message, "HTML", 0, false, false)
			if err != nil {
				log.WithFields(log.Fields{"error": err}).Warn("failed to send message on notifyNow, Error")
				return
			}
			go h.replace(id, r.Result.MessageId)
		},

		// notifyGomaa
		func(id int, time string) {
			if err := h.setScript(id); err != nil {
				log.WithFields(log.Fields{"error": err}).Warn("failed to send notify message, user language not set on gomaa /notify")
				return
			}
			message := fmt.Sprintf(h.userScript[id].GomaaDay, time)
			r, err := h.b.SendMessage(id, message, "HTML", 0, false, false)
			if err != nil {
				log.WithFields(log.Fields{"error": err}).Warn("failed to send message on notifyGomaa")
				return
			}
			go h.replace(id, r.Result.MessageId)
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
		log.WithFields(log.Fields{"error": err}).Warn("failed to send subscribe message")
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
		log.WithFields(log.Fields{"error": err}).Warn("failed to send unsubscribe message")
		return
	}
}

// replace is a helper function to delete the last message id and store the new one's id in the database.
func (h *Handler) replace(id int, message int) {
	// Delete the last message id.
	if err := h.c.Err(); err != nil {
		return
	}
	lastMessageId, err := h.u.GetPrayerMessageID(h.c, id)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warn("failed to remove last message id /notify")
	} else {
		h.deleteMessage(id, lastMessageId)
	}
	// Store the new message id.
	if err = h.c.Err(); err != nil {
		return
	}
	err = h.u.StorePrayerMessageID(h.c, id, message)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warn("failed to replace message id /notify")
	}
}
