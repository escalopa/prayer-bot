package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidLang(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		l    string
		want bool
	}{
		{
			name: "ar",
			l:    "ar",
			want: true,
		},
		{
			name: "en",
			l:    "en",
			want: true,
		},
		{
			name: "ru",
			l:    "ru",
			want: true,
		},
		{
			name: "fr",
			l:    "fr",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, IsValidLang(tt.l))
		})
	}
}
