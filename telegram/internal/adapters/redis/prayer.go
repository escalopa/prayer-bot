package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/escalopa/gopray/pkg/core"
	"github.com/go-redis/redis/v9"
)

type PrayerRepository struct {
	r *redis.Client
}

func NewPrayerRepository(r *redis.Client) *PrayerRepository {
	return &PrayerRepository{r: r}
}

func (p *PrayerRepository) StorePrayer(ctx context.Context, pt core.PrayerTime) error {
	_, err := p.r.Set(ctx, p.formatKey(pt.Day), pt, 0).Result()
	if err != nil {
		return fmt.Errorf("failed to set prayer in redis for %+v: %v", pt, err)
	}
	return nil
}

func (p *PrayerRepository) GetPrayer(ctx context.Context, day time.Time) (core.PrayerTime, error) {
	bytes, err := p.r.Get(ctx, p.formatKey(day)).Result()
	if err != nil {
		if err == redis.Nil {
			return core.PrayerTime{}, errors.New(fmt.Sprintf("prayer not found for %s", day))
		}
		return core.PrayerTime{}, fmt.Errorf("failed to get prayer from redis")
	}
	// Unmarshal
	var pt core.PrayerTime
	if err = json.Unmarshal([]byte(bytes), &pt); err != nil {
		return core.PrayerTime{}, fmt.Errorf("failed to unmarshal prayer from redis")
	}
	return pt, nil
}

func (p *PrayerRepository) formatKey(day time.Time) string {
	return fmt.Sprintf("gopray_prayer_time:%d/%d/%d", day.Day(), int(day.Month()), day.Year())
}
