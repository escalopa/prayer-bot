package redis

import (
	"testing"
	"time"

	"github.com/escalopa/gopray/telegram/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestPrayerRepository(t *testing.T) {
	t.Parallel()

	client, errRedis := New(testRedisURL)
	require.NoError(t, errRedis)

	now := time.Now()

	tests := []struct {
		name  string
		day   time.Time
		times *domain.PrayerTime
	}{
		{
			name: "default",
			day:  now,
			times: domain.NewPrayerTime(now,
				now.Add(2*time.Second),
				now.Add(3*time.Second),
				now.Add(4*time.Second),
				now.Add(5*time.Second),
				now.Add(6*time.Second),
				now.Add(7*time.Second)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pr := NewPrayerRepository(client, tt.name)

			ctx, cancel := testContext()

			// Get prayer times
			got, err := pr.GetPrayer(ctx, tt.day)
			require.Empty(t, got)
			require.ErrorIs(t, err, domain.ErrNotFound)

			// Store prayer times
			err = pr.StorePrayer(ctx, tt.times)
			require.NoError(t, err)

			// Get prayer times
			got, err = pr.GetPrayer(ctx, tt.day)
			require.NoError(t, err)

			// Compare times
			const errorFormat = "expected %v | got %v"

			require.WithinDurationf(t, tt.times.Fajr, got.Fajr, 1*time.Second, errorFormat, tt.times.Fajr, got.Fajr)
			require.WithinDurationf(t, tt.times.Dohaa, got.Dohaa, 1*time.Second, errorFormat, tt.times.Dohaa, got.Dohaa)
			require.WithinDurationf(t, tt.times.Dhuhr, got.Dhuhr, 1*time.Second, errorFormat, tt.times.Dhuhr, got.Dhuhr)
			require.WithinDurationf(t, tt.times.Asr, got.Asr, 1*time.Second, errorFormat, tt.times.Asr, got.Asr)
			require.WithinDurationf(t, tt.times.Maghrib, got.Maghrib, 1*time.Second, errorFormat, tt.times.Maghrib, got.Maghrib)
			require.WithinDurationf(t, tt.times.Isha, got.Isha, 1*time.Second, errorFormat, tt.times.Isha, got.Isha)

			cancel()

			// Store prayer times
			err = pr.StorePrayer(ctx, tt.times)
			require.Error(t, err)

			// Get prayer times
			_, err = pr.GetPrayer(ctx, tt.day)
			require.Error(t, err)
		})
	}
}
