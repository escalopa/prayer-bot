package handler

import (
	"context"
	"sync"

	"github.com/SakoDroid/telego"
	app "github.com/escalopa/gopray/telegram/internal/application"
	"github.com/escalopa/gopray/telegram/internal/domain"
)

type Handler struct {
	ctx context.Context
	bot *telego.Bot
	uc  *app.UseCase

	botOwner int // Bot owner's ID.

	chatCtx   map[int]userContext // chatID => latest chat context
	chatCtxMu sync.RWMutex

	chatScript   map[int]*domain.Script // chatID => scripts for this chat.
	chatScriptMu sync.RWMutex
}

func New(bot *telego.Bot, ownerID int, uc *app.UseCase) *Handler {
	return &Handler{
		bot: bot,
		uc:  uc,

		botOwner: ownerID,

		chatCtx:    make(map[int]userContext),
		chatScript: make(map[int]*domain.Script),
	}
}

func (h *Handler) Run(ctx context.Context) error {
	h.ctx = ctx

	if err := h.register(); err != nil {
		return err
	}

	go h.uc.SchedulePrayers(&notifier{h})
	return nil
}

func (h *Handler) register() error {
	registers := []error{
		//////////////////////////
		///// User Commands //////
		//////////////////////////

		h.bot.AddHandler("/start", h.useContext(h.useScript(h.Start)), "all"),
		h.bot.AddHandler("/help", h.useContext(h.useScript(h.Help)), "all"),
		h.bot.AddHandler("/subscribe", h.useContext(h.useScript(h.Subscribe)), "all"),
		h.bot.AddHandler("/unsubscribe", h.useContext(h.useScript(h.Unsubscribe)), "all"),
		h.bot.AddHandler("/today", h.useContext(h.useScript(h.GetPrayers)), "all"),
		h.bot.AddHandler("/date", h.useContext(h.useScript(h.GetPrayersByDate)), "all"),
		h.bot.AddHandler("/lang", h.useContext(h.useScript(h.SetLang)), "all"),
		h.bot.AddHandler("/feedback", h.useContext(h.useScript(h.Feedback)), "all"),
		h.bot.AddHandler("/bug", h.useContext(h.useScript(h.Bug)), "all"),

		//////////////////////////
		///// Admin Commands /////
		//////////////////////////

		h.bot.AddHandler("/respond", h.useContext(h.useScript(h.useAdmin(h.Respond))), "all"),
		h.bot.AddHandler("/subs", h.useContext(h.useScript(h.useAdmin(h.GetSubscribers))), "all"),
		h.bot.AddHandler("/sall", h.useContext(h.useScript(h.useAdmin(h.SendAll))), "all"),
	}

	for _, err := range registers {
		if err != nil {
			return err
		}
	}

	return nil
}

func (h *Handler) Stop() {
	h.bot.Stop()
	h.uc.Close()
}
