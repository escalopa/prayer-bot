package memory

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/escalopa/gopray/pkg/core"
)

func TestPrayerRepository(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pr := NewPrayerRepository()

	tests := []struct {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store prayer
			err := pr.StorePrayer(ctx, tt.prayer)
			require.NoError(t, err, "failed to store prayer")
			// Get prayer
			p, err := pr.GetPrayer(ctx, tt.prayer.Day, tt.prayer.Month)
			require.NoError(t, err, "failed to get prayer")
			// Compare
			require.Equal(t, tt.prayer, p, "prayer not equal")
		})
	}

	// Test cancel
	cancel()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store prayer
			err := pr.StorePrayer(ctx, tt.prayer)
			require.Error(t, err, "error expected")
			// Get prayer
			p, err := pr.GetPrayer(ctx, tt.prayer.Day, tt.prayer.Month)
			require.Error(t, err, "error expected")
			// Compare
			require.NotEqual(t, tt.prayer, p, "prayer equal")
		})
	}
}
