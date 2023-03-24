package handler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/escalopa/gopray/pkg/language"

	objs "github.com/SakoDroid/telego/objects"
)

func (h *Handler) SetLang(u *objs.Update) {
	var messageID int
	chatID := u.Message.Chat.Id
	kb := h.b.CreateInlineKeyboard()

	ctx, cancel := context.WithTimeout(h.userCtx[u.Message.Chat.Id].ctx, 1*time.Minute)
	// Deletes the message after the button is pressed or after 1 hour.
	go func() {
		defer cancel()
		<-ctx.Done()
		h.deleteMessage(chatID, messageID)

	}()

	for i, lang := range language.AvaliableLanguages() {
		//Adds a callback button with handler.
		row := i/2 + 1 // 2 buttons per row.
		kb.AddCallbackButtonHandler(lang.Long, lang.Short, row, func(u *objs.Update) {
			defer cancel()
			// Sets the lang.
			err := h.u.SetLang(h.c, chatID, u.CallbackQuery.Data)
			if err != nil {
				log.Printf("failed to set lang to %s: %v", u.CallbackQuery.Data, err)
				_, err = h.b.AdvancedMode().AAnswerCallbackQuery(u.CallbackQuery.Id,
					fmt.Sprintf(h.userScript[chatID].LanguageSelectionFail, u.CallbackQuery.Data),
					true, "", 0)
				if err != nil {
					log.Printf("failed to send callback query on /lang: %s", err)
				}
				return
			}
			// Get the script for the user.
			script, err := h.u.GetScript(h.c, u.CallbackQuery.Data)
			if err != nil {
				log.Printf("failed to get script for %s: %v", u.CallbackQuery.Data, err)
				_, err = h.b.AdvancedMode().AAnswerCallbackQuery(u.CallbackQuery.Id,
					fmt.Sprintf(h.userScript[chatID].LanguageSelectionFail, u.CallbackQuery.Data),
					true, "", 0)
				if err != nil {
					log.Printf("failed to send callback query on /lang: %s", err)
				}
				return
			}
			// Update the user script.
			h.userScript[chatID] = script
			_, err = h.b.SendMessage(chatID, fmt.Sprintf(h.userScript[chatID].LanguageSelectionSuccess, u.CallbackQuery.Data), "HTML", 0, false, false)
			if err != nil {
				log.Printf("failed to send message on /lang: %s", err)
			}
		})
	}

	// Sends the message along with the keyboard.
	r, err := h.b.AdvancedMode().ASendMessage(
		chatID,
		h.userScript[chatID].LanguageSelectionStart,
		"",
		u.Message.MessageId,
		false,
		false,
		nil,
		false,
		false,
		kb,
	)
	if err != nil {
		log.Printf("failed to send message on /lang: %s", err)
	}
	messageID = r.Result.MessageId
}
