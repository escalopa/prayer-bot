package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLanguageRepository(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	lr := NewLanguageRepository()

	tests := []struct {
		id   int
		lang string
	}{
		{1, "en"},
		{2, "ar"},
		{3, "ru"},
	}

	for _, tt := range tests {
		err := lr.SetLang(ctx, tt.id, tt.lang)
		if err != nil {
			t.Errorf("failed to set language: %v", err)
		}

		lang, err := lr.GetLang(ctx, tt.id)
		if err != nil {
			t.Errorf("failed to get language: %v", err)
		}

		if lang != tt.lang {
			t.Errorf("expected %s, got %s", tt.lang, lang)
		}
	}

	cancel()

	for _, tt := range tests {
		// Set language
		err := lr.SetLang(ctx, tt.id, tt.lang)
		require.Error(t, err, "expected error, got nil")
		// Get language
		lang, err := lr.GetLang(ctx, tt.id)
		require.Error(t, err, "expected error, got nil")
		require.Equal(t, "", lang, "expected empty string, got %s", lang)
	}
}
