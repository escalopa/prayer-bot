package handler

import (
	"fmt"
	"log"
	"strconv"

	objs "github.com/SakoDroid/telego/objects"
)

func (h *Handler) GetPrayers(u *objs.Update) {
	prayers, err := h.ac.GetPrayers()
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An unexpected error occurred while getting prayers. Please try again later.", 0)
		return
	}

	log.Println("User ID", u.Message.Chat.Id)
	message := fmt.Sprintf("---\n%s\n---", prayers.EnHTML())
	_, err = h.b.SendMessage(u.Message.Chat.Id, message, "HTML", 0, false, false)
	if err != nil {
		log.Println(err)
	}
}

func (h *Handler) GetPrayersByDate(u *objs.Update) {
	chatID := strconv.Itoa(u.Message.Chat.Id)
	ch, err := h.b.AdvancedMode().RegisterChannel(chatID, "message")
	defer h.b.AdvancedMode().UnRegisterChannel(chatID, "message")
	if err != nil {
		return
	}

	h.b.SendMessage(u.Message.Chat.Id, "Please send the date in the format of <b><u>DD/MM</u></b> without leading zeros", "HTML", 0, false, false)
	u = <-*ch
	date := u.Message.Text

	// TODO: Add validation for date format using regex
	prayers, err := h.ac.GetPrayersByDate(date)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An unexpected error occurred while getting prayers. Please try again later.", 0)
		return
	}

	log.Println("User ID", u.Message.Chat.Id)
	message := fmt.Sprintf("---\n%s\n---", prayers.EnHTML())
	_, err = h.b.SendMessage(u.Message.Chat.Id, message, "HTML", 0, false, false)
	if err != nil {
		log.Println(err)
	}
}
