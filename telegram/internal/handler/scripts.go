package handler

import (
	"github.com/escalopa/gopray/telegram/internal/domain"
)

func (h *Handler) setScript(chatID int) error {
	sc, ok := h.chatScript[chatID]
	if !ok || sc == nil {
		// Get lang for chatID
		lang, err := h.uc.GetLang(h.ctx, chatID)
		if err != nil {
			lang = domain.DefaultLang().Short // Set default lang if failed to get lang
		}

		// Load script for lang
		script, err := h.uc.GetScript(h.ctx, lang)
		if err != nil {
			return err
		}

		h.setChatScript(chatID, script)
	}
	return nil
}
