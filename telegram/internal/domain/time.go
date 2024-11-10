package domain

import (
	"time"
)

func Time(day int, month time.Month, loc *time.Location) time.Time {
	year := time.Now().In(loc).Year()
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}

func Date(day time.Time, clock time.Time, loc *time.Location) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), clock.Hour(), clock.Minute(), 0, 0, loc)
}

func Clock(hour, minute int, loc *time.Location) time.Time {
	return time.Date(0, 0, 0, hour, minute, 0, 0, loc)
}
