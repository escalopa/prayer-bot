package domain

import (
	"errors"
	"time"
)

var (
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

type ReminderConfig struct {
	Offset    time.Duration `json:"offset"`
	MessageID int           `json:"message_id"`
	LastAt    time.Time     `json:"last_at"`
}

type JamaatDelay struct {
	Fajr    time.Duration `json:"fajr"`    // default 10m (set on creation only)
	Shuruq  time.Duration `json:"shuruq"`  // default 10m (set on creation only)
	Dhuhr   time.Duration `json:"dhuhr"`   // default 10m (set on creation only)
	Asr     time.Duration `json:"asr"`     // default 10m (set on creation only)
	Maghrib time.Duration `json:"maghrib"` // default 10m (set on creation only)
	Isha    time.Duration `json:"isha"`    // default 20m (set on creation only)
}

func (j *JamaatDelay) GetDelayByPrayerID(prayerID PrayerID) time.Duration {
	switch prayerID {
	case PrayerIDFajr:
		return j.Fajr
	case PrayerIDShuruq:
		return j.Shuruq
	case PrayerIDDhuhr:
		return j.Dhuhr
	case PrayerIDAsr:
		return j.Asr
	case PrayerIDMaghrib:
		return j.Maghrib
	case PrayerIDIsha:
		return j.Isha
	default:
		return 0
	}
}

type Reminder struct {
	Today       ReminderConfig `json:"today"`
	Soon        ReminderConfig `json:"soon"`
	Arrive      ReminderConfig `json:"arrive"`
	JamaatDelay JamaatDelay    `json:"jamaat_delay"`
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
		IsGroup      bool
		Reminder     Reminder
	}
)
