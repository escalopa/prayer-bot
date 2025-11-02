package domain

import "time"

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

	Reminder struct {
		Tomorrow *ReminderConfig `json:"tomorrow"`
		Soon     *ReminderConfig `json:"soon"`
		Arrive   *ReminderConfig `json:"arrive"`
		Jamaat   *JamaatConfig   `json:"jamaat"`
	}
)

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
