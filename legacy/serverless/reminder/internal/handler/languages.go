package handler

import (
	_ "embed"

	"gopkg.in/yaml.v3"
)

type (
	Text struct {
		Prayer          map[int]string `yaml:"prayer"` // domain.PrayerID to prayer name
		PrayerSoon      string         `yaml:"prayer_soon"`
		PrayerArrived   string         `yaml:"prayer_arrived"`
		PrayerJoin      string         `yaml:"prayer_join"`
		PrayerJoinDelay string         `yaml:"prayer_join_delay"`
		PrayerJamaat    string         `yaml:"prayer_jamaat"`
	}

	languagesProvider struct {
		storage map[string]*Text
	}
)

//go:embed languages/text.yaml
var textData string

func newLanguagesProvider() (*languagesProvider, error) {
	var storage map[string]*Text
	err := yaml.Unmarshal([]byte(textData), &storage)
	if err != nil {
		return nil, err
	}
	return &languagesProvider{storage: storage}, nil
}

func (p *languagesProvider) GetText(languageCode string) *Text { return p.storage[languageCode] }
