package memory

import (
	"fmt"

	"github.com/escalopa/gopray/pkg/prayer"
)

type PrayerRepository struct {
	prayers map[string]prayer.PrayerTimes
}

func NewPrayerRepository() *PrayerRepository {
	return &PrayerRepository{prayers: make(map[string]prayer.PrayerTimes)}
}

func (pr *PrayerRepository) GetPrayer(day, month int) (prayer.PrayerTimes, error) {
	key := fmt.Sprintf("%d/%d", day, month)
	return pr.prayers[key], nil
}

func (pr *PrayerRepository) StorePrayer(p prayer.PrayerTimes) error {
	key := fmt.Sprintf("%d/%d", p.Day, p.Month)
	pr.prayers[key] = p
	return nil
}
