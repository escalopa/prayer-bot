package redis

import (
	"context"
	"encoding/json"
	"fmt"

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

func (p *PrayerRepository) StorePrayer(ctx context.Context, times core.PrayerTimes) error {
	_, err := p.r.Set(ctx, p.formatKey(times.Day, times.Month), times, 0).Result()
	if err != nil {
		return errors.Wrap(err, "failed to set prayer in redis")
	}
	return nil
}

func (p *PrayerRepository) GetPrayer(ctx context.Context, day, month int) (core.PrayerTimes, error) {
	bytes, err := p.r.Get(ctx, p.formatKey(day, month)).Result()
	if err != nil {
		if err == redis.Nil {
			return core.PrayerTimes{}, errors.New(fmt.Sprintf("prayer not found for %d/%d", day, month))
		}
		return core.PrayerTimes{}, errors.Wrap(err, "failed to get prayer from redis")
	}
	// Unmarshal
	var pt core.PrayerTimes
	if err = json.Unmarshal([]byte(bytes), &pt); err != nil {
		return core.PrayerTimes{}, errors.Wrap(err, "failed to unmarshal prayer from redis")
	}
	return pt, nil
}

func (p *PrayerRepository) formatKey(day, month int) string {
	return fmt.Sprintf("gopray_prayer_time:%d/%d", day, month)
}
