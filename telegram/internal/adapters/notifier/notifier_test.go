package notifier

import (
	"math"
	"testing"
	"time"
)

func TestNotifier_CalculateTimeLeft(t *testing.T) {
	now, err := now()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name               string
		prayerAfter        time.Time
		upcomingReminder   uint
		expectedUpcomingAt time.Duration
		expectedStartsAt   time.Duration
		expectedStartIn    uint
	}{
		{
			name:               "Prayer is about to start, but the reminder is smaller than prayer time",
			prayerAfter:        now.Add(15 * time.Minute),
			upcomingReminder:   10,
			expectedUpcomingAt: 5 * time.Minute,
			expectedStartsAt:   10 * time.Minute,
			expectedStartIn:    10,
		},
		{
			name:               "Prayer is about to start, but reminder is bigger than prayer time",
			prayerAfter:        now.Add(15 * time.Minute),
			upcomingReminder:   20,
			expectedUpcomingAt: 0 * time.Minute,
			expectedStartsAt:   15 * time.Minute,
			expectedStartIn:    15,
		},
		{
			name:               "Prayer is about to start, but reminder is equal to prayer time",
			prayerAfter:        now.Add(15 * time.Minute),
			upcomingReminder:   15,
			expectedUpcomingAt: 0 * time.Minute,
			expectedStartsAt:   15 * time.Minute,
			expectedStartIn:    15,
		},
		{
			name:               "Prayer is about to start, but reminder is so small that it's equal to 0",
			prayerAfter:        now.Add(15 * time.Minute),
			upcomingReminder:   1,
			expectedUpcomingAt: 14 * time.Minute,
			expectedStartsAt:   1 * time.Minute,
			expectedStartIn:    1,
		},
		{
			name:               "Prayer is about to start, in 0 minutes",
			prayerAfter:        now.Add(0 * time.Minute),
			upcomingReminder:   10,
			expectedUpcomingAt: 0 * time.Minute,
			expectedStartsAt:   0 * time.Minute,
			expectedStartIn:    0,
		},
		{
			name:               "Prayer is about to start, in long time",
			prayerAfter:        now.Add(10 * time.Hour),
			upcomingReminder:   20,
			expectedUpcomingAt: 9*time.Hour + 40*time.Minute,
			expectedStartsAt:   20 * time.Minute,
			expectedStartIn:    20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := New(nil, nil, nil, tt.upcomingReminder)
			upcomingAt, startsAt, startsIn := n.calculateLeftTime(tt.prayerAfter)
			// We compoare the time difference with 1 minute because the time difference due to the `now`` function.
			if upcomingAt != tt.expectedUpcomingAt && time.Duration(math.Abs(float64(upcomingAt-tt.expectedUpcomingAt))) != 1*time.Minute {
				t.Errorf("Expected upcomingAt to be %v, got %v", tt.expectedUpcomingAt, upcomingAt)
			}
			if startsAt != tt.expectedStartsAt && time.Duration(math.Abs(float64(startsAt-tt.expectedStartsAt))) != 1*time.Minute {
				t.Errorf("Expected startsAt to be %v, got %v", tt.expectedStartsAt, startsAt)
			}
			if startsIn != tt.expectedStartIn && tt.expectedStartIn-startsIn != 1 {
				t.Errorf("Expected startsIn to be %v, got %v", tt.expectedStartIn, startsIn)
			}
		})
	}
}
