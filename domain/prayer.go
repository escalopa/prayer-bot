package domain

import (
	"fmt"
	"time"
)

type PrayerID int

const (
	PrayerIDUnknown PrayerID = 0
	PrayerIDFajr    PrayerID = 1
	PrayerIDShuruq  PrayerID = 2
	PrayerIDDhuhr   PrayerID = 3
	PrayerIDAsr     PrayerID = 4
	PrayerIDMaghrib PrayerID = 5
	PrayerIDIsha    PrayerID = 6
)

type PrayerDay struct {
	Date    time.Time `json:"date"`
	Fajr    time.Time `json:"fajr"`
	Shuruq  time.Time `json:"shuruq"`
	Dhuhr   time.Time `json:"dhuhr"`
	Asr     time.Time `json:"asr"`
	Maghrib time.Time `json:"maghrib"`
	Isha    time.Time `json:"isha"`
}

//revive:disable:argument-limit
func NewPrayerDay(
	date time.Time,
	fajr time.Time,
	shuruq time.Time,
	dhuhr time.Time,
	asr time.Time,
	maghrib time.Time,
	isha time.Time,
) *PrayerDay {
	return &PrayerDay{
		Date:    date,
		Fajr:    fajr,
		Shuruq:  shuruq,
		Dhuhr:   dhuhr,
		Asr:     asr,
		Maghrib: maghrib,
		Isha:    isha,
	}
}

func FormatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dh%dm", h, m)
}

func DateUTC(day int, month time.Month, year int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
