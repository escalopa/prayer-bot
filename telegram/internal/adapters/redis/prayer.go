package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/escalopa/gopray/telegram/internal/domain"
	"github.com/go-redis/redis/v9"
	"github.com/pkg/errors"
)

type PrayerRepository struct {
	client *redis.Client
	prefix string
}

func NewPrayerRepository(client *redis.Client, prefix string) *PrayerRepository {
	return &PrayerRepository{
		client: client,
		prefix: prefix,
	}
}

func (p *PrayerRepository) StorePrayer(ctx context.Context, pt *domain.PrayerTime) error {
	_, err := p.client.Set(ctx, p.formatKey(pt.Day), pt, 0).Result()
	if err != nil {
		return errors.Errorf("StorePrayer: %v", err)
	}
	return nil
}

func (p *PrayerRepository) GetPrayer(ctx context.Context, day time.Time) (*domain.PrayerTime, error) {
	bytes, err := p.client.Get(ctx, p.formatKey(day)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrNotFound
		}
		return nil, errors.Errorf("GetPrayer: %v", err)
	}
	var pt domain.PrayerTime
	if err = json.Unmarshal([]byte(bytes), &pt); err != nil {
		return nil, errors.Errorf("GetPrayer: %v", err)
	}
	return &pt, nil
}

func (p *PrayerRepository) formatKey(day time.Time) string {
	return fmt.Sprintf("%s_gopray_prayer_time:%d/%d/%d", p.prefix, day.Day(), int(day.Month()), day.Year())
}
