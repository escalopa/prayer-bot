package redis

import (
	"testing"

	"github.com/escalopa/gopray/telegram/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestNewLanguageRepository(t *testing.T) {
	t.Parallel()

	client, errRedis := New(testRedisURL)
	require.NoError(t, errRedis)

	tests := []struct {
		name string
		id   int
		lang string
	}{
		{
			name: "default",
			id:   1,
			lang: "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lr := NewLanguageRepository(client, tt.name)

			ctx, cancel := testContext()

			// Get language
			lang, err := lr.GetLang(ctx, tt.id)
			require.Empty(t, lang)
			require.ErrorIs(t, err, domain.ErrNotFound)

			// Set language
			err = lr.SetLang(ctx, tt.id, tt.lang)
			require.NoError(t, err)

			// Get language
			lang, err = lr.GetLang(ctx, tt.id)
			require.NoError(t, err)
			require.Equal(t, tt.lang, lang)

			cancel()

			// Set language
			err = lr.SetLang(ctx, tt.id, tt.lang)
			require.Error(t, err)

			// Get language
			_, err = lr.GetLang(ctx, tt.id)
			require.Error(t, err)
		})
	}
}
