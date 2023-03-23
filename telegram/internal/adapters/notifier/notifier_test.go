package notifier

import (
	"testing"
	"time"

	"github.com/escalopa/gopray/telegram/internal/adapters/memory"
	"github.com/stretchr/testify/require"
)

func TestNotifier_New(t *testing.T) {
	loc := time.FixedZone("UTC", 0)
	pr := memory.NewPrayerRepository()
	lr := memory.NewLanguageRepository()
	sr := memory.NewSubscriberRepository()

	// Nil location
	var err error
	_, err = New(30*time.Minute, 11*time.Hour, WithTimeLocation(nil), WithPrayerRepository(pr), WithLanguageRepository(lr), WithSubscriberRepository(sr))
	require.Error(t, err, "expected error got nil")
	// Nil prayer repository
	_, err = New(30*time.Minute, 11*time.Hour, WithTimeLocation(loc), WithPrayerRepository(nil), WithLanguageRepository(lr), WithSubscriberRepository(sr))
	require.Error(t, err, "expected error got nil")
	// Nil language repository
	_, err = New(30*time.Minute, 11*time.Hour, WithTimeLocation(loc), WithPrayerRepository(pr), WithLanguageRepository(nil), WithSubscriberRepository(sr))
	require.Error(t, err, "expected error got nil")
	// Nil subscriber repository
	_, err = New(30*time.Minute, 11*time.Hour, WithTimeLocation(loc), WithPrayerRepository(pr), WithLanguageRepository(lr), WithSubscriberRepository(nil))
	require.Error(t, err, "expected error got nil")

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
			_, err = New(time.Duration(tt.ur)*time.Minute, time.Duration(tt.gnh)*time.Hour,
				WithTimeLocation(loc),
				WithPrayerRepository(pr),
				WithLanguageRepository(lr),
				WithSubscriberRepository(sr))
			require.Truef(t, (err != nil) == tt.wantErr, "New() error = %v, wantErr %v", err, tt.wantErr)
		})
	}
}

func TestNotifier_CalculateTimeLeft(t *testing.T) {
	tests := []struct {
		name               string
		prayerAfter        time.Duration
		upcomingReminder   time.Duration
		expectedUpcomingAt time.Duration
		expectedStartsAt   time.Duration
		expectedStartIn    int
	}{
		{
			name:               "Prayer is about to start, but the reminder is smaller than prayer time",
			prayerAfter:        15 * time.Minute,
			upcomingReminder:   10 * time.Minute,
			expectedUpcomingAt: 5 * time.Minute,
			expectedStartsAt:   10 * time.Minute,
			expectedStartIn:    10,
		},
		{
			name:               "Prayer is about to start, but reminder is bigger than prayer time",
			prayerAfter:        15 * time.Minute,
			upcomingReminder:   20 * time.Minute,
			expectedUpcomingAt: 0 * time.Minute,
			expectedStartsAt:   15 * time.Minute,
			expectedStartIn:    15,
		},
		{
			name:               "Prayer is about to start, but reminder is equal to prayer time",
			prayerAfter:        15 * time.Minute,
			upcomingReminder:   15 * time.Minute,
			expectedUpcomingAt: 0 * time.Minute,
			expectedStartsAt:   15 * time.Minute,
			expectedStartIn:    15,
		},
		{
			name:               "Prayer is about to start, but reminder is so small that it's equal to 0",
			prayerAfter:        15 * time.Minute,
			upcomingReminder:   1 * time.Minute,
			expectedUpcomingAt: 14 * time.Minute,
			expectedStartsAt:   1 * time.Minute,
			expectedStartIn:    1,
		},
		{
			name:               "Prayer is about to start, in 0 minutes",
			prayerAfter:        0 * time.Minute,
			upcomingReminder:   10 * time.Minute,
			expectedUpcomingAt: 0 * time.Minute,
			expectedStartsAt:   0 * time.Minute,
			expectedStartIn:    0,
		},
		{
			name:               "Prayer is about to start, in long time",
			prayerAfter:        10 * time.Hour,
			upcomingReminder:   20 * time.Minute,
			expectedUpcomingAt: 9*time.Hour + 40*time.Minute,
			expectedStartsAt:   20 * time.Minute,
			expectedStartIn:    20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := time.LoadLocation("Africa/Cairo")
			require.NoError(t, err)
			// Create a notifier with upcoming reminder equal `tt.upcomingReminder` and 1h for gomaa time.
			// We use 1h because it's the least time that we can use.
			n, err := New(tt.upcomingReminder, 1*time.Hour, WithTimeLocation(loc),
				WithPrayerRepository(memory.NewPrayerRepository()),
				WithLanguageRepository(memory.NewLanguageRepository()),
				WithSubscriberRepository(memory.NewSubscriberRepository()))
			require.NoError(t, err)

			now := n.now()
			upcomingAt, startsAt := n.timeLeft(now.Add(tt.prayerAfter))
			// We compare the time difference with 1 minute because the time difference due to the `now`` function.
			require.WithinDuration(t, now.Add(tt.expectedUpcomingAt), now.Add(upcomingAt), time.Second, "expected upcomingAt to be equal")
			require.WithinDuration(t, now.Add(tt.expectedStartsAt), now.Add(startsAt), time.Second, "expected startsAt to be equal")
		})
	}
}
