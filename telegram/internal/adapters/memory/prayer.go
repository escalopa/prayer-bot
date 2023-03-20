package memory

import (
	"fmt"

	"github.com/escalopa/gopray/pkg/core"
)

type PrayerRepository struct {
	prayers map[string]core.PrayerTimes
}

func NewPrayerRepository() *PrayerRepository {
	return &PrayerRepository{prayers: make(map[string]core.PrayerTimes)}
}

func (pr *PrayerRepository) GetPrayer(day, month int) (core.PrayerTimes, error) {
	key := fmt.Sprintf("%d/%d", day, month)
	return pr.prayers[key], nil
}

func (pr *PrayerRepository) StorePrayer(p core.PrayerTimes) error {
	key := fmt.Sprintf("%d/%d", p.Day, p.Month)
	pr.prayers[key] = p
	return nil
}
