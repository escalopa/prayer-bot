package redis

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v9"
)

type LanguageRepository struct {
	r *redis.Client
}

func NewLanguageRepository(r *redis.Client) *LanguageRepository {
	return &LanguageRepository{r: r}
}

func (l *LanguageRepository) GetLang(id int) (string, error) {
	res := l.r.Get(context.TODO(), fmt.Sprintf("lang:%d", id))
	return res.Result()
}

func (l *LanguageRepository) SetLang(id int, lang string) error {
	res := l.r.Set(context.TODO(), fmt.Sprintf("lang:%d", id), lang, 0)
	return res.Err()
}
