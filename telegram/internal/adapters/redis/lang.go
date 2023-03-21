package redis

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/go-redis/redis/v9"
)

type LanguageRepository struct {
	r *redis.Client
}

func NewLanguageRepository(r *redis.Client) *LanguageRepository {
	return &LanguageRepository{r: r}
}

func (l *LanguageRepository) GetLang(ctx context.Context, id int) (string, error) {
	lang, err := l.r.Get(ctx, l.formatKey(id)).Result()
	if err != nil {
		return "", errors.Wrap(err, "failed to get lang from redis")
	}
	return lang, nil
}

func (l *LanguageRepository) SetLang(ctx context.Context, id int, lang string) error {
	_, err := l.r.Set(ctx, l.formatKey(id), lang, 0).Result()
	if err != nil {
		return errors.Wrap(err, "failed to set lang in redis")
	}
	return nil
}

func (l *LanguageRepository) formatKey(id int) string {
	return fmt.Sprintf("gopray_lang:%d", id)
}
