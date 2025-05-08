package internal

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type (
	Month struct {
		ID   int
		Name string
	}

	Language struct {
		Code string
		Name string
	}

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

		Remind   InteractiveMessage `yaml:"remind"`
		Language InteractiveMessage `yaml:"language"`

		SubscriptionSuccess   string `yaml:"subscription_success"`
		UnsubscriptionSuccess string `yaml:"unsubscription_success"`

		PrayerSoon    string `yaml:"prayer_soon"`
		PrayerArrived string `yaml:"prayer_arrived"`

		Help string `yaml:"help"`

		Feedback InteractiveMessage `yaml:"feedback"`
		Bug      InteractiveMessage `yaml:"bug"`

		HelpAdmin string             `yaml:"help_admin"`
		Reply     InteractiveMessage `yaml:"reply"`
		Announce  InteractiveMessage `yaml:"announce"`
		Stats     string             `yaml:"stats"`

		Cancel string `yaml:"cancel"`
		Noop   string `yaml:"noop"`
		Error  string `yaml:"error"`
	}

	languagesProvider struct {
		storage map[string]*Text
	}
)

func (t *Text) GetMonths() []Month {
	months := make([]Month, 0, len(t.Month))
	for number, name := range t.Month {
		months = append(months, Month{
			ID:   number,
			Name: name,
		})
	}

	sort.Slice(months, func(i, j int) bool {
		return months[i].ID < months[j].ID
	})

	return months
}

func newLanguageProvider() (*languagesProvider, error) {
	const pattern = "internal/languages/*.yaml" // relative to the `main.go` directory (source root)

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

func (p *languagesProvider) GetText(languageCode string) *Text { return p.storage[languageCode] }

func (p *languagesProvider) IsSupportedCode(languageCode string) bool {
	_, ok := p.storage[languageCode]
	return ok
}

func (p *languagesProvider) GetLanguages() []Language {
	languages := make([]Language, 0, len(p.storage))
	for code, text := range p.storage {
		languages = append(languages, Language{
			Code: code,
			Name: text.Name,
		})
	}

	sort.Slice(languages, func(i, j int) bool {
		return languages[i].Code < languages[j].Code
	})

	return languages
}
