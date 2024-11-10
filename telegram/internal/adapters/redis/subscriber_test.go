package redis

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSubscriberRepository(t *testing.T) {
	t.Parallel()

	client, errRedis := New(testRedisURL)
	require.NoError(t, errRedis)

	tests := []struct {
		name   string
		input  []int
		output []int
	}{
		{
			name:   "default",
			input:  []int{1, 2, 3},
			output: []int{1, 2, 3},
		},
		{
			name:   "empty",
			input:  []int{},
			output: []int{},
		},
		{
			name:   "duplicate",
			input:  []int{1, 1, 2, 2, 3, 3},
			output: []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sr := NewSubscriberRepository(client, tt.name)

			ctx, cancel := testContext()

			// Get subscribers
			ids, err := sr.GetSubscribers(ctx)
			require.Empty(t, ids)
			require.NoError(t, err)

			// Store subscriber
			for _, id := range tt.input {
				err = sr.StoreSubscriber(ctx, id)
				require.NoError(t, err)
			}

			// Get subscribers
			ids, err = sr.GetSubscribers(ctx)
			require.NoError(t, err)

			sort.Ints(ids)
			require.Equal(t, tt.output, ids)

			// Remove subscriber
			for _, id := range tt.input {
				err = sr.RemoveSubscribe(ctx, id)
				require.NoError(t, err)
			}

			// Get subscribers
			ids, err = sr.GetSubscribers(ctx)
			require.Empty(t, ids)
			require.NoError(t, err)

			cancel()

			// Store subscriber
			for _, id := range tt.input {
				err = sr.StoreSubscriber(ctx, id)
				require.Error(t, err)
			}

			// Get subscribers
			_, err = sr.GetSubscribers(ctx)
			require.Error(t, err)

			// Remove subscriber
			for _, id := range tt.input {
				err = sr.RemoveSubscribe(ctx, id)
				require.Error(t, err)
			}
		})
	}
}
