package handler

import (
	"context"
	"testing"
	"time"

	"github.com/escalopa/prayer-bot/domain"
)

func TestTomorrowReminder_ShouldTrigger(t *testing.T) {
	midnight := time.Date(2026, 4, 7, 22, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		now    time.Time
		lastAt time.Time
		offset time.Duration
		want   bool
	}{
		{
			name:   "trigger when now past offset and lastAt before",
			now:    midnight,
			lastAt: midnight.Add(-2 * time.Hour),
			offset: 3 * time.Hour,
			want:   true,
		},
		{
			name:   "no trigger when now before deadline",
			now:    midnight.Add(-2 * time.Hour),
			lastAt: midnight.Add(-3 * time.Hour),
			offset: 1 * time.Hour,
			want:   false,
		},
		{
			name:   "no trigger when already sent",
			now:    midnight,
			lastAt: midnight.Add(30 * time.Minute),
			offset: 3 * time.Hour,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chat := &domain.Chat{
				Reminder: &domain.Reminder{
					Tomorrow: &domain.ReminderConfig{
						Offset: domain.Duration(tt.offset),
						LastAt: tt.lastAt,
					},
				},
			}
			should, _ := (&TomorrowReminder{}).ShouldTrigger(context.Background(), chat, nil, tt.now)
			if should != tt.want {
				t.Fatalf("should trigger = %v, want %v", should, tt.want)
			}
		})
	}
}

func TestJamaatReminder_ShouldTrigger(t *testing.T) {
	base := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)
	prayerDay := &domain.PrayerDay{
		Fajr:    base.Add(5 * time.Hour),
		Dhuhr:   base.Add(12 * time.Hour),
		Asr:     base.Add(15 * time.Hour),
		Maghrib: base.Add(18 * time.Hour),
		Isha:    base.Add(20 * time.Hour),
	}

	tests := []struct {
		name     string
		now      time.Time
		lastAt   time.Time
		delays   map[domain.PrayerID]time.Duration
		enabled  bool
		want     bool
		prayerID domain.PrayerID
	}{
		{
			name:    "trigger after delay",
			now:     prayerDay.Asr.Add(15 * time.Minute),
			lastAt:  base,
			enabled: true,
			delays: map[domain.PrayerID]time.Duration{
				domain.PrayerIDAsr: 10 * time.Minute,
			},
			want:     true,
			prayerID: domain.PrayerIDAsr,
		},
		{
			name:    "does not trigger when already sent",
			now:     prayerDay.Maghrib.Add(12 * time.Minute),
			lastAt:  prayerDay.Maghrib.Add(13 * time.Minute),
			enabled: true,
			delays: map[domain.PrayerID]time.Duration{
				domain.PrayerIDMaghrib: 10 * time.Minute,
			},
			want: false,
		},
		{
			name:    "skips when disabled",
			now:     prayerDay.Dhuhr.Add(11 * time.Minute),
			lastAt:  base,
			enabled: false,
			delays: map[domain.PrayerID]time.Duration{
				domain.PrayerIDDhuhr: 10 * time.Minute,
			},
			want: false,
		},
		{
			name:    "skips when delay is zero",
			now:     prayerDay.Fajr.Add(25 * time.Minute),
			lastAt:  base,
			enabled: true,
			delays: map[domain.PrayerID]time.Duration{
				domain.PrayerIDFajr: 0,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chat := &domain.Chat{
			Reminder: &domain.Reminder{
				Jamaat: &domain.JamaatConfig{
					Enabled: tt.enabled,
					Delay:   &domain.JamaatDelayConfig{},
					State: &domain.ReminderConfig{
						LastAt: tt.lastAt,
					},
				},
			},
			}
			for prayerID, delay := range tt.delays {
				chat.Reminder.Jamaat.Delay.SetDelayByPrayerID(prayerID, delay)
			}

			should, prayerID := (&JamaatReminder{}).ShouldTrigger(context.Background(), chat, prayerDay, tt.now)
			if should != tt.want {
				t.Fatalf("should trigger = %v, want %v", should, tt.want)
			}
			if tt.want && prayerID != tt.prayerID {
				t.Fatalf("prayer id = %v, want %v", prayerID, tt.prayerID)
			}
		})
	}
}
