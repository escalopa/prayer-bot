package main

import (
	"context"

	"github.com/escalopa/gopray/telegram/internal/handler"

	bt "github.com/SakoDroid/telego"
	cfg "github.com/SakoDroid/telego/configs"
	"github.com/escalopa/gopray/pkg/config"

	gpc "github.com/escalopa/gopray/pkg/error"
	"github.com/escalopa/gopray/telegram/internal/adapters/notifier"
	"github.com/escalopa/gopray/telegram/internal/adapters/parser"
	"github.com/escalopa/gopray/telegram/internal/adapters/redis"
	"github.com/escalopa/gopray/telegram/internal/application"
)

func main() {

	c := config.NewConfig()

	bot, err := bt.NewBot(cfg.Default(c.Get("BOT_TOKEN")))
	gpc.CheckError(err)

	err = bot.Run()
	gpc.CheckError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up the database.
	r := redis.New(c.Get("CACHE_URL"))
	defer r.Close()
	pr := redis.NewPrayerRepository(r)
	sr := redis.NewSubscriberRepository(r)
	lr := redis.NewLanguageRepository(r)

	// Create schedule parser & parse the schedule.
	p := parser.New(c.Get("DATA_PATH"), pr)
	err = p.ParseSchedule()
	gpc.CheckError(err)

	n := notifier.New(pr, sr, lr)
	go n.Notify()

	a := application.New(n, pr, lr)
	run(bot, a, ctx)
}

func run(b *bt.Bot, a *application.UseCase, ctx context.Context) {

	//The general update channel.
	updateChannel := b.GetUpdateChannel()
	h := handler.New(b, a, ctx)
	h.Register()

	//Monitors any other update.
	for {
		update := <-*updateChannel
		if update.Message == nil {
			continue
		}
		h.Help(update)
	}
}
