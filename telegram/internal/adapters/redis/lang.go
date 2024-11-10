package redis

import (
	"context"
	"fmt"

	"github.com/escalopa/gopray/telegram/internal/domain"
	"github.com/go-redis/redis/v9"
	"github.com/pkg/errors"
)

type LanguageRepository struct {
	client *redis.Client
	prefix string
}

func NewLanguageRepository(client *redis.Client, prefix string) *LanguageRepository {
	return &LanguageRepository{
		client: client,
		prefix: prefix,
	}
}

func (l *LanguageRepository) SetLang(ctx context.Context, id int, lang string) error {
	_, err := l.client.Set(ctx, l.formatKey(id), lang, 0).Result()
	if err != nil {
		return errors.Errorf("SetLang: %v", err)
	}
	return nil
}

func (l *LanguageRepository) GetLang(ctx context.Context, id int) (string, error) {
	lang, err := l.client.Get(ctx, l.formatKey(id)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", domain.ErrNotFound
		}
		return "", errors.Errorf("GetLang: %v", err)
	}
	return lang, nil
}

func (l *LanguageRepository) formatKey(id int) string {
	return fmt.Sprintf("%s:lang:%d", l.prefix, id)
}
