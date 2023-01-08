package notifier

import (
	"math"
	"testing"
	"time"
)

func TestNotifier_New(t *testing.T) {
	tests := []struct {
		name    string
		ur      int
		gnh     int
		wantErr bool
	}{
		{
			name: "Valid ur and gnh",
			ur:   5,
			gnh:  10,
		}, {
			name:    "Invalid ur -1",
			ur:      -1,
			gnh:     10,
			wantErr: true,
		}, {
			name:    "Invalid ur 0",
			ur:      -1,
			gnh:     10,
			wantErr: true,
		}, {
			name:    "Invalid ur 60",
			ur:      60,
			gnh:     10,
			wantErr: true,
		}, {
			name:    "Invalid ur 61",
			ur:      61,
			gnh:     10,
			wantErr: true,
		}, {
			name:    "Invalid gnh -1",
			ur:      5,
			gnh:     -1,
			wantErr: true,
		}, {
			name:    "Invalid gnh 0",
			ur:      5,
			gnh:     0,
			wantErr: true,
		}, {
			name:    "Invalid gnh 12",
			ur:      5,
			gnh:     12,
			wantErr: true,
		}, {
			name:    "Invalid gnh 13",
			ur:      5,
			gnh:     13,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(nil, nil, nil, tt.ur, tt.gnh)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestNotifier_CalculateTimeLeft(t *testing.T) {
	tests := []struct {
		name               string
		prayerAfter        time.Duration
		upcomingReminder   int
		expectedUpcomingAt time.Duration
		expectedStartsAt   time.Duration
		expectedStartIn    int
	}{
		{
			name:               "Prayer is about to start, but the reminder is smaller than prayer time",
			prayerAfter:        15 * time.Minute,
			upcomingReminder:   10,
			expectedUpcomingAt: 5 * time.Minute,
			expectedStartsAt:   10 * time.Minute,
			expectedStartIn:    10,
		},
		{
			name:               "Prayer is about to start, but reminder is bigger than prayer time",
			prayerAfter:        15 * time.Minute,
			upcomingReminder:   20,
			expectedUpcomingAt: 0 * time.Minute,
			expectedStartsAt:   15 * time.Minute,
			expectedStartIn:    15,
		},
		{
			name:               "Prayer is about to start, but reminder is equal to prayer time",
			prayerAfter:        15 * time.Minute,
			upcomingReminder:   15,
			expectedUpcomingAt: 0 * time.Minute,
			expectedStartsAt:   15 * time.Minute,
			expectedStartIn:    15,
		},
		{
			name:               "Prayer is about to start, but reminder is so small that it's equal to 0",
			prayerAfter:        15 * time.Minute,
			upcomingReminder:   1,
			expectedUpcomingAt: 14 * time.Minute,
			expectedStartsAt:   1 * time.Minute,
			expectedStartIn:    1,
		},
		{
			name:               "Prayer is about to start, in 0 minutes",
			prayerAfter:        0 * time.Minute,
			upcomingReminder:   10,
			expectedUpcomingAt: 0 * time.Minute,
			expectedStartsAt:   0 * time.Minute,
			expectedStartIn:    0,
		},
		{
			name:               "Prayer is about to start, in long time",
			prayerAfter:        10 * time.Hour,
			upcomingReminder:   20,
			expectedUpcomingAt: 9*time.Hour + 40*time.Minute,
			expectedStartsAt:   20 * time.Minute,
			expectedStartIn:    20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := New(nil, nil, nil, tt.upcomingReminder, 7)
			if err != nil {
				t.Fatal(err)
			}
			now, err := n.now()
			if err != nil {
				t.Fatal(err)
			}
			upcomingAt, startsAt, startsIn := n.calculateLeftTime(now.Add(tt.prayerAfter))
			// We compare the time difference with 1 minute because the time difference due to the `now`` function.
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
