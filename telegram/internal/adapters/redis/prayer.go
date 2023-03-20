package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/escalopa/gopray/pkg/core"
	"github.com/go-redis/redis/v9"
)

type PrayerRepository struct {
	r *redis.Client
}

func NewPrayerRepository(r *redis.Client) *PrayerRepository {
	return &PrayerRepository{r: r}
}

func (p *PrayerRepository) StorePrayer(date string, times core.PrayerTimes) error {
	err := p.r.Set(context.Background(), fmt.Sprintf("prayer:%s", date), times, 0)
	return err.Err()
}

func (p *PrayerRepository) GetPrayer(date string) (core.PrayerTimes, error) {
	data := p.r.Get(context.Background(), fmt.Sprintf("prayer:%s", date))
	if data.Err() != nil {
		return core.PrayerTimes{}, data.Err()
	}
	var pt core.PrayerTimes
	if err := json.Unmarshal([]byte(data.Val()), &pt); err != nil {
		return core.PrayerTimes{}, err
	}
	return pt, nil
}
