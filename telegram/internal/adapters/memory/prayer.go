package memory

import (
	"github.com/escalopa/gopray/pkg/prayer"
)

type PrayerRepository struct {
	prayers map[string]prayer.PrayerTimes
}

func NewPrayerRepository() *PrayerRepository {
	return &PrayerRepository{prayers: make(map[string]prayer.PrayerTimes)}
}

func (pr *PrayerRepository) GetPrayer(date string) (prayer.PrayerTimes, error) {
	return pr.prayers[date], nil
}

func (pr *PrayerRepository) StorePrayer(date string, times prayer.PrayerTimes) error {
	pr.prayers[date] = times
	return nil
}
