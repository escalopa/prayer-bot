package config

import (
	log "github.com/catalystgo/logger/cli"
	"strconv"
	"time"

	"github.com/escalopa/goconfig"
	"github.com/pkg/errors"
)

var (
	errUpcomingReminder = errors.New("UPCOMING_REMINDER must be between 1 and 59")
	errGomaaNotifyHour  = errors.New("GOMAA_NOTIFY_HOUR must be between 0 and 11")
)

type AppConfig struct {
	Port string

	OwnerID int

	BotToken string
	BotData  string
	Location *time.Location

	CacheURL      string
	CachePrefix   string
	LanguagesPath string

	UpcomingReminder time.Duration
	JummahReminder   time.Duration
}

func InitAppConfig() (*AppConfig, error) {
	cfg := goconfig.New()

	AppCfg := &AppConfig{
		Port: cfg.Get("PORT"),

		BotToken: cfg.Get("BOT_TOKEN"),
		BotData:  cfg.Get("BOT_DATA"),

		CacheURL:      cfg.Get("CACHE_URL"),
		CachePrefix:   cfg.Get("CACHE_PREFIX"),
		LanguagesPath: cfg.Get("LANGUAGES_PATH"),
	}

	ownerID, err := strconv.Atoi(cfg.Get("OWNER_ID"))
	if err != nil {
		return nil, err
	}
	AppCfg.OwnerID = ownerID

	loc, err := time.LoadLocation(cfg.Get("LOCATION"))
	if err != nil {
		return nil, err
	}
	AppCfg.Location = loc

	upcomingReminder, err := time.ParseDuration(cfg.Get("UPCOMING_REMINDER"))
	if err != nil {
		return nil, err
	}
	AppCfg.UpcomingReminder = upcomingReminder

	jummahReminder, err := time.ParseDuration(cfg.Get("JUMMAH_REMINDER"))
	if err != nil {
		return nil, err
	}
	AppCfg.JummahReminder = jummahReminder

	err = AppCfg.validate()
	if err != nil {
		return nil, err
	}

	logCfg(AppCfg)

	return AppCfg, nil
}

func (c *AppConfig) validate() error {
	checkEmpty := func(s string, name string) error {
		if s == "" {
			return errors.Errorf("%s is required", name)
		}
		return nil
	}

	checks := []struct {
		name  string
		value string
	}{
		{"BOT_TOKEN", c.BotToken},
		{"BOT_DATA", c.BotData},
		{"OWNER_ID", strconv.Itoa(c.OwnerID)},
		{"CACHE_URL", c.CacheURL},
		{"CACHE_PREFIX", c.CachePrefix},
		{"LANGUAGES_PATH", c.LanguagesPath},
	}

	for _, check := range checks {
		if err := checkEmpty(check.value, check.name); err != nil {
			return err
		}
	}

	if c.UpcomingReminder.Minutes() <= 0 || c.UpcomingReminder.Minutes() >= 60 {
		return errUpcomingReminder
	}

	if c.JummahReminder.Hours() <= 0 || c.JummahReminder.Hours() >= 12 {
		return errGomaaNotifyHour
	}

	return nil
}

func logCfg(cfg *AppConfig) {
	log.Infof("App Config:")
	log.Infof("--------------------")
	log.Infof("PORT: %s", cfg.Port)
	log.Infof("OWNER_ID: %d", cfg.OwnerID)
	log.Infof("BOT_TOKEN: %s", mask(cfg.BotToken))
	log.Infof("BOT_DATA: %s", cfg.BotData)
	log.Infof("BOT_LOCATION: %s", cfg.Location.String())
	log.Infof("CACHE_URL: %s", mask(cfg.CacheURL))
	log.Infof("CACHE_PREFIX: %s", cfg.CachePrefix)
	log.Infof("LANGUAGES_PATH: %s", cfg.LanguagesPath)
	log.Infof("UPCOMING_REMINDER: %s", cfg.UpcomingReminder.String())
	log.Infof("JUMMAH_REMINDER: %s", cfg.JummahReminder.String())
	log.Infof("--------------------")
}

func mask(s string) string {
	if len(s) > 6 {
		return s[:3] + "***" + s[len(s)-3:]
	}
	return "***"
}
