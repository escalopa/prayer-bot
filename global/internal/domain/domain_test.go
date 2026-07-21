package domain

import (
	"testing"
	"time"
)

func validProfile() PrayerProfile {
	return PrayerProfile{
		Latitude:         30.0,
		Longitude:        31.0,
		Timezone:         "Africa/Cairo",
		Method:           MethodEgyptian,
		Madhab:           MadhabShafii,
		HighLatitudeRule: HighLatitudeAngleBased,
		HijriAdjustment:  0,
	}
}

func TestPrayerProfileValidateAcceptsAWellFormedProfile(t *testing.T) {
	if err := validProfile().Validate(); err != nil {
		t.Fatalf("expected a valid profile, got %v", err)
	}
}

func TestPrayerProfileValidateRejectsEachInvalidField(t *testing.T) {
	tests := map[string]func(*PrayerProfile){
		"latitude below range":  func(p *PrayerProfile) { p.Latitude = -91 },
		"latitude above range":  func(p *PrayerProfile) { p.Latitude = 91 },
		"longitude below range": func(p *PrayerProfile) { p.Longitude = -181 },
		"longitude above range": func(p *PrayerProfile) { p.Longitude = 181 },
		"blank timezone":        func(p *PrayerProfile) { p.Timezone = "   " },
		"unknown timezone":      func(p *PrayerProfile) { p.Timezone = "Mars/Olympus" },
		"unsupported method":    func(p *PrayerProfile) { p.Method = "made_up" },
		"unsupported madhab":    func(p *PrayerProfile) { p.Madhab = "made_up" },
		"unsupported highlat":   func(p *PrayerProfile) { p.HighLatitudeRule = "made_up" },
		"hijri below range":     func(p *PrayerProfile) { p.HijriAdjustment = -3 },
		"hijri above range":     func(p *PrayerProfile) { p.HijriAdjustment = 3 },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			profile := validProfile()
			mutate(&profile)
			if err := profile.Validate(); err == nil {
				t.Fatalf("expected %s to be rejected", name)
			}
		})
	}
}

func TestPrayerProfileValidateAcceptsHijriBoundaries(t *testing.T) {
	for _, adjustment := range []int{-2, -1, 0, 1, 2} {
		profile := validProfile()
		profile.HijriAdjustment = adjustment
		if err := profile.Validate(); err != nil {
			t.Fatalf("hijri adjustment %d should be valid: %v", adjustment, err)
		}
	}
}

func TestRoundedCoordinatesLimitsToThreeDecimals(t *testing.T) {
	tests := []struct {
		lat, lon         float64
		wantLat, wantLon float64
	}{
		{55.1234567, 49.9876543, 55.123, 49.988},
		{-33.86785, 151.20732, -33.868, 151.207},
		{0, 0, 0, 0},
	}
	for _, tc := range tests {
		gotLat, gotLon := RoundedCoordinates(tc.lat, tc.lon)
		if gotLat != tc.wantLat || gotLon != tc.wantLon {
			t.Fatalf("RoundedCoordinates(%v, %v) = (%v, %v), want (%v, %v)",
				tc.lat, tc.lon, gotLat, gotLon, tc.wantLat, tc.wantLon)
		}
	}
}

func TestMethodValidCoversEverySupportedMethod(t *testing.T) {
	for _, method := range SupportedMethods() {
		if !method.Valid() {
			t.Fatalf("supported method %q reported invalid", method)
		}
	}
	if Method("nonsense").Valid() {
		t.Fatal("unknown method reported valid")
	}
	if len(SupportedMethods()) != 9 {
		t.Fatalf("expected 9 supported methods, got %d", len(SupportedMethods()))
	}
}

func TestMadhabAndHighLatitudeRuleValidation(t *testing.T) {
	for _, madhab := range []Madhab{MadhabShafii, MadhabHanafi} {
		if !madhab.Valid() {
			t.Fatalf("madhab %q should be valid", madhab)
		}
	}
	if Madhab("maliki").Valid() {
		t.Fatal("unsupported madhab reported valid")
	}
	for _, rule := range []HighLatitudeRule{HighLatitudeAngleBased, HighLatitudeMiddleNight, HighLatitudeSeventhNight} {
		if !rule.Valid() {
			t.Fatalf("high-latitude rule %q should be valid", rule)
		}
	}
	if HighLatitudeRule("none").Valid() {
		t.Fatal("unsupported high-latitude rule reported valid")
	}
}

func TestPrayerValid(t *testing.T) {
	for _, prayer := range []Prayer{PrayerFajr, PrayerSunrise, PrayerDhuhr, PrayerAsr, PrayerMaghrib, PrayerIsha} {
		if !prayer.Valid() {
			t.Fatalf("prayer %q should be valid", prayer)
		}
	}
	if Prayer("tahajjud").Valid() {
		t.Fatal("unknown prayer reported valid")
	}
}

func TestReminderKindClassifiers(t *testing.T) {
	weekly := []ReminderKind{ReminderWeeklyFasting, ReminderWeeklyKahf}
	occasion := []ReminderKind{ReminderOccasionMajor, ReminderOccasionFasting, ReminderOccasionObserved}
	neither := []ReminderKind{ReminderBefore, ReminderAt, ReminderTomorrow}

	for _, kind := range weekly {
		if !kind.Weekly() || kind.Occasion() {
			t.Fatalf("%q must classify as weekly only", kind)
		}
	}
	for _, kind := range occasion {
		if !kind.Occasion() || kind.Weekly() {
			t.Fatalf("%q must classify as occasion only", kind)
		}
	}
	for _, kind := range neither {
		if kind.Weekly() || kind.Occasion() {
			t.Fatalf("%q must be neither weekly nor occasion", kind)
		}
	}
}

func TestValidPreReminderMinutes(t *testing.T) {
	for _, minutes := range SupportedPreReminderMinutes() {
		if !ValidPreReminderMinutes(minutes) {
			t.Fatalf("supported pre-reminder %d reported invalid", minutes)
		}
	}
	for _, minutes := range []int{-5, 1, 7, 25, 90} {
		if ValidPreReminderMinutes(minutes) {
			t.Fatalf("unsupported pre-reminder %d reported valid", minutes)
		}
	}
}

func TestDayScheduleAt(t *testing.T) {
	at := time.Date(2026, time.July, 20, 5, 12, 0, 0, time.UTC)
	schedule := DaySchedule{Times: map[Prayer]time.Time{
		PrayerFajr: at,
		PrayerAsr:  {}, // present but zero: treated as absent
	}}

	if got, ok := schedule.At(PrayerFajr); !ok || !got.Equal(at) {
		t.Fatalf("At(fajr) = (%s, %v), want (%s, true)", got, ok, at)
	}
	if _, ok := schedule.At(PrayerAsr); ok {
		t.Fatal("a zero prayer time must be reported as absent")
	}
	if _, ok := schedule.At(PrayerIsha); ok {
		t.Fatal("a missing prayer must be reported as absent")
	}
}
