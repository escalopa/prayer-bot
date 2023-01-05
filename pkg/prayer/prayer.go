package prayer

import (
	"encoding/json"
	"time"
)

type PrayerTimes struct {
	Date    string    `json:"date"`
	Fajr    time.Time `json:"fajr"`
	Sunrise time.Time `json:"sunrise"`
	Dhuhr   time.Time `json:"dhuhr"`
	Asr     time.Time `json:"asr"`
	Maghrib time.Time `json:"maghrib"`
	Isha    time.Time `json:"isha"`
}

func New(date string, fajr, sunrise, dhuhr, asr, maghrib, isha time.Time) PrayerTimes {
	return PrayerTimes{
		Date:    date,
		Fajr:    fajr,
		Sunrise: sunrise,
		Dhuhr:   dhuhr,
		Asr:     asr,
		Maghrib: maghrib,
		Isha:    isha,
	}
}

func (p PrayerTimes) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p PrayerTimes) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &p)
}
