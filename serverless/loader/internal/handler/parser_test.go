package handler

import (
	"testing"
	"time"

	"github.com/escalopa/prayer-bot/domain"
)

func TestParseDate(t *testing.T) {
	want := domain.DateUTC(1, time.January, 2024)
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{name: "unpadded both", input: "1/1/2024", want: want},
		{name: "padded both", input: "01/01/2024", want: want},
		{name: "day unpadded month padded", input: "1/01/2024", want: want},
		{name: "day padded month unpadded", input: "01/1/2024", want: want},
		{name: "invalid", input: "not-a-date", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDate(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}
