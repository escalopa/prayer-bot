package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHistoryRepository(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		userID    int
		messageID int
	}{
		{
			name:      "default",
			userID:    1,
			messageID: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				ctx = context.Background()
				hr  = NewHistoryRepository()
			)

			// Get message id
			messageID, err := hr.GetPrayerMessageID(ctx, tt.userID)
			require.Error(t, err)
			require.Equal(t, 0, messageID)

			// Store message id
			err = hr.StorePrayerMessageID(ctx, tt.userID, tt.messageID)
			require.NoError(t, err)

			// Re-get message id
			messageID, err = hr.GetPrayerMessageID(ctx, tt.userID)
			require.NoError(t, err)
			require.Equal(t, tt.messageID, messageID)
		})
	}
}
