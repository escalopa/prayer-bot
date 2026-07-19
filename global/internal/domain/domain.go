package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

type Chat struct {
	TelegramChatID int64
	Type           string
	LanguageCode   string
	BlockedAt      *time.Time
}

type Method string

const (
	MethodMWL       Method = "mwl"
	MethodEgyptian  Method = "egyptian"
	MethodUmmAlQura Method = "umm_al_qura"
	MethodKarachi   Method = "karachi"
	MethodISNA      Method = "isna"
	MethodDiyanet   Method = "diyanet"
	MethodKemenag   Method = "kemenag"
	MethodMUIS      Method = "muis"
	MethodJAKIM     Method = "jakim"
)

func (m Method) Valid() bool {
	switch m {
	case MethodMWL, MethodEgyptian, MethodUmmAlQura, MethodKarachi,
		MethodISNA, MethodDiyanet, MethodKemenag, MethodMUIS, MethodJAKIM:
		return true
	default:
		return false
	}
}

func SupportedMethods() []Method {
	return []Method{
		MethodMWL, MethodEgyptian, MethodUmmAlQura, MethodKarachi,
		MethodISNA, MethodDiyanet, MethodKemenag, MethodMUIS, MethodJAKIM,
	}
}

type Madhab string

const (
	MadhabShafii Madhab = "shafii"
	MadhabHanafi Madhab = "hanafi"
)

func (m Madhab) Valid() bool { return m == MadhabShafii || m == MadhabHanafi }

type HighLatitudeRule string

const (
	HighLatitudeAngleBased   HighLatitudeRule = "angle_based"
	HighLatitudeMiddleNight  HighLatitudeRule = "middle_of_night"
	HighLatitudeSeventhNight HighLatitudeRule = "one_seventh"
)

func (r HighLatitudeRule) Valid() bool {
	return r == HighLatitudeAngleBased || r == HighLatitudeMiddleNight || r == HighLatitudeSeventhNight
}

type Adjustments struct {
	Fajr    int `json:"fajr"`
	Sunrise int `json:"sunrise"`
	Dhuhr   int `json:"dhuhr"`
	Asr     int `json:"asr"`
	Maghrib int `json:"maghrib"`
	Isha    int `json:"isha"`
}

type PrayerProfile struct {
	ChatID           int64
	Latitude         float64
	Longitude        float64
	Timezone         string
	PlaceID          string
	LocationLabel    string // Only a user-supplied label may be persisted here.
	Method           Method
	Madhab           Madhab
	HighLatitudeRule HighLatitudeRule
	Adjustments      Adjustments
	HijriAdjustment  int
	Version          int64
	UpdatedAt        time.Time
}

func (p PrayerProfile) Validate() error {
	if p.Latitude < -90 || p.Latitude > 90 || p.Longitude < -180 || p.Longitude > 180 {
		return fmt.Errorf("invalid coordinates")
	}
	if strings.TrimSpace(p.Timezone) == "" {
		return fmt.Errorf("timezone is required")
	}
	if !p.Method.Valid() {
		return fmt.Errorf("unsupported method %q", p.Method)
	}
	if !p.Madhab.Valid() {
		return fmt.Errorf("unsupported madhab %q", p.Madhab)
	}
	if !p.HighLatitudeRule.Valid() {
		return fmt.Errorf("unsupported high-latitude rule %q", p.HighLatitudeRule)
	}
	if p.HijriAdjustment < -2 || p.HijriAdjustment > 2 {
		return fmt.Errorf("hijri adjustment must be between -2 and 2")
	}
	_, err := time.LoadLocation(p.Timezone)
	return err
}

// RoundedCoordinates limits stored precision to roughly a city block. Raw
// Telegram coordinates should only live long enough to resolve the timezone.
func RoundedCoordinates(latitude, longitude float64) (float64, float64) {
	return math.Round(latitude*1000) / 1000, math.Round(longitude*1000) / 1000
}

type Prayer string

const (
	PrayerFajr    Prayer = "fajr"
	PrayerSunrise Prayer = "sunrise"
	PrayerDhuhr   Prayer = "dhuhr"
	PrayerAsr     Prayer = "asr"
	PrayerMaghrib Prayer = "maghrib"
	PrayerIsha    Prayer = "isha"
)

func (p Prayer) Valid() bool {
	switch p {
	case PrayerFajr, PrayerSunrise, PrayerDhuhr, PrayerAsr, PrayerMaghrib, PrayerIsha:
		return true
	default:
		return false
	}
}

type DaySchedule struct {
	Date     time.Time
	Timezone string
	Times    map[Prayer]time.Time
}

func (s DaySchedule) At(prayer Prayer) (time.Time, bool) {
	t, ok := s.Times[prayer]
	return t, ok && !t.IsZero()
}

type ReminderKind string

const (
	ReminderBefore           ReminderKind = "before"
	ReminderAt               ReminderKind = "at"
	ReminderTomorrow         ReminderKind = "tomorrow"
	ReminderWeeklyFasting    ReminderKind = "weekly_fasting"
	ReminderWeeklyKahf       ReminderKind = "weekly_kahf"
	ReminderOccasionMajor    ReminderKind = "occasion_major"
	ReminderOccasionFasting  ReminderKind = "occasion_fasting"
	ReminderOccasionObserved ReminderKind = "occasion_observed"
)

func (kind ReminderKind) Weekly() bool {
	return kind == ReminderWeeklyFasting || kind == ReminderWeeklyKahf
}

func (kind ReminderKind) Occasion() bool {
	return kind == ReminderOccasionMajor ||
		kind == ReminderOccasionFasting ||
		kind == ReminderOccasionObserved
}

type ReminderRule struct {
	ID            int64
	ChatID        int64
	Kind          ReminderKind
	Prayer        Prayer
	OffsetMinutes int
	LocalTime     string
	Enabled       bool
}

func SupportedPreReminderMinutes() []int {
	return []int{0, 5, 10, 15, 20, 30, 45, 60}
}

func ValidPreReminderMinutes(value int) bool {
	for _, candidate := range SupportedPreReminderMinutes() {
		if value == candidate {
			return true
		}
	}
	return false
}

type ReminderSchedule struct {
	ID             int64
	RuleID         int64
	ChatID         int64
	ProfileVersion int64
	LocalDate      string
	PrayerAt       time.Time
	NextRunAt      time.Time
	State          string
}

type DeliveryTask struct {
	DeliveryKey    string    `json:"delivery_key"`
	ScheduleID     int64     `json:"schedule_id"`
	RuleID         int64     `json:"rule_id"`
	ChatID         int64     `json:"chat_id"`
	ProfileVersion int64     `json:"profile_version"`
	ScheduledFor   time.Time `json:"scheduled_for"`
}

type MessageDeletionTask struct {
	DeletionKey string `json:"deletion_key"`
	ChatID      int64  `json:"chat_id"`
	MessageID   int64  `json:"message_id"`
}

type CalendarSubscription struct {
	ChatID       int64
	FeedToken    string
	UIDNamespace string
	Enabled      bool
}
