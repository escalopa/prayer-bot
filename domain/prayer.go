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
