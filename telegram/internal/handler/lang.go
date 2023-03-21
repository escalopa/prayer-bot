package handler

import (
	"context"
	"fmt"
	"log"
	"time"

	objs "github.com/SakoDroid/telego/objects"
	"github.com/escalopa/gopray/pkg/core"
)

func (h *Handler) SetLang(u *objs.Update) {
	var messageID int
	chatID := u.Message.Chat.Id
	kb := h.b.CreateInlineKeyboard()

	ctx, cancel := context.WithTimeout(h.c, 1*time.Hour)
	pressed := make(chan struct{})
	// Deletes the message after the button is pressed or after 1 hour.
	go func() {
		defer cancel()
		select {
		case <-ctx.Done():
		case <-pressed:
		}
		h.deleteMessage(chatID, messageID)
	}()

	for i, language := range core.AvaliableLanguages() {
		//Adds a callback button with handler.
		row := i/2 + 1 // 2 buttons per row.
		kb.AddCallbackButtonHandler(language, language, row, func(u *objs.Update) {
			defer func() { pressed <- struct{}{} }()
			// Sets the language.
			err := h.u.SetLang(h.c, chatID, u.CallbackQuery.Data)
			callBackMessage := "Successfully set language to %s"
			if err != nil {
				log.Println(err)
				callBackMessage = "Failed to set language to %"
			}
			// Sends the callback message.
			_, err = h.b.AdvancedMode().AAnswerCallbackQuery(u.CallbackQuery.Id,
				fmt.Sprintf(callBackMessage, u.CallbackQuery.Data),
				true, "", 0)
			if err != nil {
				log.Println(err)
			}
		})
	}

	// Sends the message along with the keyboard.
	r, err := h.b.AdvancedMode().ASendMessage(u.Message.Chat.Id, "Choose language", "", u.Message.MessageId, false, false, nil, false, false, kb)
	if err != nil {
		log.Println(err)
	}
	messageID = r.Result.MessageId
}
