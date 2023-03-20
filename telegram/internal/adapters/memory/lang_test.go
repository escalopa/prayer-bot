package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLanguageRepository(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	lr := NewLanguageRepository()

	testCases := []struct {
		id   int
		lang string
	}{
		{1, "en"},
		{2, "ar"},
		{3, "fr"},
		{4, "en"},
		{5, "ar"},
	}

	for _, tc := range testCases {
		err := lr.SetLang(ctx, tc.id, tc.lang)
		if err != nil {
			t.Errorf("failed to set language: %v", err)
		}

		lang, err := lr.GetLang(ctx, tc.id)
		if err != nil {
			t.Errorf("failed to get language: %v", err)
		}

		if lang != tc.lang {
			t.Errorf("expected %s, got %s", tc.lang, lang)
		}
	}

	cancel()

	for _, tc := range testCases {
		// Set language
		err := lr.SetLang(ctx, tc.id, tc.lang)
		require.Error(t, err, "expected error, got nil")
		// Get language
		lang, err := lr.GetLang(ctx, tc.id)
		require.Error(t, err, "expected error, got nil")
		require.Equal(t, "", lang, "expected empty string, got %s", lang)
	}
}
