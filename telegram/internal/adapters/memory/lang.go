package memory

import (
	"context"
	"sync"
)

type LanguageRepository struct {
	languages map[int]string
	mu        sync.RWMutex
}

func NewLanguageRepository() *LanguageRepository {
	return &LanguageRepository{languages: make(map[int]string)}
}

func (l *LanguageRepository) SetLang(_ context.Context, id int, lang string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.languages[id] = lang

	return nil
}

func (l *LanguageRepository) GetLang(_ context.Context, id int) (string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	lang, ok := l.languages[id]
	if !ok || lang == "" {
		return "en", nil
	}

	return lang, nil
}
