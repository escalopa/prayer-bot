package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubscriberRepository(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sr := NewSubscriberRepository()

	testCases := []struct {
		name string
		id   int
	}{
		{
			name: "Test 1",
			id:   1,
		},
		{
			name: "Test 2",
			id:   2,
		},
		{
			name: "Test 3",
			id:   3,
		},
	}

	var stack []int
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Store subscriber
			err := sr.StoreSubscriber(ctx, tc.id)
			require.NoError(t, err, "failed to store subscriber")
			// Get subscribers
			ids, err := sr.GetSubscribers(ctx)
			require.NoError(t, err, "failed to get subscribers")
			// Compare
			require.Equal(t, len(ids), len(stack)+1, "subscribers length not equal")
			stack = append(stack, tc.id)
			// Remove subscriber
			err = sr.RemoveSubscribe(ctx, tc.id)
			require.NoError(t, err, "failed to remove subscriber")
			// Get subscribers
			ids, err = sr.GetSubscribers(ctx)
			require.NoError(t, err, "failed to get subscribers")
			// Compare
			require.Equal(t, len(ids), len(stack)-1, "subscribers length not equal")
			stack = stack[:len(stack)-1]
		})
	}

	// Test cancel
	cancel()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Store subscriber
			err := sr.StoreSubscriber(ctx, tc.id)
			require.Error(t, err, "expected error, got nil")
			// Get subscribers
			ids, err := sr.GetSubscribers(ctx)
			require.Error(t, err, "expected error, got nil")
			require.Equal(t, 0, len(ids), "expected empty slice, got %v", ids)
			// Remove subscriber
			err = sr.RemoveSubscribe(ctx, tc.id)
			require.Error(t, err, "expected error, got nil")
		})
	}
}
