package internal

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/escalopa/prayer-bot/domain"
)

const (
	telegramSecretTokenHeader = "X-Telegram-Bot-Api-Secret-Token"

	secretTokenPartsCount = 2
)

func Authenticate(config map[uint8]*domain.BotConfig, headers map[string]string) (uint8, error) {
	secretToken := headers[telegramSecretTokenHeader]
	if secretToken == "" {
		return 0, fmt.Errorf("empty secret token header")
	}

	parts := strings.Split(secretToken, "-")
	if len(parts) != secretTokenPartsCount {
		return 0, fmt.Errorf("unexpected secret token format")
	}

	botID, err := strconv.ParseUint(parts[0], 10, 8)
	if err != nil {
		return 0, fmt.Errorf("parse bot_id: %s => %v", parts[0], err)
	}

	if botID == 0 {
		return 0, fmt.Errorf("bot_id cannot be 0")
	}

	botConfig, ok := config[uint8(botID)]
	if !ok {
		return 0, fmt.Errorf("bot config not found")
	}

	if botConfig.Secret != secretToken {
		return 0, fmt.Errorf("secret token mismatch")
	}

	return uint8(botID), nil
}
