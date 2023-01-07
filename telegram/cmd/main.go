package main

import (
	"context"
	"log"
	"strconv"

	"github.com/escalopa/gopray/telegram/internal/adapters/memory"
	"github.com/escalopa/gopray/telegram/internal/handler"

	bt "github.com/SakoDroid/telego"
	cfg "github.com/SakoDroid/telego/configs"
	"github.com/escalopa/gopray/pkg/config"

	gpe "github.com/escalopa/gopray/pkg/error"
	"github.com/escalopa/gopray/telegram/internal/adapters/notifier"
	"github.com/escalopa/gopray/telegram/internal/adapters/parser"
	"github.com/escalopa/gopray/telegram/internal/adapters/redis"
	"github.com/escalopa/gopray/telegram/internal/application"
)

func main() {

	c := config.NewConfig()

	// TODO: Add a logger.
	bot, err := bt.NewBot(cfg.Default(c.Get("BOT_TOKEN")))
	gpe.CheckError(err)

	err = bot.Run()
	gpe.CheckError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up the database.
	r := redis.New(c.Get("CACHE_URL"))
	defer r.Close()
	// pr := redis.NewPrayerRepository(r)
	pr := memory.NewPrayerRepository() // Use memory for prayer repository. To not hit the cache on every reload.
	sr := redis.NewSubscriberRepository(r)
	lr := redis.NewLanguageRepository(r)
	log.Println("Connected to Cache...")

	// Create schedule parser & parse the schedule.
	p := parser.New(c.Get("DATA_PATH"), pr)
	err = p.ParseSchedule()
	gpe.CheckError(err, "Error parsing schedule")

	// Create notifier.
	ur := c.Get("UPCOMING_REMINDER")
	urInt, err := strconv.Atoi(ur)
	gpe.CheckError(err, "UPCOMING_REMINDER must be an integer")
	n := notifier.New(pr, sr, lr, uint(urInt))
	log.Println("Notifier created...")

	a := application.New(n, pr, lr)
	run(bot, a, ctx)
}

func run(b *bt.Bot, a *application.UseCase, ctx context.Context) {

	//The general update channel.
	updateChannel := b.GetUpdateChannel()
	h := handler.New(b, a, ctx)
	h.Start()

	//Monitors any other update.
	for {
		update := <-*updateChannel
		if update.Message == nil {
			continue
		}
		h.Help(update)
	}
}
