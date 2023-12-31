package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/escalopa/gopray/pkg/core"
)

type PrayerRepository struct {
	prayers map[string]core.PrayerTime
}

func NewPrayerRepository() *PrayerRepository {
	return &PrayerRepository{prayers: make(map[string]core.PrayerTime)}
}

func (pr *PrayerRepository) GetPrayer(ctx context.Context, day time.Time) (core.PrayerTime, error) {
	if err := ctx.Err(); err != nil {
		return core.PrayerTime{}, fmt.Errorf("failed to inmemory GetPrayer: %v", err)
	}
	key := formatKey(day)
	val, ok := pr.prayers[key]
	if !ok {
		return core.PrayerTime{}, fmt.Errorf("prayer not found for %s", key)
	}
	return val, nil
}

func (pr *PrayerRepository) StorePrayer(ctx context.Context, p core.PrayerTime) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("failed to inmemory StorePrayer: %v", err)
	}
	key := formatKey(p.Day)
	pr.prayers[key] = p
	return nil
}

func formatKey(day time.Time) string {
	return fmt.Sprintf("%d/%d/%d", day.Day(), day.Month(), day.Year())
}
