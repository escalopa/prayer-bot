package domain

import (
	"sync"
	"time"
)

var (
	loc  *time.Location
	once sync.Once
)

func SetLocation(l *time.Location) {
	once.Do(func() {
		loc = l
	})
}

func GetLocation() *time.Location {
	return loc
}

func Time(day int, month time.Month, year int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}

func Date(day time.Time, clock time.Time) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), clock.Hour(), clock.Minute(), 0, 0, loc)
}
