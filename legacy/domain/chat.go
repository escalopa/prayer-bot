package domain

import (
	"errors"
)

var (
	ErrUnmarshalJSON   = errors.New("unmarshal json")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrNotFound        = errors.New("not found")
	ErrAlreadyExists   = errors.New("already exists")
	ErrInternal        = errors.New("internal")
)

type (
	Stats struct {
		Users            uint64            // users using the bot
		Subscribed       uint64            // subscribed users
		Unsubscribed     uint64            // unsubscribed users
		LanguagesGrouped map[string]uint64 // users grouped by language
	}

	Chat struct {
		BotID        int64
		ChatID       int64
		State        string
		LanguageCode string
		Subscribed   bool
		Reminder     *Reminder
	}
)
