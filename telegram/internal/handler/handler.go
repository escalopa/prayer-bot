package handler

import (
	"context"
	"log"

	bt "github.com/SakoDroid/telego"
	"github.com/escalopa/gopray/telegram/internal/application"
)

type Handler struct {
	c context.Context
	b *bt.Bot
	u *application.UseCase
}

func New(ctx context.Context, b *bt.Bot, u *application.UseCase) *Handler {
	return &Handler{
		b: b,
		u: u,
		c: ctx,
	}
}

func (h *Handler) Start() error {
	err := h.register()
	if err != nil {
		return err
	}
	h.setupBundler()
	go h.notifySubscribers() // Notify subscriber about the prayer times.
	return nil
}

func (h *Handler) register() error {
	var err error
	err = h.b.AddHandler("/help", h.Help, "all")
	if err != nil {
		return err
	}
	err = h.b.AddHandler("/subscribe", h.Subscribe, "all")
	if err != nil {
		return err
	}
	err = h.b.AddHandler("/unsubscribe", h.Unsubscribe, "all")
	if err != nil {
		return err
	}
	err = h.b.AddHandler("/today", h.GetPrayers, "all")
	if err != nil {
		return err
	}
	err = h.b.AddHandler("/date", h.GetPrayersByDate, "all")
	if err != nil {
		return err
	}
	err = h.b.AddHandler("/lang", h.SetLang, "all")
	if err != nil {
		return err
	}
	err = h.b.AddHandler("/feedback", h.Feedback, "all")
	if err != nil {
		return err
	}
	err = h.b.AddHandler("/bug", h.Bug, "all")
	if err != nil {
		return err
	}

	//////////////////////////
	///// Admin Commands /////
	//////////////////////////

	err = h.b.AddHandler("/respond", h.Respond, "all")
	if err != nil {
		return err
	}
	err = h.b.AddHandler("/subs", h.GetSubscribers, "all")
	if err != nil {
		return err
	}
	err = h.b.AddHandler("/sall", h.SendAll, "all")
	if err != nil {
		return err
	}
	return nil
}

// TODO: Implement bundler for multi language support.
func (h *Handler) setupBundler() {}

// simpleSend sends a simple message to the chat with the given chatID & text and replyTo.
func (h *Handler) simpleSend(chatID int, text string, replyTo int) (messageID int) {
	r, err := h.b.SendMessage(chatID, text, "", replyTo, false, false)
	if err != nil {
		log.Printf("Error: %s, Failed to simpleSend", err)
		return 0
	}
	return r.Result.MessageId
}

// cancelOperation checks if the message is /cancel and sends a response.
// Returns true if the message is /cancel.
func (h *Handler) cancelOperation(message, response string, chatID int) bool {
	if message == "/cancel" {
		h.simpleSend(chatID, response, 0)
		return true
	}
	return false
}

func (h *Handler) deleteMessage(chatID, messageID int) {
	editor := h.b.GetMsgEditor(chatID)
	_, err := editor.DeleteMessage(messageID)
	if err != nil {
		log.Printf("Error: %s, Failed to delete message", err)
	}
}
