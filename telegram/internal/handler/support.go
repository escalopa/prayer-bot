package handler

import (
	"fmt"
	"log"
	"strconv"

	objs "github.com/SakoDroid/telego/objects"
)

const (
	botOwnerID = 1385434843
)

func (h *Handler) Feedback(u *objs.Update) {
	chatID := strconv.Itoa(u.Message.Chat.Id)
	ch, err := h.b.AdvancedMode().RegisterChannel(chatID, "message")
	defer h.b.AdvancedMode().UnRegisterChannel(chatID, "message")
	if err != nil {
		return
	}

	h.simpleSend(u.Message.Chat.Id, "Please send your feedback as text message", 0)
	u = <-*ch
	text := u.Message.Text

	message := fmt.Sprintf(`
	Feedback Message... 💬
	
	<b>ID:</b> %d
	<b>Username:</b> %s
	<b>Full Name:</b> %s %s
	<b>Feedback:</b> %s
	`, u.Message.Chat.Id, u.Message.Chat.Username, u.Message.Chat.FirstName, u.Message.Chat.LastName, text)
	_, err = h.b.SendMessage(botOwnerID, message, "HTML", 0, false, false)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An unexpected error occurred while sending your feedback. Please try again later.", 0)
		log.Println(err)
		return
	}

	h.simpleSend(u.Message.Chat.Id, "Thank you for your feedback! 🙏", 0)

}

func (h *Handler) Bug(u *objs.Update) {
	chatID := strconv.Itoa(u.Message.Chat.Id)
	ch, err := h.b.AdvancedMode().RegisterChannel(chatID, "message")
	defer h.b.AdvancedMode().UnRegisterChannel(chatID, "message")
	if err != nil {
		return
	}

	h.simpleSend(u.Message.Chat.Id, "Please send your bug report as text message", 0)
	u = <-*ch
	text := u.Message.Text

	message := fmt.Sprintf(`
	Bug Report... 🐞🐛
	
	<b>ID:</b> %d
	<b>Username:</b> %s
	<b>Full Name:</b> %s %s
	<b>Bug Report:</b> %s
	`, u.Message.Chat.Id, u.Message.Chat.Username, u.Message.Chat.FirstName, u.Message.Chat.LastName, text)
	_, err = h.b.SendMessage(botOwnerID, message, "HTML", 0, false, false)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An unexpected error occurred while sending your bug report. Please try again later.", 0)
		log.Println(err)
		return
	}

	h.simpleSend(u.Message.Chat.Id, "Thank you for your bug report!, We will fix it 🛠️ ASAP.", 0)

}
