package main

import (
	"context"
	"fmt"

	log "github.com/catalystgo/logger/cli"

	bt "github.com/SakoDroid/telego"
	telegoCfg "github.com/SakoDroid/telego/configs"
	"github.com/escalopa/gopray/telegram/internal/adapters/memory"
	"github.com/escalopa/gopray/telegram/internal/adapters/parser"
	"github.com/escalopa/gopray/telegram/internal/adapters/redis"
	"github.com/escalopa/gopray/telegram/internal/adapters/scheduler"
	app "github.com/escalopa/gopray/telegram/internal/application"
	"github.com/escalopa/gopray/telegram/internal/config"
	"github.com/escalopa/gopray/telegram/internal/handler"
	"github.com/escalopa/gopray/telegram/internal/server"
	redis2 "github.com/go-redis/redis/v9"
)

func main() {
	// Create base context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	appCfg, err := config.InitAppConfig()
	checkError(err, "init app config")

	// Set up the database.
	r, err := redis.New(appCfg.CacheURL)
	checkError(err, "failed to connect to redis")

	defer func(r *redis2.Client) {
		checkError(r.Close(), "failed to close redis client")
	}(r)

	scr := memory.NewScriptRepository() // Use memory for script repository to not hit db on every call.

	lp := parser.NewScriptParser(appCfg.LanguagesPath, scr)
	checkError(lp.Parse(), "failed to parse languages")

	srv := server.New()

	bot, err := bt.NewBot(telegoCfg.Default(appCfg.BotToken))
	checkError(err, "create bot")

	loc := appCfg.Location

	pr := memory.NewPrayerRepository() // Use memory for prayer repository to not hit db on every call.
	sr := redis.NewSubscriberRepository(r, appCfg.CachePrefix)
	lr := redis.NewLanguageRepository(r, appCfg.CachePrefix)
	hr := redis.NewHistoryRepository(r, appCfg.CachePrefix)

	pp := parser.NewPrayerParser(appCfg.BotData, pr, loc)
	checkError(pp.LoadSchedule(ctx), "parse schedule")

	sch := scheduler.New(appCfg.UpcomingReminder, appCfg.JummahReminder, loc, pr, sr)

	uc := app.NewUseCase(ctx, loc, sch, pr, scr, hr, lr, sr)
	h := handler.New(bot, appCfg.OwnerID, uc)

	srv.Run(ctx, h, appCfg.Port)
}

func checkError(err error, message string) {
	if err != nil {
		message = fmt.Sprintf("%s => %v", message, err)
		log.Fatal(message)
	}
}
