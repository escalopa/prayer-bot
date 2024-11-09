package memory

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubscriberRepository(t *testing.T) {
	t.Parallel()

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
			name:   "duplicated",
			input:  []int{1, 1, 2, 2, 3, 3},
			output: []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				err error

				ctx = context.Background()
				sr  = NewSubscriberRepository()
			)

			// Store subscriber
			for _, id := range tt.input {
				err = sr.StoreSubscriber(ctx, id)
				require.NoError(t, err)
			}

			// Get subscribers
			ids, err := sr.GetSubscribers(ctx)

			sort.Ints(ids) // sort the ids to make sure the order is correct for comparison

			require.NoError(t, err)
			require.Equal(t, tt.output, ids)

			// Remove subscriber
			for _, id := range tt.input {
				err = sr.RemoveSubscribe(ctx, id)
				require.NoError(t, err)
			}

			// Get subscribers
			ids, err = sr.GetSubscribers(ctx)
			require.NoError(t, err)
			require.Empty(t, ids)
		})
	}
}
