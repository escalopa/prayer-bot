package language

// Script is a struct that holds all the scripts for each language
type Script struct {
	DataPickerStart string `json:"DataPickerStart"`

	January   string `json:"January"`
	February  string `json:"February"`
	March     string `json:"March"`
	April     string `json:"April"`
	May       string `json:"May"`
	June      string `json:"June"`
	July      string `json:"July"`
	August    string `json:"August"`
	September string `json:"September"`
	October   string `json:"October"`
	November  string `json:"November"`
	December  string `json:"December"`

	LanguageSelectionStart   string `json:"LanguageSelectionStart"`
	LanguageSelectionSuccess string `json:"LanguageSelectionSuccess"`
	LanguageSelectionFail    string `json:"LanguageSelectionFail"`

	Fajr    string `json:"Fajr"`
	Sunrise string `json:"Sunrise"`
	Dhuhr   string `json:"Dhuhr"`
	Asr     string `json:"Asr"`
	Maghrib string `json:"Maghrib"`
	Isha    string `json:"Isha"`

	PrayrifyTableDay    string `json:"PrayrifyTableDay"`
	PrayrifyTablePrayer string `json:"PrayrifyTablePrayer"`
	PrayrifyTableTime   string `json:"PrayrifyTableTime"`
	PrayerFail          string `json:"PrayerFail"`

	SubscriptionSuccess string `json:"SubscriptionSuccess"`
	SubscriptionError   string `json:"SubscriptionError"`

	UnsubscriptionSuccess string `json:"UnsubscriptionSuccess"`
	UnsubscriptionError   string `json:"UnsubscriptionError"`

	PrayerSoon    string `json:"PrayerSoon"`
	PrayerArrived string `json:"PrayerArrived"`
	GomaaDay      string `json:"GomaaDay"`

	Help string `json:"Help"`

	FeedbackStart   string `json:"FeedbackStart"`
	FeedbackSuccess string `json:"FeedbackSuccess"`
	FeedbackFail    string `json:"FeedbackFail"`

	BugReportStart   string `json:"BugReportStart"`
	BugReportSuccess string `json:"BugReportSuccess"`
	BugReportFail    string `json:"BugReportFail"`
}

func (s *Script) GetMonthNames() [12]string {
	return [12]string{
		s.January,
		s.February,
		s.March,
		s.April,
		s.May,
		s.June,
		s.July,
		s.August,
		s.September,
		s.October,
		s.November,
		s.December,
	}
}

///////////////////////////////////////////////////////////////////////////////

// Language is a struct that holds the long and short name of a language
type Language struct {
	Long  string
	Short string
}

var (
	Arabic  = Language{Long: "العربية", Short: "ar"}
	English = Language{Long: "English", Short: "en"}
	Russian = Language{Long: "Русский", Short: "ru"}
)

// IsValidLang returns true if the given language is valid
func IsValidLang(l string) bool {
	switch l {
	case Arabic.Short, English.Short, Russian.Short:
		return true
	}
	return false
}

// AvaliableLanguages returns all the avaliable languages for the application
func AvaliableLanguages() []Language {
	return []Language{Arabic, English, Russian}
}

// DefaultLang returns the default language for the application
func DefaultLang() Language {
	return English
}
