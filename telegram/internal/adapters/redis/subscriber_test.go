package redis

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSubscriberRepository(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	sr := NewSubscriberRepository(New(testRedisURL))
	require.NotNil(t, sr)

	tests := []struct {
		name string
		ids  []int
	}{
		{
			name: "Test 1",
			ids:  []int{1, 2, 3},
		},
		{
			name: "Test 2",
			ids:  []int{4, 5, 6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store subscriber
			for _, id := range tt.ids {
				err := sr.StoreSubscriber(ctx, id)
				require.NoError(t, err, "expected no error, got %v", err)
			}
			// Get subscribers
			ids, err := sr.GetSubscribers(ctx)
			require.NoError(t, err, "expected no error, got %v", err)
			require.Equal(t, tt.ids, ids, "expected %v, got %v", tt.ids, ids)
			// Remove subscriber
			for _, id := range tt.ids {
				err = sr.RemoveSubscribe(ctx, id)
				require.NoError(t, err, "expected no error, got %v", err)
			}
		})
	}

	// Test cancel
	cancel()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store subscriber
			for _, id := range tt.ids {
				err := sr.StoreSubscriber(ctx, id)
				require.Error(t, err, "expected error, got nil")
			}
			// Get subscribers
			ids, err := sr.GetSubscribers(ctx)
			require.Error(t, err, "expected error, got nil")
			require.Empty(t, ids, "expected empty slice, got %v", ids)
			// Remove subscriber
			for _, id := range tt.ids {
				err = sr.RemoveSubscribe(ctx, id)
				require.Error(t, err, "expected error, got nil")
			}
		})
	}
}
