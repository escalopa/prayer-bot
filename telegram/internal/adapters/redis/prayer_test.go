package redis

import (
	"context"
	"testing"
	"time"

	"github.com/escalopa/gopray/pkg/core"
	"github.com/stretchr/testify/require"
)

func TestPrayerRepository(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pr := NewPrayerRepository(New(testRedisURL))

	tests := []struct {
		name  string
		day   int
		month int
		times core.PrayerTimes
	}{
		{
			name:  "test prayer times",
			day:   1,
			month: 1,
			times: core.New(1, 1,
				time.Now().Add(2*time.Second),
				time.Now().Add(3*time.Second),
				time.Now().Add(4*time.Second),
				time.Now().Add(5*time.Second),
				time.Now().Add(6*time.Second),
				time.Now().Add(7*time.Second)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store prayer times
			err := pr.StorePrayer(ctx, tt.times)
			require.NoError(t, err, "expected no error, got %v", err)
			// Get prayer times
			got, err := pr.GetPrayer(ctx, tt.day, tt.month)
			require.NoError(t, err, "expected no error, got %v", err)
			// Compare times
			require.WithinDurationf(t, tt.times.Fajr, got.Fajr, 1*time.Second, "expected %v, got %v", tt.times.Fajr, got.Fajr)
			require.WithinDurationf(t, tt.times.Sunrise, got.Sunrise, 1*time.Second, "expected %v, got %v", tt.times.Sunrise, got.Sunrise)
			require.WithinDurationf(t, tt.times.Dhuhr, got.Dhuhr, 1*time.Second, "expected %v, got %v", tt.times.Dhuhr, got.Dhuhr)
			require.WithinDurationf(t, tt.times.Asr, got.Asr, 1*time.Second, "expected %v, got %v", tt.times.Asr, got.Asr)
			require.WithinDurationf(t, tt.times.Maghrib, got.Maghrib, 1*time.Second, "expected %v, got %v", tt.times.Maghrib, got.Maghrib)
			require.WithinDurationf(t, tt.times.Isha, got.Isha, 1*time.Second, "expected %v, got %v", tt.times.Isha, got.Isha)
		})
	}

	cancel()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pr.StorePrayer(ctx, tt.times)
			require.Error(t, err, "expected error, got nil")
			got, err := pr.GetPrayer(ctx, tt.day, tt.month)
			require.Error(t, err, "expected error, got nil")
			require.Emptyf(t, got, "expected empty PrayerTimes, got %v", got)
		})
	}
}
