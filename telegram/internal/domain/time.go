package core

import (
	"sync"
	"time"
)

var (
	once sync.Once
	loc  = time.UTC
)

func DefaultTime(day, month, year int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc)
}

func SetLocation(l *time.Location) {
	once.Do(func() {
		loc = l
	})
}

func GetLocation() *time.Location {
	return loc
}
