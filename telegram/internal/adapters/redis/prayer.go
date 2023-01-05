package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/escalopa/gopray/pkg/prayer"
	"github.com/go-redis/redis/v9"
)

type PrayerRepository struct {
	r *redis.Client
}

func NewPrayerRepository(r *redis.Client) *PrayerRepository {
	return &PrayerRepository{r: r}
}

func (p *PrayerRepository) StorePrayer(date string, times prayer.PrayerTimes) error {
	err := p.r.Set(context.TODO(), fmt.Sprintf("prayer:%s", date), times, 0)
	return err.Err()
}

func (p *PrayerRepository) GetPrayer(date string) (prayer.PrayerTimes, error) {
	data := p.r.Get(context.TODO(), fmt.Sprintf("prayer:%s", date))
	if data.Err() != nil {
		return prayer.PrayerTimes{}, data.Err()
	}
	var pt prayer.PrayerTimes
	if err := json.Unmarshal([]byte(data.Val()), &pt); err != nil {
		return prayer.PrayerTimes{}, err
	}
	return pt, nil
}
