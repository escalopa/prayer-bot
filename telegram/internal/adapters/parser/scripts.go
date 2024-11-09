package parser

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	app "github.com/escalopa/gopray/telegram/internal/application"
	"github.com/escalopa/gopray/telegram/internal/domain"
	"github.com/pkg/errors"
)

type ScriptParser struct {
	ctx context.Context

	path string
	scr  app.ScriptRepository
}

func NewScriptParser(path string, scr app.ScriptRepository) *ScriptParser {
	return &ScriptParser{
		ctx:  context.Background(),
		path: path,
		scr:  scr,
	}
}

func (p *ScriptParser) Parse() error {
	return filepath.Walk(p.path, p.process)
}

func (p *ScriptParser) process(path string, info os.FileInfo, err error) error {
	if err != nil {
		return errors.Wrap(err, "error reading script file")
	}

	// check if it's a file and has json extension, otherwise skip
	if info.IsDir() || filepath.Ext(path) != ".json" {
		return nil
	}

	fileName := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))

	// read file
	date, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// unmarshal json
	var script domain.Script
	err = json.Unmarshal(date, &script)
	if err != nil {
		return err
	}

	// save script
	err = p.scr.StoreScript(p.ctx, fileName, &script)
	if err != nil {
		return err
	}

	log.Printf("successfully parsed script: %s", fileName)

	return nil
}
