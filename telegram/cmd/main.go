package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	redis2 "github.com/go-redis/redis/v9"

	"github.com/escalopa/gopray/pkg/core"
	"github.com/escalopa/gopray/telegram/internal/adapters/memory"
	"github.com/escalopa/gopray/telegram/internal/handler"

	bt "github.com/SakoDroid/telego"
	cfg "github.com/SakoDroid/telego/configs"
	"github.com/escalopa/goconfig"

	"github.com/escalopa/gopray/telegram/internal/adapters/notifier"
	"github.com/escalopa/gopray/telegram/internal/adapters/parser"
	"github.com/escalopa/gopray/telegram/internal/adapters/redis"
	"github.com/escalopa/gopray/telegram/internal/application"
)

var (
	c = goconfig.New()
)

func main() {

	// Create a new bot instance.
	bot, err := bt.NewBot(cfg.Default(c.Get("BOT_TOKEN")))
	checkError(err, "failed to create bot instance")

	// Parse bot owner id
	ownerIDString := c.Get("BOT_OWNER_ID")
	ownerID, err := strconv.Atoi(ownerIDString)
	checkError(err, "failed to parse BOT_OWNER_ID")

	// Create base context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load application time location.
	loc, err := time.LoadLocation(c.Get("TIME_LOCATION"))
	checkError(err, "failed to load time location")
	log.Printf("successfully loaded time zone: %s", loc)
	core.SetLocation(loc)

	// Set up the database.
	r := redis.New(c.Get("CACHE_URL"))
	defer func(r *redis2.Client) {
		checkError(r.Close(), "failed to close redis client")
	}(r)

	// pr := redis.NewPrayerRepository(r)
	pr := memory.NewPrayerRepository() // Use memory for prayer repository. To not hit the cache on every reload.
	sr := redis.NewSubscriberRepository(r)
	lr := redis.NewLanguageRepository(r)
	hr := redis.NewHistoryRepository(r)
	scr := memory.NewScriptRepository() // Use memory for script repository. To not hit the cache on every reload.
	log.Println("successfully connected to database")

	// Create schedule parser & parse the schedule.
	pp := parser.NewPrayerParser(c.Get("DATA_PATH"), parser.WithPrayerRepository(pr))
	checkError(pp.ParseSchedule(ctx), "failed to parse schedule")
	log.Println("successfully parsed prayer's schedule")

	// Create language parser & parse the languages.
	lp := parser.NewScriptParser(c.Get("LANGUAGES_PATH"), parser.WithScriptRepository(scr))
	checkError(lp.ParseScripts(ctx), "failed to parse languages")
	log.Println("successfully parsed languages")

	// Parse upcoming reminder.
	ur := c.Get("UPCOMING_REMINDER")
	urDuration, err := time.ParseDuration(ur)
	checkError(err, "failed to parse UPCOMING_REMINDER")
	log.Printf("successfully parsed upcoming reminder %s", ur)

	// Parse gomaa notify hour.
	gnh := c.Get("GOMAA_NOTIFY_HOUR")
	gnhDuration, err := time.ParseDuration(gnh)
	checkError(err, "failed to parse GOMAA_NOTIFY_HOUR")
	log.Printf("successfully parsed gomaa notify hour %s", gnh)

	// Create notifier.
	n, err := notifier.New(urDuration, gnhDuration,
		notifier.WithPrayerRepository(pr),
		notifier.WithSubscriberRepository(sr),
		notifier.WithLanguageRepository(lr),
	)
	checkError(err)
	log.Printf("successfully created notifier with upcoming reminder: %s and gomaa notify hour: %s", ur, gnh)

	// Create use cases.
	useCases := application.New(ctx,
		application.WithNotifier(n),
		application.WithPrayerRepository(pr),
		application.WithSubscriberRepository(sr),
		application.WithLanguageRepository(lr),
		application.WithHistoryRepository(hr),
		application.WithScriptRepository(scr),
	)
	log.Println("successfully created use cases")
	run(ctx, bot, ownerID, useCases)
}

func run(ctx context.Context, b *bt.Bot, ownerID int, useCases *application.UseCase) {
	// Create handler & start it.
	h := handler.New(ctx, b, ownerID, useCases)
	checkError(h.Run(), "failed to start handler")
	checkError(b.Run(), "failed to run bot")
	//The general update channel.
	updateChannel := b.GetUpdateChannel()

	go health()
	for {
		update := <-*updateChannel
		if update.Message == nil {
			continue
		}
		_, err := b.SendMessage(update.Message.Chat.Id, "/help", "", 0, false, false)
		if err != nil {
			log.Printf("failed to send message on unknown command, %v", err)
		}
	}
}

func health() {
	port := c.Get("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	log.Printf("starting server on port: %s", port)
	err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func checkError(err error, message ...string) {
	if err != nil {
		log.Fatal(err, message)
	}
}
