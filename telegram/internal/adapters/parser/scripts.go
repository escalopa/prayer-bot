package parser

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/escalopa/gopray/pkg/language"
	"github.com/escalopa/gopray/telegram/internal/application"
	"github.com/pkg/errors"
)

type ScriptParser struct {
	path string
	sr   application.ScriptRepository
}

func NewScriptParser(path string, opts ...func(*ScriptParser)) *ScriptParser {
	p := &ScriptParser{
		path: path,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func WithScriptRepository(sr application.ScriptRepository) func(*ScriptParser) {
	return func(p *ScriptParser) {
		p.sr = sr
	}
}

func (p *ScriptParser) ParseScripts(ctx context.Context) error {
	log.Printf("parsing scripts from path: %s", p.path)
	// read all scripts from path and set the key to the script name
	err := filepath.Walk(p.path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "error reading script file")
		}
		if !info.IsDir() && filepath.Ext(path) == ".json" {
			fileName := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
			// read file
			date, err := os.ReadFile(path)
			if err != nil {
				return errors.Wrap(err, "error reading script file")
			}
			// unmarshal json
			var script language.Script
			err = json.Unmarshal(date, &script)
			if err != nil {
				return errors.Wrap(err, "error unmarshalling script")
			}
			// save script
			err = p.sr.StoreScript(ctx, fileName, &script)
			if err != nil {
				return errors.Wrap(err, "error saving script")
			}
			log.Printf("successfully parsed script: %s", fileName)
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "error reading scripts")
	}
	return nil
}
