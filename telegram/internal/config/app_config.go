package config

import (
	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/escalopa/goconfig"
	"github.com/pkg/errors"
)

var (
	errUpcomingReminder = errors.New("UPCOMING_REMINDER must be between 1 and 59")
	errGomaaNotifyHour  = errors.New("GOMAA_NOTIFY_HOUR must be between 0 and 11")
)

type BotConfig struct {
	Name     string   `json:"name"`
	Prefix   string   `json:"prefix"`
	Token    string   `json:"token"`
	Data     string   `json:"data"`
	Location *timeLoc `json:"location"`
}

type AppConfig struct {
	Port string

	OwnerID    int
	BotsConfig []BotConfig

	CacheURL      string
	LanguagesPath string

	UpcomingReminder time.Duration
	JummahReminder   time.Duration
}

func InitAppConfig() (*AppConfig, error) {
	cfg := goconfig.New()

	AppCfg := &AppConfig{
		Port:          cfg.Get("PORT"),
		CacheURL:      cfg.Get("CACHE_URL"),
		LanguagesPath: cfg.Get("LANGUAGES_PATH"),
	}

	bowOwnerID, err := strconv.Atoi(cfg.Get("OWNER_ID"))
	if err != nil {
		return nil, err
	}
	AppCfg.OwnerID = bowOwnerID

	var botsConfig []BotConfig
	err = json.Unmarshal([]byte(cfg.Get("BOTS_CONFIG")), &botsConfig)
	if err != nil {
		return nil, err
	}
	AppCfg.BotsConfig = botsConfig

	// Read bot tokens from secret files.
	for _, bot := range AppCfg.BotsConfig {
		data, err := os.ReadFile(bot.Token)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				// TODO: log error
				continue
			}
			return nil, err
		}
		bot.Token = string(data)
	}

	upcomingReminder, err := time.ParseDuration(cfg.Get("UPCOMING_REMINDER"))
	if err != nil {
		return nil, err
	}
	// Check if the upcoming reminder is between 1 and 59 minutes.
	if upcomingReminder.Minutes() <= 0 || upcomingReminder.Minutes() >= 60 {
		return nil, errUpcomingReminder
	}
	AppCfg.UpcomingReminder = upcomingReminder

	jummahReminder, err := time.ParseDuration(cfg.Get("JUMMAH_REMINDER"))
	if err != nil {
		return nil, err
	}
	// Check if the jummah reminder is between 0 and 11 hours.
	if jummahReminder.Hours() <= 0 || jummahReminder.Hours() >= 12 {
		return nil, errGomaaNotifyHour
	}
	AppCfg.JummahReminder = jummahReminder

	logCfg(AppCfg)

	return AppCfg, nil
}

func logCfg(cfg *AppConfig) {
	//TODO: impl
}
