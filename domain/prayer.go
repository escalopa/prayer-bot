package domain

import "time"

type PrayerTimes struct {
	Date    time.Time `json:"date"`
	Fajr    time.Time `json:"fajr"`
	Shuruq  time.Time `json:"shuruq"`
	Dhuhr   time.Time `json:"dhuhr"`
	Asr     time.Time `json:"asr"`
	Maghrib time.Time `json:"maghrib"`
	Isha    time.Time `json:"isha"`
}

func NewPrayerTimes(
	date time.Time,
	fajr time.Time,
	shuruq time.Time,
	dhuhr time.Time,
	asr time.Time,
	maghrib time.Time,
	isha time.Time,
) *PrayerTimes {
	return &PrayerTimes{
		Date:    date,
		Fajr:    fajr,
		Shuruq:  shuruq,
		Dhuhr:   dhuhr,
		Asr:     asr,
		Maghrib: maghrib,
		Isha:    isha,
	}
}
