package core

import "testing"

func TestIsValidLang(t *testing.T) {
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
			if got := IsValidLang(tt.l); got != tt.want {
				t.Errorf("IsValidLang() = %v, want %v", got, tt.want)
			}
		})
	}
}
