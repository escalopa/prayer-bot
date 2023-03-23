package memory

import (
	"context"

	"github.com/escalopa/gopray/pkg/language"
	"github.com/pkg/errors"
)

type ScriptRepository struct {
	scripts map[string]*language.Script
}

func NewScriptRepository() *ScriptRepository {
	return &ScriptRepository{
		scripts: make(map[string]*language.Script),
	}
}

func (r *ScriptRepository) StoreScript(ctx context.Context, language string, script *language.Script) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	r.scripts[language] = script
	return nil
}

func (r *ScriptRepository) GetScript(ctx context.Context, language string) (*language.Script, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	script, ok := r.scripts[language]
	if !ok {
		return nil, errors.New("script not found")
	}
	return script, nil
}
