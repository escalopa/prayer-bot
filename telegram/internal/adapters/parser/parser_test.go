package parser

import (
	"testing"
	"time"
)

func TestParser_ConvertToTime(t *testing.T) {
	type Input struct {
		day   int
		month int
		time  string
	}

	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Errorf("time.LoadLocation() error = %v, wantErr %v", err, false)
		return
	}
	tests := []struct {
		name  string
		input Input
		want  time.Time
	}{
		{
			name:  "Test 1",
			input: Input{day: 1, month: 1, time: "05:30"},
			want:  time.Date(time.Now().Year(), 1, 1, 5, 30, 0, 0, loc),
		},
		{
			name:  "Test 2",
			input: Input{day: 4, month: 5, time: "15:35"},
			want:  time.Date(time.Now().Year(), 5, 4, 15, 35, 0, 0, loc),
		},
		{
			name:  "Test 3",
			input: Input{day: 31, month: 12, time: "23:59"},
			want:  time.Date(time.Now().Year(), 12, 31, 23, 59, 0, 0, loc),
		},
		{
			name:  "Test 4",
			input: Input{day: 1, month: 1, time: "00:00"},
			want:  time.Date(time.Now().Year(), 1, 1, 0, 0, 0, 0, loc),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertToTime(tt.input.time, tt.input.day, tt.input.month)
			if err != nil {
				t.Errorf("convertToTime() error = %v, wantErr %v", err, false)
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("convertToTime() got = %v, want %v", got, tt.want)
			}
		})
	}
}
