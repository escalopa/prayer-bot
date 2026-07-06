package handler

import (
	"context"
	"testing"
	"time"

	"github.com/escalopa/prayer-bot/domain"
)

// testLoc keeps the trigger math deterministic and independent of the host TZ.
var testLoc = time.UTC

// at returns 2024-01-15 hh:mm in testLoc.
func at(hour, minute int) time.Time {
	return time.Date(2024, time.January, 15, hour, minute, 0, 0, testLoc)
}

// nextDayAt returns 2024-01-16 hh:mm in testLoc.
func nextDayAt(hour, minute int) time.Time {
	return time.Date(2024, time.January, 16, hour, minute, 0, 0, testLoc)
}

// testPrayerDay is the schedule used across the reminder tests: a full day
// 2024-01-15 with its following day 2024-01-16 attached as NextDay.
func testPrayerDay() *domain.PrayerDay {
	day := &domain.PrayerDay{
		Date:    at(0, 0),
		Fajr:    at(5, 0),
		Shuruq:  at(6, 30),
		Dhuhr:   at(12, 0),
		Asr:     at(15, 0),
		Maghrib: at(18, 0),
		Isha:    at(20, 0),
	}
	day.NextDay = &domain.PrayerDay{
		Date:    nextDayAt(0, 0),
		Fajr:    nextDayAt(5, 0),
		Shuruq:  nextDayAt(6, 30),
		Dhuhr:   nextDayAt(12, 0),
		Asr:     nextDayAt(15, 0),
		Maghrib: nextDayAt(18, 0),
		Isha:    nextDayAt(20, 0),
	}
	return day
}

func TestArriveReminder_ShouldTrigger(t *testing.T) {
	t.Parallel()

	r := &ArriveReminder{}
	pd := testPrayerDay()

	tests := []struct {
		name     string
		lastAt   time.Time
		now      time.Time
		wantSend bool
		wantID   domain.PrayerID
	}{
		{"before first prayer", at(0, 0), at(4, 59), false, domain.PrayerIDUnknown},
		{"exactly at fajr", at(0, 0), at(5, 0), true, domain.PrayerIDFajr},
		{"midday picks latest passed prayer", at(0, 0), at(12, 0), true, domain.PrayerIDDhuhr},
		{"already sent at dhuhr is not resent", at(12, 0), at(12, 5), false, domain.PrayerIDUnknown},
		{"downtime skips stale and sends latest", at(6, 0), at(15, 30), true, domain.PrayerIDAsr},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			chat := &domain.Chat{Reminder: &domain.Reminder{
				Arrive: &domain.ReminderConfig{LastAt: tt.lastAt},
			}}

			gotSend, gotID := r.ShouldTrigger(context.Background(), chat, pd, tt.now)
			if gotSend != tt.wantSend || gotID != tt.wantID {
				t.Fatalf("ShouldTrigger() = (%v, %v), want (%v, %v)", gotSend, gotID, tt.wantSend, tt.wantID)
			}
		})
	}
}

func TestSoonReminder_ShouldTrigger(t *testing.T) {
	t.Parallel()

	r := &SoonReminder{}
	pd := testPrayerDay()
	const offset = 15 * time.Minute

	tests := []struct {
		name     string
		lastAt   time.Time
		now      time.Time
		wantSend bool
		wantID   domain.PrayerID
	}{
		{"before fajr trigger window", at(0, 0), at(4, 44), false, domain.PrayerIDUnknown},
		{"exactly at fajr trigger", at(0, 0), at(4, 45), true, domain.PrayerIDFajr},
		{"midday picks latest triggered prayer", at(0, 0), at(11, 45), true, domain.PrayerIDDhuhr},
		{"next day fajr near midnight", at(21, 0), nextDayAt(4, 45), true, domain.PrayerIDFajr},
		{"already sent is not resent", at(11, 45), at(11, 45), false, domain.PrayerIDUnknown},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			chat := &domain.Chat{Reminder: &domain.Reminder{
				Soon: &domain.ReminderConfig{LastAt: tt.lastAt, Offset: domain.Duration(offset)},
			}}

			gotSend, gotID := r.ShouldTrigger(context.Background(), chat, pd, tt.now)
			if gotSend != tt.wantSend || gotID != tt.wantID {
				t.Fatalf("ShouldTrigger() = (%v, %v), want (%v, %v)", gotSend, gotID, tt.wantSend, tt.wantID)
			}
		})
	}
}

func TestTomorrowReminder_ShouldTrigger(t *testing.T) {
	t.Parallel()

	r := &TomorrowReminder{}
	const offset = 3 * time.Hour // trigger is 3h before next midnight, i.e. 21:00 today

	tests := []struct {
		name     string
		lastAt   time.Time
		now      time.Time
		wantSend bool
	}{
		{"before trigger", at(0, 0), at(20, 59), false},
		{"exactly at trigger", at(0, 0), at(21, 0), true},
		{"after trigger not yet sent", at(0, 0), at(22, 0), true},
		{"already sent today", at(21, 0), at(22, 0), false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			chat := &domain.Chat{Reminder: &domain.Reminder{
				Tomorrow: &domain.ReminderConfig{LastAt: tt.lastAt, Offset: domain.Duration(offset)},
			}}

			gotSend, gotID := r.ShouldTrigger(context.Background(), chat, nil, tt.now)
			if gotSend != tt.wantSend {
				t.Fatalf("ShouldTrigger() send = %v, want %v", gotSend, tt.wantSend)
			}
			if gotID != domain.PrayerIDUnknown {
				t.Fatalf("ShouldTrigger() id = %v, want %v", gotID, domain.PrayerIDUnknown)
			}
		})
	}
}

func TestPrayerTimeByID(t *testing.T) {
	t.Parallel()

	pd := testPrayerDay()

	tests := []struct {
		name     string
		prayerID domain.PrayerID
		now      time.Time
		want     time.Time
	}{
		{"upcoming fajr returns today's time", domain.PrayerIDFajr, at(4, 0), at(5, 0)},
		{"passed fajr rolls to next day", domain.PrayerIDFajr, at(6, 0), nextDayAt(5, 0)},
		{"dhuhr has no next day and stays current", domain.PrayerIDDhuhr, at(13, 0), at(12, 0)},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := prayerTimeByID(pd, tt.prayerID, tt.now); !got.Equal(tt.want) {
				t.Fatalf("prayerTimeByID() = %v, want %v", got, tt.want)
			}
		})
	}
}
