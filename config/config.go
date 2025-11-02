package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/escalopa/prayer-bot/domain"
)

func Load() (botConfig map[int64]*domain.BotConfig, _ error) {
	encodedData := os.Getenv("APP_CONFIG")

	// Decode base64-encoded config
	data, err := base64.StdEncoding.DecodeString(encodedData)
	if err != nil {
		return nil, fmt.Errorf("decode base64 config: %v", err)
	}

	err = json.Unmarshal(data, &botConfig)
	if err != nil {
		return nil, fmt.Errorf("unmarshal bot config: %v", err)
	}

	return botConfig, nil
}
