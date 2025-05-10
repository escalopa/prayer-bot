package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/escalopa/prayer-bot/domain"
)

func Load() (botConfig map[int64]*domain.BotConfig, _ error) {
	data := os.Getenv("APP_CONFIG")

	err := json.Unmarshal([]byte(data), &botConfig)
	if err != nil {
		return nil, fmt.Errorf("unmarshal bot config: %v", err)
	}

	return botConfig, nil
}
