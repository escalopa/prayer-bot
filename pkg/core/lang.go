package core

import "strings"

type Language string

const (
	AR Language = "ar"
	EN Language = "en"
	RU Language = "ru"
)

func (l Language) String() string {
	return string(l)
}

// IsValidLang returns true if the given language is valid
func IsValidLang(l string) bool {
	switch l {
	case AR.String(), EN.String(), RU.String():
		return true
	}
	return false
}

// AvaliableLanguages returns all the avaliable languages for the application
func AvaliableLanguages() []string {
	return []string{
		strings.ToUpper(EN.String()),
		strings.ToUpper(RU.String()),
		strings.ToUpper(AR.String()),
	}
}

func DefaultLang() string {
	return EN.String()
}
