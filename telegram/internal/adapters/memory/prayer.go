package memory

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/escalopa/gopray/pkg/core"
)

type PrayerRepository struct {
	prayers map[string]core.PrayerTimes
}

func NewPrayerRepository() *PrayerRepository {
	return &PrayerRepository{prayers: make(map[string]core.PrayerTimes)}
}

func (pr *PrayerRepository) GetPrayer(ctx context.Context, day, month int) (core.PrayerTimes, error) {
	if err := ctx.Err(); err != nil {
		return core.PrayerTimes{}, err
	}
	key := formatKey(day, month)
	val, ok := pr.prayers[key]
	if !ok {
		return core.PrayerTimes{}, errors.New(fmt.Sprintf("prayer not found for %d/%d", day, month))
	}
	return val, nil
}

func (pr *PrayerRepository) StorePrayer(ctx context.Context, p core.PrayerTimes) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key := formatKey(p.Day, p.Month)
	pr.prayers[key] = p
	return nil
}

func formatKey(day, month int) string {
	return fmt.Sprintf("%d/%d", day, month)
}
