package domain

import (
	"encoding/json"
	"time"
)

type PrayerTime struct {
	Day     time.Time `json:"day"`
	Fajr    time.Time `json:"fajr"`
	Dohaa   time.Time `json:"dohaa"`
	Dhuhr   time.Time `json:"dhuhr"`
	Asr     time.Time `json:"asr"`
	Maghrib time.Time `json:"maghrib"`
	Isha    time.Time `json:"isha"`
}

func NewPrayerTime(day, fajr, dohaa, dhuhr, asr, maghrib, isha time.Time) *PrayerTime {
	return &PrayerTime{
		Day:     day,
		Fajr:    fajr,
		Dohaa:   dohaa,
		Dhuhr:   dhuhr,
		Asr:     asr,
		Maghrib: maghrib,
		Isha:    isha,
	}
}

func (p PrayerTime) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p PrayerTime) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &p)
}
