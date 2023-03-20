package main

import (
	"context"
	"log"
	"time"

	redis2 "github.com/go-redis/redis/v9"

	"github.com/escalopa/gopray/telegram/internal/adapters/memory"
	"github.com/escalopa/gopray/telegram/internal/handler"

	bt "github.com/SakoDroid/telego"
	cfg "github.com/SakoDroid/telego/configs"
	"github.com/escalopa/goconfig"

	gpe "github.com/escalopa/gopray/pkg/error"
	"github.com/escalopa/gopray/telegram/internal/adapters/notifier"
	"github.com/escalopa/gopray/telegram/internal/adapters/parser"
	"github.com/escalopa/gopray/telegram/internal/adapters/redis"
	"github.com/escalopa/gopray/telegram/internal/application"
)

func main() {
	c := goconfig.New()

	// Create a new bot instance.
	bot, err := bt.NewBot(cfg.Default(c.Get("BOT_TOKEN")))
	gpe.CheckError(err, "failed to create bot instance")

	// Create base context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load application time location.
	loc, err := time.LoadLocation(c.Get("TIME_LOCATION"))
	gpe.CheckError(err, "failed to load time location")

	// Set up the database.
	r := redis.New(c.Get("CACHE_URL"))
	defer func(r *redis2.Client) {
		gpe.CheckError(r.Close(), "failed to close redis client")
	}(r)
	// pr := redis.NewPrayerRepository(r)
	pr := memory.NewPrayerRepository() // Use memory for prayer repository. To not hit the cache on every reload.
	sr := redis.NewSubscriberRepository(r)
	lr := redis.NewLanguageRepository(r)
	log.Println("successfully connected to database")

	// Create schedule parser & parse the schedule.
	p := parser.New(c.Get("DATA_PATH"), parser.WithPrayerRepository(pr), parser.WithTimeLocation(loc))
	gpe.CheckError(p.ParseSchedule(ctx), "failed to parse schedule")
	log.Println("successfully parsed prayer's schedule")

	// Parse upcoming reminder.
	ur := c.Get("UPCOMING_REMINDER")
	urDuration, err := time.ParseDuration(ur)
	gpe.CheckError(err, "failed to parse UPCOMING_REMINDER")
	log.Printf("successfully parsed upcoming reminder %s", ur)

	// Parse gomaa notify hour.
	gnh := c.Get("GOMAA_NOTIFY_HOUR")
	gnhDuration, err := time.ParseDuration(gnh)
	gpe.CheckError(err, "failed to parse GOMAA_NOTIFY_HOUR")
	log.Printf("successfully parsed gomaa notify hour %s", gnh)

	// Create notifier.
	n, err := notifier.New(urDuration, gnhDuration,
		notifier.WithPrayerRepository(pr),
		notifier.WithSubscriberRepository(sr),
		notifier.WithLanguageRepository(lr),
		notifier.WithTimeLocation(loc),
	)
	gpe.CheckError(err)
	log.Printf("successfully created notifier with upcoming reminder: %s and gomaa notify hour: %s", ur, gnh)

	// Create use cases.
	useCases := application.New(ctx,
		application.WithNotifier(n),
		application.WithPrayerRepository(pr),
		application.WithSubscriberRepository(sr),
		application.WithLanguageRepository(lr),
	)
	log.Println("successfully created use cases")
	run(ctx, bot, useCases)
}

func run(ctx context.Context, b *bt.Bot, useCases *application.UseCase) {
	// Create handler & start it.
	h := handler.New(ctx, b, useCases)
	gpe.CheckError(h.Start(), "failed to start handler")
	gpe.CheckError(b.Run(), "failed to run bot")
	//The general update channel.
	updateChannel := b.GetUpdateChannel()
	for {
		update := <-*updateChannel
		if update.Message == nil {
			continue
		}
		h.Help(update)
	}
}
