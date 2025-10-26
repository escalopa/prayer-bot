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
		Fajr    time.Duration `json:"fajr"`
		Dhuhr   time.Duration `json:"dhuhr"`
		Asr     time.Duration `json:"asr"`
		Maghrib time.Duration `json:"maghrib"`
		Isha    time.Duration `json:"isha"`
	}

	JamaatConfig struct {
		Enabled bool               `json:"enabled"`
		Delay   *JamaatDelayConfig `json:"delay"`
	}

	ReminderConfig struct {
		Offset    time.Duration `json:"offset"`
		MessageID int           `json:"message_id"`
		LastAt    time.Time     `json:"last_at"`
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
		return j.Fajr
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

func (j *JamaatDelayConfig) SetDelayByPrayerID(prayerID PrayerID, delay time.Duration) {
	switch prayerID {
	case PrayerIDFajr:
		j.Fajr = delay
	case PrayerIDDhuhr:
		j.Dhuhr = delay
	case PrayerIDAsr:
		j.Asr = delay
	case PrayerIDMaghrib:
		j.Maghrib = delay
	case PrayerIDIsha:
		j.Isha = delay
	}
}
