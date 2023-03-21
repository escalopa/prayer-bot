package redis

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHistoryRepository(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h := NewHistoryRepository(New(testRedisURL))

	tests := []struct {
		name      string
		userID    int
		messageID int
	}{
		{
			name:      "store and get prayer message id",
			userID:    1,
			messageID: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Default prayers
			// Get message id
			messageID, err := h.GetPrayerMessageID(ctx, tt.userID)
			require.Error(t, err)
			require.Equal(t, 0, messageID)
			// Store message id
			err = h.StorePrayerMessageID(ctx, tt.userID, tt.messageID)
			require.NoError(t, err)
			// Re-get message id
			messageID, err = h.GetPrayerMessageID(ctx, tt.userID)
			require.NoError(t, err)
			require.Equal(t, 1, messageID)
		})
	}

	// Test 0 user id
	t.Run("store and get prayer message id with 0 user id", func(t *testing.T) {
		// Default prayers
		// Get message id
		messageID, err := h.GetPrayerMessageID(ctx, 0)
		require.Error(t, err)
		require.Equal(t, 0, messageID)
		// Store message id
		err = h.StorePrayerMessageID(ctx, 0, 1)
		require.Error(t, err)
		// Re-get message id
		messageID, err = h.GetPrayerMessageID(ctx, 0)
		require.Error(t, err)
		require.Equal(t, 0, messageID)
	})

	// Test cancel
	cancel()

	for _, tt := range tests {
		t.Run(tt.name+"_Cancel", func(t *testing.T) {
			// Default prayers
			// Get message id
			messageID, err := h.GetPrayerMessageID(ctx, tt.userID)
			require.Error(t, err)
			require.Equal(t, 0, messageID)
			// Store message id
			err = h.StorePrayerMessageID(ctx, tt.userID, tt.messageID)
			require.Error(t, err)
			// Re-get message id
			messageID, err = h.GetPrayerMessageID(ctx, tt.userID)
			require.Error(t, err)
			require.Equal(t, 0, messageID)
		})
	}

}
