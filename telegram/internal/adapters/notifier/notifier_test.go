package notifier

import (
	"testing"
	"time"
)

func TestNotifier_CalculateTimeLeft(t *testing.T) {
	tests := []struct {
		name               string
		prayerAfter        uint
		upcomingReminder   uint
		expectedUpcomingAt time.Duration
		expectedStartsAt   time.Duration
	}{
		{
			name:               "Prayer is about to start",
			prayerAfter:        15,
			upcomingReminder:   10,
			expectedUpcomingAt: 5 * time.Minute,
			expectedStartsAt:   15 * time.Minute,
		},
		{
			name:               "Prayer is about to start, but reminder is bigger than prayer time",
			prayerAfter:        15,
			upcomingReminder:   20,
			expectedUpcomingAt: 1 * time.Minute,
			expectedStartsAt:   15 * time.Minute,
		},
		{
			name:               "Prayer is about to start, but reminder is equal to prayer time",
			prayerAfter:        15,
			upcomingReminder:   15,
			expectedUpcomingAt: 1 * time.Minute,
			expectedStartsAt:   15 * time.Minute,
		},
		{
			name:               "Prayer is about to start, but reminder is 0",
			prayerAfter:        15,
			upcomingReminder:   0,
			expectedUpcomingAt: 15 * time.Minute,
			expectedStartsAt:   15 * time.Minute,
		},
		{
			name:               "Prayer is about to start, in 0 minutes",
			prayerAfter:        0,
			upcomingReminder:   10,
			expectedUpcomingAt: 1 * time.Minute,
			expectedStartsAt:   1 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := New(nil, nil, nil, tt.upcomingReminder)
			upcomingAt, startsAt := n.calculateLeftTime(tt.prayerAfter)
			if upcomingAt != tt.expectedUpcomingAt {
				t.Errorf("Expected upcomingAt to be %v, got %v", tt.expectedUpcomingAt, upcomingAt)
			}
			if startsAt != tt.expectedStartsAt {
				t.Errorf("Expected startsAt to be %v, got %v", tt.expectedStartsAt, startsAt)
			}
		})
	}
}
