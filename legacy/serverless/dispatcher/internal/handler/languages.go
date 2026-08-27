package handler

import (
	"embed"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed languages/*.yaml
var languageFiles embed.FS

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

	RemindMenuText struct {
		TitleEnabled   string `yaml:"title_enabled"`
		TitleDisabled  string `yaml:"title_disabled"`
		Enable         string `yaml:"enable"`
		Disable        string `yaml:"disable"`
		Tomorrow       string `yaml:"tomorrow"`
		Soon           string `yaml:"soon"`
		JamaatSettings string `yaml:"jamaat_settings"`
		Close          string `yaml:"close"`
	}

	RemindEditText struct {
		TitleTomorrow string `yaml:"title_tomorrow"`
		TitleSoon     string `yaml:"title_soon"`
	}

	JamaatMenuText struct {
		TitleEnabled  string `yaml:"title_enabled"`
		TitleDisabled string `yaml:"title_disabled"`
		Enable        string `yaml:"enable"`
		Disable       string `yaml:"disable"`
	}

	JamaatEditText struct {
		Title string `yaml:"title"`
	}

	ButtonsText struct {
		Save string `yaml:"save"`
		Back string `yaml:"back"`
	}

	InfoTypeText struct {
		Private string `yaml:"private"`
		Group   string `yaml:"group"`
	}

	InfoText struct {
		Default  string       `yaml:"default"`
		Jamaat   string       `yaml:"jamaat"`
		Type     InfoTypeText `yaml:"type"`
		Enabled  string       `yaml:"enabled"`
		Disabled string       `yaml:"disabled"`
	}

	Text struct {
		Name string `yaml:"name"`

		Weekday map[int]string `yaml:"weekday"` // time.Weekday to weekday name
		Month   map[int]string `yaml:"month"`   // time.Month to month name
		Prayer  map[int]string `yaml:"prayer"`  // domain.PrayerID to prayer name

		PrayerDate string `yaml:"prayer_date"`

		Remind   InteractiveMessage `yaml:"remind"`
		Language InteractiveMessage `yaml:"language"`

		RemindMenu RemindMenuText `yaml:"remind_menu"`
		RemindEdit RemindEditText `yaml:"remind_edit"`
		JamaatMenu JamaatMenuText `yaml:"jamaat_menu"`
		JamaatEdit JamaatEditText `yaml:"jamaat_edit"`
		Buttons    ButtonsText    `yaml:"buttons"`
		Info       InfoText       `yaml:"info"`

		PrayerSoon string `yaml:"prayer_soon"`

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
	entries, err := languageFiles.ReadDir("languages")
	if err != nil {
		return nil, fmt.Errorf("read embedded languages: %w", err)
	}

	storage := make(map[string]*Text, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		languageCode := strings.TrimSuffix(entry.Name(), ".yaml")
		content, err := languageFiles.ReadFile("languages/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read language %q: %w", languageCode, err)
		}

		var text Text
		if err := yaml.Unmarshal(content, &text); err != nil {
			return nil, fmt.Errorf("parse language %q: %w", languageCode, err)
		}

		storage[languageCode] = &text
	}

	if len(storage) == 0 {
		return nil, fmt.Errorf("no language files embedded")
	}
	if storage[defaultLanguageCode] == nil {
		return nil, fmt.Errorf("default language %q not embedded", defaultLanguageCode)
	}

	return &languagesProvider{storage: storage}, nil
}

func (p *languagesProvider) GetText(languageCode string) *Text {
	if text := p.storage[languageCode]; text != nil {
		return text
	}
	return p.storage[defaultLanguageCode]
}

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
