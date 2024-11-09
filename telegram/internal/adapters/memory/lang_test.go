package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLanguageRepository(t *testing.T) {
	t.Parallel()

	const (
		defaultLang = "en"
	)

	tests := []struct {
		name string
		id   int
		lang string
	}{
		{
			name: "default",
			id:   1,
			lang: "ar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				ctx = context.Background()
				lr  = NewLanguageRepository()
			)

			// GetLang
			lang, err := lr.GetLang(ctx, tt.id)
			require.NoError(t, err)
			require.Equal(t, defaultLang, lang)

			// SetLang
			err = lr.SetLang(ctx, tt.id, tt.lang)
			require.NoError(t, err)

			// GetLang
			lang, err = lr.GetLang(ctx, tt.id)
			require.NoError(t, err)
			require.Equal(t, tt.lang, lang)
		})
	}
}
