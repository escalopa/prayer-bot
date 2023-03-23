package redis

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLanguageRepository(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	c := New(testRedisURL)
	lr := NewLanguageRepository(c)
	require.NotNil(t, lr)
	defer func() {
		require.NoError(t, c.Close())
	}()

	tests := []struct {
		name string
		id   int
		lang string
	}{
		{
			name: "Test 1",
			id:   1,
			lang: "en",
		},
		{
			name: "Test 2",
			id:   2,
			lang: "ar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := lr.SetLang(ctx, tt.id, tt.lang)
			require.NoError(t, err, "expected no error, got %v", err)
			lang, err := lr.GetLang(ctx, tt.id)
			require.NoError(t, err, "expected no error, got %v", err)
			require.Equal(t, tt.lang, lang, "expected %s, got %s", tt.lang, lang)
		})
	}

	// Test cancel
	cancel()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set language
			err := lr.SetLang(ctx, tt.id, tt.lang)
			require.Error(t, err, "expected error, got nil")
			// Get language
			lang, err := lr.GetLang(ctx, tt.id)
			require.Error(t, err, "expected error, got nil")
			require.Equal(t, "", lang, "expected empty string, got %s", lang)
		})
	}
}
