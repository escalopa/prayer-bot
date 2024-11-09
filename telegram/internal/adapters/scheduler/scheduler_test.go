package scheduler

import (
	"testing"
	"time"

	"github.com/escalopa/gopray/telegram/internal/adapters/memory"
	"github.com/stretchr/testify/require"
)

func TestNotifierCalculateTimeLeft(t *testing.T) {
	t.Parallel()

	const (
		jummahReminder = 10 * time.Hour
	)

	tests := []struct {
		name               string
		prayerAfter        time.Duration
		upcomingReminder   time.Duration
		expectedUpcomingAt time.Duration
		expectedStartsAt   time.Duration
		expectedStartIn    int
	}{
		{
			name:               "prayer_is_about_to_start_but_the_reminder_is_smaller_than_prayer time",
			prayerAfter:        15 * time.Minute,
			upcomingReminder:   10 * time.Minute,
			expectedUpcomingAt: 5 * time.Minute,
			expectedStartsAt:   10 * time.Minute,
			expectedStartIn:    10,
		},
		{
			name:               "prayer_is_about_to_start_but_reminder_is_bigger_than_prayer_time",
			prayerAfter:        15 * time.Minute,
			upcomingReminder:   20 * time.Minute,
			expectedUpcomingAt: 0 * time.Minute,
			expectedStartsAt:   15 * time.Minute,
			expectedStartIn:    15,
		},
		{
			name:               "prayer_is_about_to_start_but_reminder_is_equal_to_prayer_time",
			prayerAfter:        15 * time.Minute,
			upcomingReminder:   15 * time.Minute,
			expectedUpcomingAt: 0 * time.Minute,
			expectedStartsAt:   15 * time.Minute,
			expectedStartIn:    15,
		},
		{
			name:               "prayer_is_about_to_start_but_reminder_is_so_small_that_it's_equal_to_0",
			prayerAfter:        15 * time.Minute,
			upcomingReminder:   1 * time.Minute,
			expectedUpcomingAt: 14 * time.Minute,
			expectedStartsAt:   1 * time.Minute,
			expectedStartIn:    1,
		},
		{
			name:               "prayer_is_about_to_start_in_0_minutes",
			prayerAfter:        0 * time.Minute,
			upcomingReminder:   10 * time.Minute,
			expectedUpcomingAt: 0 * time.Minute,
			expectedStartsAt:   0 * time.Minute,
			expectedStartIn:    0,
		},
		{
			name:               "prayer_is_about_to_start_in_long_time",
			prayerAfter:        10 * time.Hour,
			upcomingReminder:   20 * time.Minute,
			expectedUpcomingAt: 9*time.Hour + 40*time.Minute,
			expectedStartsAt:   20 * time.Minute,
			expectedStartIn:    20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.upcomingReminder, jummahReminder, time.UTC,
				memory.NewPrayerRepository(),
				memory.NewSubscriberRepository(),
			)

			now := s.now()

			upcomingAt, startsAt := s.timeLeft(now.Add(tt.prayerAfter))

			// We compare the time difference with 1 minute because the time difference due to the `now`` function.
			require.WithinDuration(t, now.Add(tt.expectedUpcomingAt), now.Add(upcomingAt), time.Second)
			require.WithinDuration(t, now.Add(tt.expectedStartsAt), now.Add(startsAt), time.Second)
		})
	}
}
