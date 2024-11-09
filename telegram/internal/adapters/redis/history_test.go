package redis

import (
	"testing"

	"github.com/escalopa/gopray/telegram/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestHistoryRepository(t *testing.T) {
	t.Parallel()

	client, errRedis := New(testRedisURL)
	require.NoError(t, errRedis)

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

			hr := NewHistoryRepository(client, tt.name)

			ctx, cancel := testContext()

			// Get message id
			messageID, err := hr.GetPrayerMessageID(ctx, tt.userID)
			require.Empty(t, messageID)
			require.ErrorIs(t, err, domain.ErrNotFound)

			// Store message id
			err = hr.StorePrayerMessageID(ctx, tt.userID, tt.messageID)
			require.NoError(t, err)

			// Get message id
			messageID, err = hr.GetPrayerMessageID(ctx, tt.userID)
			require.NoError(t, err)
			require.Equal(t, 1, messageID)

			cancel()

			// Store message id
			err = hr.StorePrayerMessageID(ctx, tt.userID, tt.messageID)
			require.Error(t, err)

			// Get message id
			_, err = hr.GetPrayerMessageID(ctx, tt.userID)
			require.Error(t, err)
		})
	}
}
