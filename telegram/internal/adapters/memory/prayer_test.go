package memory

import (
	"context"
	"testing"
	"time"

	"github.com/escalopa/gopray/pkg/core"
	"github.com/stretchr/testify/require"
)

func TestPrayerRepository(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		prayer *core.PrayerTime
	}{
		{
			name: "default",
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
			t.Parallel()

			var (
				ctx = context.Background()
				pr  = NewPrayerRepository()
			)

			// Get prayer
			p, err := pr.GetPrayer(ctx, tt.prayer.Day)
			require.Error(t, err)
			require.Nil(t, p)

			// Store prayer
			err = pr.StorePrayer(ctx, tt.prayer)
			require.NoError(t, err)

			// Get prayer
			p, err = pr.GetPrayer(ctx, tt.prayer.Day)
			require.NoError(t, err)
			require.Equal(t, tt.prayer, p)
		})
	}
}
