package core

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
