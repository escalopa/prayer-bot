package memory

import (
	"context"
	"sync"

	"github.com/escalopa/gopray/telegram/internal/domain"
)

type ScriptRepository struct {
	scripts map[string]*domain.Script
	mu      sync.RWMutex
}

func NewScriptRepository() *ScriptRepository {
	return &ScriptRepository{
		scripts: make(map[string]*domain.Script),
	}
}

func (scr *ScriptRepository) StoreScript(_ context.Context, language string, script *domain.Script) error {
	scr.mu.Lock()
	defer scr.mu.Unlock()

	scr.scripts[language] = script

	return nil
}

func (scr *ScriptRepository) GetScript(_ context.Context, language string) (*domain.Script, error) {
	scr.mu.RLock()
	defer scr.mu.RUnlock()

	script, ok := scr.scripts[language]
	if !ok || script == nil {
		return nil, domain.ErrNotFound
	}

	return script, nil
}
