package internal

import (
	"fmt"
	"path"
	"strconv"
	"strings"
)

const (
	filenameParts = 2
)

// ExtractBotID extracts the bot ID from the filename.
// example filename: "BOT_ID-CITY_NAME.csv"
func ExtractBotID(filename string) (uint8, error) {
	parts := strings.Split(path.Base(filename), "-")
	if len(parts) != filenameParts {
		return 0, fmt.Errorf("unexpected filename format: %s", filename)
	}

	botID, err := strconv.ParseUint(parts[0], 10, 8)
	if err != nil {
		return 0, fmt.Errorf("parse bot_id: %s => %w", parts[0], err)
	}

	return uint8(botID), nil
}
