package handler

import (
	"context"
	"sync"

	objs "github.com/SakoDroid/telego/objects"

	"github.com/SakoDroid/telego"
	app "github.com/escalopa/gopray/telegram/internal/application"
	"github.com/escalopa/gopray/telegram/internal/domain"
)

type Handler struct {
	ctx context.Context
	bot *telego.Bot
	uc  *app.UseCase

	botOwner int // Bot owner's ID.

	internal struct {
		chatCtx   map[int]userContext // chatID => latest chat context
		chatCtxMu sync.RWMutex

		chatScript   map[int]*domain.Script // chatID => scripts for this chat.
		chatScriptMu sync.RWMutex
	}
}

func New(bot *telego.Bot, ownerID int, uc *app.UseCase) *Handler {
	h := &Handler{
		bot: bot,
		uc:  uc,

		botOwner: ownerID,
	}
	h.internal.chatCtx = make(map[int]userContext)
	h.internal.chatScript = make(map[int]*domain.Script)
	return h
}

func (h *Handler) Run(ctx context.Context) error {
	h.ctx = ctx

	if err := h.register(); err != nil {
		return err
	}
	if err := h.bot.Run(); err != nil {
		return err
	}

	go h.processUnknown()
	go h.uc.SchedulePrayers(&notifier{h})

	return nil
}

func (h *Handler) register() error {
	registers := []error{
		//////////////////////////
		///// User Commands //////
		//////////////////////////

		h.bot.AddHandler(cmdStart, h.useContext(h.useScript(h.Start)), "all"),
		h.bot.AddHandler(cmdHelp, h.useContext(h.useScript(h.Help)), "all"),
		h.bot.AddHandler(cmdSubscribe, h.useContext(h.useScript(h.Subscribe)), "all"),
		h.bot.AddHandler(cmdUnsubscribe, h.useContext(h.useScript(h.Unsubscribe)), "all"),
		h.bot.AddHandler(cmdToday, h.useContext(h.useScript(h.GetPrayers)), "all"),
		h.bot.AddHandler(cmdDate, h.useContext(h.useScript(h.GetPrayersByDate)), "all"),
		h.bot.AddHandler(cmdLang, h.useContext(h.useScript(h.SetLang)), "all"),
		h.bot.AddHandler(cmdFeedback, h.useContext(h.useScript(h.Feedback)), "all"),
		h.bot.AddHandler(cmdBug, h.useContext(h.useScript(h.Bug)), "all"),

		//////////////////////////
		///// Admin Commands /////
		//////////////////////////

		h.bot.AddHandler(cmdRespond, h.useContext(h.useScript(h.useAdmin(h.Respond))), "all"),
		h.bot.AddHandler(cmdGetSubscribe, h.useContext(h.useScript(h.useAdmin(h.GetSubscribers))), "all"),
		h.bot.AddHandler(cmdSendAll, h.useContext(h.useScript(h.useAdmin(h.SendAll))), "all"),
	}

	for _, err := range registers {
		if err != nil {
			return err
		}
	}

	return nil
}

func (h *Handler) processUnknown() {
	for u := range *h.bot.GetUpdateChannel() {
		if u.Message == nil {
			continue
		}

		chatID := getChatID(u)

		h.deleteMessage(chatID, u.Message.MessageId)
		h.simpleSend(chatID, cmdHelp)
	}
}

// readInput reads the next input from the input channel or returns nil if the context is done.
func (h *Handler) readInput(ctx context.Context, inputChan <-chan *objs.Update) *objs.Update {
	ctx, cancel := context.WithTimeout(ctx, inputTimeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return nil
	case u := <-inputChan:
		return u
	}
}

func (h *Handler) Stop() {
	h.bot.Stop()
	h.uc.Close()
}
