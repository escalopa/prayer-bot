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

func (lr *LanguageRepository) SetLang(_ context.Context, id int, lang string) error {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	lr.languages[id] = lang

	return nil
}

func (lr *LanguageRepository) GetLang(_ context.Context, id int) (string, error) {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	lang, ok := lr.languages[id]
	if !ok || lang == "" {
		return "en", nil
	}

	return lang, nil
}
