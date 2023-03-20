package handler

import (
	"context"
	"log"

	bt "github.com/SakoDroid/telego"
	gpe "github.com/escalopa/gopray/pkg/error"
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

func (h *Handler) Start() {
	h.register()
	h.setupBundler()
	go h.NotifySubscribers() // Notify subscriber about the prayer times.
	log.Println("Bot started.")
}

func (h *Handler) register() {
	var err error
	err = h.b.AddHandler("/help", h.Help, "all")
	gpe.CheckError(err)
	err = h.b.AddHandler("/subscribe", h.Subscribe, "all")
	gpe.CheckError(err)
	err = h.b.AddHandler("/unsubscribe", h.Unsubscribe, "all")
	gpe.CheckError(err)
	err = h.b.AddHandler("/today", h.GetPrayers, "all")
	gpe.CheckError(err)
	err = h.b.AddHandler("/date", h.Getprayersdate, "all")
	gpe.CheckError(err)
	err = h.b.AddHandler("/lang", h.SetLang, "all")
	gpe.CheckError(err)
	err = h.b.AddHandler("/feedback", h.Feedback, "all")
	gpe.CheckError(err)
	err = h.b.AddHandler("/bug", h.Bug, "all")
	gpe.CheckError(err)
	err = h.b.AddHandler("/respond", h.Respond, "all")
	gpe.CheckError(err)
}

// TODO: Implement bundler for multi language support.
func (h *Handler) setupBundler() {}

// SimpleSend sends a simple message
func (h *Handler) simpleSend(chatID int, text string, replyTo int) {
	_, err := h.b.SendMessage(chatID, text, "", replyTo, false, false)
	if err != nil {
		log.Printf("Error: %s, Failed to simpleSend", err)
	}
}

func (h *Handler) cancelOperation(message, response string, chatID int) bool {
	if message == "/cancel" {
		h.simpleSend(chatID, response, 0)
		return true
	}
	return false
}
