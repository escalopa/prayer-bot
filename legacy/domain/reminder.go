package domain

import (
	"fmt"
	"time"
)

type ReminderType string

const (
	ReminderTypeTomorrow ReminderType = "tomorrow"
	ReminderTypeSoon     ReminderType = "soon"
	ReminderTypeArrive   ReminderType = "arrive"
)

func (rt ReminderType) String() string {
	return string(rt)
}

type (
	JamaatDelayConfig struct {
		Fajr    Duration `json:"fajr"`
		Dhuhr   Duration `json:"dhuhr"`
		Asr     Duration `json:"asr"`
		Maghrib Duration `json:"maghrib"`
		Isha    Duration `json:"isha"`
	}

	JamaatConfig struct {
		Enabled bool               `json:"enabled"`
		Delay   *JamaatDelayConfig `json:"delay"`
	}

	ReminderConfig struct {
		Offset    Duration  `json:"offset"`
		MessageID int       `json:"message_id"`
		LastAt    time.Time `json:"last_at"`
	}

	// PrayerOverrideConfig stores a fixed local clock time (HH:MM) for a prayer.
	// An empty value keeps the calculated time for that prayer.
	PrayerOverrideConfig struct {
		Fajr    string `json:"fajr,omitempty"`
		Dhuhr   string `json:"dhuhr,omitempty"`
		Asr     string `json:"asr,omitempty"`
		Maghrib string `json:"maghrib,omitempty"`
		Isha    string `json:"isha,omitempty"`
	}

	Reminder struct {
		Tomorrow  *ReminderConfig       `json:"tomorrow"`
		Soon      *ReminderConfig       `json:"soon"`
		Arrive    *ReminderConfig       `json:"arrive"`
		Jamaat    *JamaatConfig         `json:"jamaat"`
		Overrides *PrayerOverrideConfig `json:"overrides,omitempty"`
	}
)

func ParsePrayerClock(value string) (hour, minute int, err error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid prayer clock %q: %w", value, err)
	}
	return parsed.Hour(), parsed.Minute(), nil
}

func (c *PrayerOverrideConfig) Get(prayerID PrayerID) string {
	if c == nil {
		return ""
	}
	switch prayerID {
	case PrayerIDFajr:
		return c.Fajr
	case PrayerIDDhuhr:
		return c.Dhuhr
	case PrayerIDAsr:
		return c.Asr
	case PrayerIDMaghrib:
		return c.Maghrib
	case PrayerIDIsha:
		return c.Isha
	default:
		return ""
	}
}

func (c *PrayerOverrideConfig) Set(prayerID PrayerID, value string) bool {
	switch prayerID {
	case PrayerIDFajr:
		c.Fajr = value
	case PrayerIDDhuhr:
		c.Dhuhr = value
	case PrayerIDAsr:
		c.Asr = value
	case PrayerIDMaghrib:
		c.Maghrib = value
	case PrayerIDIsha:
		c.Isha = value
	default:
		return false
	}
	return true
}

func (j *JamaatDelayConfig) GetDelayByPrayerID(prayerID PrayerID) time.Duration {
	switch prayerID {
	case PrayerIDFajr:
		return time.Duration(j.Fajr)
	case PrayerIDDhuhr:
		return time.Duration(j.Dhuhr)
	case PrayerIDAsr:
		return time.Duration(j.Asr)
	case PrayerIDMaghrib:
		return time.Duration(j.Maghrib)
	case PrayerIDIsha:
		return time.Duration(j.Isha)
	default:
		return 0
	}
}

func (j *JamaatDelayConfig) SetDelayByPrayerID(prayerID PrayerID, delay time.Duration) {
	switch prayerID {
	case PrayerIDFajr:
		j.Fajr = Duration(delay)
	case PrayerIDDhuhr:
		j.Dhuhr = Duration(delay)
	case PrayerIDAsr:
		j.Asr = Duration(delay)
	case PrayerIDMaghrib:
		j.Maghrib = Duration(delay)
	case PrayerIDIsha:
		j.Isha = Duration(delay)
	}
}
