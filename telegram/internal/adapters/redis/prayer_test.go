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
		day   time.Time
		times core.PrayerTime
	}{
		{
			name: "test prayer times",
			day:  core.DefaultTime(1, 1, 2023),
			times: core.NewPrayerTime(core.DefaultTime(1, 1, 2023),
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
			require.NoError(t, err, "expected no error but got %v", err)
			// Get prayer times
			got, err := pr.GetPrayer(ctx, tt.day)
			require.NoError(t, err, "expected no error but got %v", err)
			// Compare times

			const errorFormat = "expected %v | got %v"
			require.WithinDurationf(t, tt.times.Fajr, got.Fajr, 1*time.Second, errorFormat, tt.times.Fajr, got.Fajr)
			require.WithinDurationf(t, tt.times.Dohaa, got.Dohaa, 1*time.Second, errorFormat, tt.times.Dohaa, got.Dohaa)
			require.WithinDurationf(t, tt.times.Dhuhr, got.Dhuhr, 1*time.Second, errorFormat, tt.times.Dhuhr, got.Dhuhr)
			require.WithinDurationf(t, tt.times.Asr, got.Asr, 1*time.Second, errorFormat, tt.times.Asr, got.Asr)
			require.WithinDurationf(t, tt.times.Maghrib, got.Maghrib, 1*time.Second, errorFormat, tt.times.Maghrib, got.Maghrib)
			require.WithinDurationf(t, tt.times.Isha, got.Isha, 1*time.Second, errorFormat, tt.times.Isha, got.Isha)
		})
	}

	cancel()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pr.StorePrayer(ctx, tt.times)
			require.Error(t, err, "expected error | got nil")
			got, err := pr.GetPrayer(ctx, tt.day)
			require.Error(t, err, "expected error | got nil")
			require.Emptyf(t, got, "expected empty PrayerTimes, got %v", got)
		})
	}
}
