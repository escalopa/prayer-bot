package domain

import (
	"strconv"
	"time"
)

type (
	location time.Location

	BotConfig struct {
		BotID    int64     `json:"bot_id"`
		OwnerID  int64     `json:"owner_id"`
		Token    string    `json:"token"`
		Secret   string    `json:"secret"`
		Location *location `json:"location"`
	}
)

func (l *location) UnmarshalJSON(bytes []byte) error {
	data, err := strconv.Unquote(string(bytes))
	if err != nil {
		return err
	}

	loc, err := time.LoadLocation(data)
	if err != nil {
		return err
	}

	*l = location(*loc)
	return nil
}

func (l *location) V() *time.Location {
	if l == nil {
		return nil
	}
	return (*time.Location)(l)
}
