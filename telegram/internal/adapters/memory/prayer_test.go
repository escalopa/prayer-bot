package memory

import (
	"context"
	"testing"
	"time"

	"github.com/escalopa/gopray/telegram/internal/domain"

	"github.com/stretchr/testify/require"
)

func TestPrayerRepository(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name   string
		prayer *domain.PrayerTime
	}{
		{
			name: "default",
			prayer: domain.NewPrayerTime(
				now,
				now.Add(1*time.Hour),
				now.Add(2*time.Hour),
				now.Add(3*time.Hour),
				now.Add(4*time.Hour),
				now.Add(5*time.Hour),
				now.Add(6*time.Hour),
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
