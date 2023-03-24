package handler

import (
	"log"

	objs "github.com/SakoDroid/telego/objects"
	"github.com/escalopa/gopray/pkg/language"
)

// scriptWrapper is a wrapper for user commands to load user script if not loaded
func (h *Handler) scriptWrapper(command func(u *objs.Update)) func(u *objs.Update) {
	return func(u *objs.Update) {
		err := h.setScript(u.Message.Chat.Id)
		if err != nil {
			log.Printf("failed to set script on scriptWrapper: %v", err)
			h.simpleSend(u.Message.Chat.Id, "unexpected error, Use /bug to report the error if it remains", 0)
			return
		}
		command(u)
	}

}

func (h *Handler) setScript(chatID int) error {
	sc, ok := h.userScript[chatID]
	if !ok || sc == nil {
		lang, err := h.u.GetLang(h.c, chatID)
		if err != nil {
			log.Printf("failed to get lang on scriptWrapper: %v", err)
			lang = language.DefaultLang().Short
		}
		// Load user script
		script, err := h.u.GetScript(h.c, lang)
		if err != nil {
			log.Printf("failed to get script on scriptWrapper: %v", err)
			return err
		}
		h.userScript[chatID] = script
	}
	return nil
}
