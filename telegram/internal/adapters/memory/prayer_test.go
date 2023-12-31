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
		prayer core.PrayerTime
	}{
		{
			name: "Test 1",
			prayer: core.NewPrayerTime(
				time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), // Day
				time.Now().Add(1*time.Hour),                 // Fajr
				time.Now().Add(2*time.Hour),                 // Sunrise
				time.Now().Add(3*time.Hour),                 // Dhuhr
				time.Now().Add(4*time.Hour),                 // Asr
				time.Now().Add(5*time.Hour),                 // Maghrib
				time.Now().Add(6*time.Hour),                 // Isha
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store prayer
			err := pr.StorePrayer(ctx, tt.prayer)
			require.NoError(t, err, "failed to store prayer")
			// Get prayer
			p, err := pr.GetPrayer(ctx, tt.prayer.Day)
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
			p, err := pr.GetPrayer(ctx, tt.prayer.Day)
			require.Error(t, err, "error expected")
			// Compare
			require.NotEqual(t, tt.prayer, p, "prayer equal")
		})
	}
}
