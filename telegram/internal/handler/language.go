package handler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/escalopa/gopray/telegram/internal/domain"

	objs "github.com/SakoDroid/telego/objects"
)

func (h *Handler) SetLang(u *objs.Update) {
	var messageID int

	chatID := getChatID(u)
	script := h.getChatScript(chatID)
	kb := h.bot.CreateInlineKeyboard()

	ctx, cancel := context.WithTimeout(h.getChatCtx(chatID), 1*time.Minute)
	// Deletes the message after the button is pressed or after 1 hour.
	go func() {
		defer cancel()
		<-ctx.Done()
		h.deleteMessage(chatID, messageID)
	}()

	for i, lang := range domain.AvailableLanguages() {
		row := i/2 + 1 // 2 buttons per row.
		kb.AddCallbackButtonHandler(lang.Long, lang.Short, row, h.setLangKeyboardCallback(ctx, cancel, chatID))
	}

	// Sends the message along with the keyboard.
	r, err := h.bot.AdvancedMode().ASendMessage(
		chatID,
		script.LanguageSelectionStart,
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

func (h *Handler) setLangKeyboardCallback(ctx context.Context, cancel context.CancelFunc, chatID int) func(update *objs.Update) {
	return func(u *objs.Update) {
		defer cancel()

		script := h.getChatScript(chatID)
		selectedLanguage := u.CallbackQuery.Data

		var err error
		defer func() {
			if err == nil {
				return
			}

			_, err = h.bot.AdvancedMode().AAnswerCallbackQuery(
				u.CallbackQuery.Id,
				fmt.Sprintf(script.LanguageSelectionFail, selectedLanguage),
				true, "", 0)
			if err != nil {
				log.Printf("failed to send callback query on /lang: %s", err)
			}
		}()

		// Sets the lang
		err = h.uc.SetLang(ctx, chatID, selectedLanguage)
		if err != nil {
			log.Printf("failed to set lang to %s: %v", selectedLanguage, err)
			return
		}

		// Get script for chatID
		script, err = h.uc.GetScript(ctx, selectedLanguage)
		if err != nil {
			log.Printf("failed to get script for %s: %v", selectedLanguage, err)
			return
		}

		// Update chatID script
		h.setChatScript(chatID, script)
		_, err = h.bot.SendMessage(
			chatID,
			fmt.Sprintf(script.LanguageSelectionSuccess, selectedLanguage),
			"HTML",
			0, false, false)
		if err != nil {
			log.Printf("failed to send message on /lang: %s", err)
		}
	}
}
