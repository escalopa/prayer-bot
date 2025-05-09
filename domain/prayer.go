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

func Date(day int, month time.Month, year int, loc *time.Location) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}

func DateTime(day time.Time, clock time.Time, loc *time.Location) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), clock.Hour(), clock.Minute(), 0, 0, loc)
}

func Now(loc *time.Location) time.Time {
	now := time.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, loc)
}

// FormatDuration formats the duration into a string with hours and minutes only.
func FormatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dh%dm", h, m)
}
