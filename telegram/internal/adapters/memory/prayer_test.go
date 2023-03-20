package memory

import (
	"context"
	"testing"
	"time"

	"github.com/escalopa/gopray/pkg/core"
	"github.com/stretchr/testify/require"
)

func TestPrayerRepository(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pr := NewPrayerRepository()

	testCases := []struct {
		name   string
		prayer core.PrayerTimes
	}{
		{
			name: "Test 1",
			prayer: core.PrayerTimes{
				Day:     1,
				Month:   1,
				Fajr:    time.Now().Add(1 * time.Hour),
				Dhuhr:   time.Now().Add(2 * time.Hour),
				Asr:     time.Now().Add(3 * time.Hour),
				Maghrib: time.Now().Add(4 * time.Hour),
				Isha:    time.Now().Add(5 * time.Hour),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Store prayer
			err := pr.StorePrayer(ctx, tc.prayer)
			require.NoError(t, err, "failed to store prayer")
			// Get prayer
			p, err := pr.GetPrayer(ctx, tc.prayer.Day, tc.prayer.Month)
			require.NoError(t, err, "failed to get prayer")
			// Compare
			require.Equal(t, tc.prayer, p, "prayer not equal")
		})
	}

	// Test cancel
	cancel()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Store prayer
			err := pr.StorePrayer(ctx, tc.prayer)
			require.Error(t, err, "error expected")
			// Get prayer
			p, err := pr.GetPrayer(ctx, tc.prayer.Day, tc.prayer.Month)
			require.Error(t, err, "error expected")
			// Compare
			require.NotEqual(t, tc.prayer, p, "prayer equal")
		})
	}
}
