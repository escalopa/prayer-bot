package i18n

import (
	"strings"

	"github.com/escalopa/prayer-bot/global/internal/domain"
)

const (
	ActionToday     = "today"
	ActionTomorrow  = "tomorrow"
	ActionNext      = "next"
	ActionLocation  = "location"
	ActionSettings  = "settings"
	ActionReminders = "reminders"
	ActionLanguage  = "language"
	ActionHelp      = "help"
)

var mainActions = []string{
	ActionToday, ActionTomorrow, ActionNext, ActionLocation,
	ActionSettings, ActionReminders, ActionLanguage, ActionHelp,
}

// Locale contains all user-visible copy for one Telegram language.
type Locale struct {
	Code             string
	NativeName       string
	BotName          string
	ShortDescription string
	Description      string
	Commands         map[string]string
	Buttons          map[string]string
	Text             map[string]string
	Prayers          map[domain.Prayer]string
	Methods          map[domain.Method]string
	Madhabs          map[domain.Madhab]string
	HighLatitude     map[domain.HighLatitudeRule]string
	Months           []string
}

func (l Locale) Button(action string) string { return l.Buttons[action] }

func (l Locale) Message(key string) string {
	if value := l.Text[key]; value != "" {
		return value
	}
	return locales["en"].Text[key]
}

func (l Locale) Prayer(prayer domain.Prayer) string {
	if value := l.Prayers[prayer]; value != "" {
		return value
	}
	return locales["en"].Prayers[prayer]
}

func (l Locale) Method(method domain.Method) string {
	if value := l.Methods[method]; value != "" {
		return value
	}
	return locales["en"].Methods[method]
}

func (l Locale) Madhab(madhab domain.Madhab) string {
	if value := l.Madhabs[madhab]; value != "" {
		return value
	}
	return locales["en"].Madhabs[madhab]
}

func (l Locale) HighLatitudeRule(rule domain.HighLatitudeRule) string {
	if value := l.HighLatitude[rule]; value != "" {
		return value
	}
	return locales["en"].HighLatitude[rule]
}

func (l Locale) Month(number int) string {
	if number >= 1 && number <= len(l.Months) {
		return l.Months[number-1]
	}
	return ""
}

// Resolve accepts Telegram language tags such as ar-EG or pt_BR and returns a
// supported locale, falling back to English.
func Resolve(languageCode string) Locale {
	code := strings.ToLower(strings.TrimSpace(languageCode))
	if index := strings.IndexAny(code, "-_"); index >= 0 {
		code = code[:index]
	}
	if locale, ok := locales[code]; ok {
		return locale
	}
	return locales["en"]
}

func Supported() []Locale {
	codes := []string{"en", "ar", "es", "fr", "ru", "tr", "uz", "tt"}
	result := make([]Locale, 0, len(codes))
	for _, code := range codes {
		result = append(result, locales[code])
	}
	return result
}

// ActionForText maps every localized persistent-keyboard label back to a
// stable action. This also keeps keyboards sent before a language change useful.
func ActionForText(text string) string {
	text = strings.TrimSpace(text)
	for _, locale := range locales {
		for _, action := range mainActions {
			if text == locale.Button(action) {
				return action
			}
		}
	}
	return ""
}
