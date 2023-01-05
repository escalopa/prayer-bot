package handler

import (
	"context"
	"log"

	bt "github.com/SakoDroid/telego"
	objs "github.com/SakoDroid/telego/objects"
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
	// h.b.AddHandler("/start", h.Start, "all")
	h.b.AddHandler("/help", h.Help, "all")
	h.b.AddHandler("/subscribe", h.Subscribe, "all")
	h.b.AddHandler("/unsubscribe", h.Unsubscribe, "all")
	h.b.AddHandler("/prayers", h.GetPrayers, "all")
	h.b.AddHandler("/prayersByDate", h.GetPrayersByDate, "all")
	h.b.AddHandler("/lang", h.SetLang, "all")
	h.b.AddHandler("/feedback", h.Feedback, "all")
	h.b.AddHandler("/bug", h.Bug, "all")
}

func (h *Handler) Help(u *objs.Update) {
	h.simpleSend(u.Message.Chat.Id, `
	Asalamu alaykum, I am a prayers time bot that sends you daily prayers times 🙏 to always pray on time.
	
	Available commands are below: 👇	
	/help - Show this message 📖   
	/prayers - Get prayers for today ⏰
	/prayersByDate - Get prayers for a specific date 📅
	/subscribe - Subscribe to daily prayers notification 🔔
	/unsubscribe - Unsubscribe from daily prayers notification 🔕
	/lang - Set bot language  🌐
	/feedback - Send feedback to the bot developers 📩
	/bug - Report a bug to the bot developers 🐞

	`, 0)
}

// SimpleSend sends a simple message
func (bh *Handler) simpleSend(chatID int, text string, replyTo int) {
	_, err := bh.b.SendMessage(chatID, text, "", replyTo, false, false)
	if err != nil {
		log.Println(err)
	}
}
