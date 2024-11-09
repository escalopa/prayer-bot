package memory

import (
	"context"
	"sync"

	"github.com/escalopa/gopray/telegram/internal/domain"

	"github.com/escalopa/gopray/pkg/language"
)

type ScriptRepository struct {
	scripts map[string]*language.Script
	mu      sync.RWMutex
}

func NewScriptRepository() *ScriptRepository {
	return &ScriptRepository{
		scripts: make(map[string]*language.Script),
	}
}

func (r *ScriptRepository) StoreScript(_ context.Context, language string, script *language.Script) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.scripts[language] = script

	return nil
}

func (r *ScriptRepository) GetScript(_ context.Context, language string) (*language.Script, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	script, ok := r.scripts[language]
	if !ok || script == nil {
		return nil, domain.ErrNotFound
	}

	return script, nil
}
