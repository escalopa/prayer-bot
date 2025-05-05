package internal

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type languagesProvider struct {
	storage map[string]*Text
}

func (p *languagesProvider) GetText(languageCode string) *Text { return p.storage[languageCode] }

func (p *languagesProvider) IsValidCode(languageCode string) bool {
	_, ok := p.storage[languageCode]
	return ok
}

// GetValues returns a map of language codes to their names.
func (p *languagesProvider) GetValues() map[string]string {
	values := make(map[string]string)
	for code, text := range p.storage {
		values[code] = text.Name
	}
	return values
}

func newLanguageProvider() (*languagesProvider, error) {
	const pattern = "languages/*.yaml"

	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	storage := make(map[string]*Text, len(files))
	for _, file := range files {
		languageCode := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}

		var text Text
		if err := yaml.Unmarshal(content, &text); err != nil {
			return nil, err
		}

		storage[languageCode] = &text
	}

	return &languagesProvider{storage: storage}, nil
}

type (
	InteractiveMessage struct {
		Start   string `yaml:"start"`
		Success string `yaml:"success"`
	}

	Text struct {
		Name string `yaml:"name"`

		Weekday map[int]string `yaml:"weekday"` // time.Weekday to weekday name
		Month   map[int]string `yaml:"month"`   // time.Month to month name
		Prayer  map[int]string `yaml:"prayer"`  // domain.PrayerID to prayer name

		PrayerDate string `yaml:"prayer_date"`

		NotifyOffset InteractiveMessage `yaml:"notify_offset"`
		Language     InteractiveMessage `yaml:"language"`

		SubscriptionSuccess   string `yaml:"subscription_success"`
		UnsubscriptionSuccess string `yaml:"unsubscription_success"`

		PrayerSoon    string `yaml:"prayer_soon"`
		PrayerArrived string `yaml:"prayer_arrived"`

		Help string `yaml:"help"`

		Feedback  InteractiveMessage `yaml:"feedback"`
		BugReport InteractiveMessage `yaml:"bug_report"`

		HelpAdmin string             `yaml:"help_admin"`
		Reply     InteractiveMessage `yaml:"reply"`
		Announce  InteractiveMessage `yaml:"announce"`
		Stats     string             `yaml:"stats"`

		Cancel string `yaml:"cancel"`
		Error  string `yaml:"error"`
	}
)
