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

type reminderOffset int32

const (
	reminderOffset5m  reminderOffset = 5
	reminderOffset10m reminderOffset = 10
	reminderOffset15m reminderOffset = 15
	reminderOffset20m reminderOffset = 20
	reminderOffset30m reminderOffset = 30
	reminderOffset45m reminderOffset = 45
	reminderOffset60m reminderOffset = 60
	reminderOffset90m reminderOffset = 90
)

func ReminderOffsets() []int32 {
	return []int32{
		int32(reminderOffset5m),
		int32(reminderOffset10m),
		int32(reminderOffset15m),
		int32(reminderOffset20m),
		int32(reminderOffset30m),
		int32(reminderOffset45m),
		int32(reminderOffset60m),
		int32(reminderOffset90m),
	}
}

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
