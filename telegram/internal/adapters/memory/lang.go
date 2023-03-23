package memory

import (
	"context"

	"github.com/pkg/errors"
)

type LanguageRepository struct {
	m map[int]string
}

func NewLanguageRepository() *LanguageRepository {
	return &LanguageRepository{m: make(map[int]string)}
}

func (l *LanguageRepository) GetLang(ctx context.Context, id int) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if _, ok := l.m[id]; !ok {
		return "", errors.New("language not found")
	}
	return l.m[id], nil
}

func (l *LanguageRepository) SetLang(ctx context.Context, id int, lang string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	l.m[id] = lang
	return nil
}
