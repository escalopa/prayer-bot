package handler

import (
	"context"
	"log"

	bt "github.com/SakoDroid/telego"
	objs "github.com/SakoDroid/telego/objects"
	gpe "github.com/escalopa/gopray/pkg/error"
	"github.com/escalopa/gopray/telegram/internal/application"
)

type Handler struct {
	b  *bt.Bot
	ac *application.UseCase
	c  context.Context
}

func New(b *bt.Bot, ac *application.UseCase, ctx context.Context) *Handler {
	return &Handler{
		b:  b,
		ac: ac,
		c:  ctx,
	}
}

func (h *Handler) Register() {
	var err error
	err = h.b.AddHandler("/help", h.Help, "all")
	gpe.CheckError(err)
	err = h.b.AddHandler("/subscribe", h.Subscribe, "all")
	gpe.CheckError(err)
	err = h.b.AddHandler("/unsubscribe", h.Unsubscribe, "all")
	gpe.CheckError(err)
	err = h.b.AddHandler("/prayers", h.GetPrayers, "all")
	gpe.CheckError(err)
	err = h.b.AddHandler("/prayersdate", h.Getprayersdate, "all")
	gpe.CheckError(err)
	err = h.b.AddHandler("/lang", h.SetLang, "all")
	gpe.CheckError(err)
	err = h.b.AddHandler("/feedback", h.Feedback, "all")
	gpe.CheckError(err)
	err = h.b.AddHandler("/bug", h.Bug, "all")
	gpe.CheckError(err)
}

func (h *Handler) Help(u *objs.Update) {
	_, err := h.b.SendMessage(u.Message.Chat.Id, `
	Asalamu alaykum, I am kazan prayer's time bot, I can help you know prayer's time anytime to always pray on time 🙏.

	Available commands are below: 👇

	<b>Prayers</b>
	/prayers - Get prayer's time for today ⏰
	/prayersdate - Get prayer's time for a specific date 📅
	/subscribe - Subscribe to daily prayers notification 🔔
	/unsubscribe - Unsubscribe from daily prayers notification 🔕

	<b>Support</b>
	/help - Show this message 📖
	/lang - Set bot language  🌐
	/feedback - Send feedback or idea to the bot developers 📩
	/bug - Report a bug to the bot developers 🐞
	`, "HTML", 0, false, false)
	if err != nil {
		log.Println(err)
	}
}

// SimpleSend sends a simple message
func (bh *Handler) simpleSend(chatID int, text string, replyTo int) {
	_, err := bh.b.SendMessage(chatID, text, "", replyTo, false, false)
	if err != nil {
		log.Println(err)
	}
}
