package application

import (
	"testing"
)

func TestPrayer_ParserDate(t *testing.T) {
	tests := []struct {
		name      string
		date      string
		wantDay   int
		wantMonth int
		ok        bool
	}{
		{
			name:      "Valid date with /",
			date:      "25/10",
			wantDay:   25,
			wantMonth: 10,
			ok:        true,
		}, {
			name:      "Valid date with -",
			date:      "09-10",
			wantDay:   9,
			wantMonth: 10,
			ok:        true,
		}, {
			name:      "Valid date with .",
			date:      "09.10",
			wantDay:   9,
			wantMonth: 10,
			ok:        true,
		}, {
			name:      "Inalid date with |",
			date:      "09|10",
			wantDay:   0,
			wantMonth: 0,
			ok:        false,
		}, {
			name:      "Invalid date with //",
			date:      "//",
			wantDay:   0,
			wantMonth: 0,
			ok:        false,
		}, {
			name:      "Invalid date",
			date:      "09/10/2020",
			wantDay:   0,
			wantMonth: 0,
			ok:        false,
		}, {
			name:      "Invalid day 0",
			date:      "00-10",
			wantDay:   0,
			wantMonth: 0,
			ok:        false,
		}, {
			name:      "Invalid day 32",
			date:      "32-10",
			wantDay:   0,
			wantMonth: 0,
			ok:        false,
		}, {
			name:      "Invalid month 0",
			date:      "09-00",
			wantDay:   0,
			wantMonth: 0,
			ok:        false,
		}, {
			name:      "Invalid month 13",
			date:      "09-13",
			wantDay:   0,
			wantMonth: 0,
			ok:        false,
		}, {
			name:      "Invalid day and month",
			date:      "32-13",
			wantDay:   0,
			wantMonth: 0,
			ok:        false,
		}, {
			name:      "Invalid combination between day and month on May",
			date:      "31-04",
			wantDay:   0,
			wantMonth: 0,
			ok:        false,
		}, {
			name:      "Invalid combination between day and month on June",
			date:      "31-06",
			wantDay:   0,
			wantMonth: 0,
			ok:        false,
		}, {
			name:      "Invalid combination between day and month on September",
			date:      "31-09",
			wantDay:   0,
			wantMonth: 0,
			ok:        false,
		}, {
			name:      "Invalid combination between day and month on November",
			date:      "31-11",
			wantDay:   0,
			wantMonth: 0,
			ok:        false,
		}, {
			name:      "Invalid combination between day and month on February",
			date:      "29-02",
			wantDay:   0,
			wantMonth: 0,
			ok:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			day, month, ok := parseDate(tt.date)
			if day != tt.wantDay {
				t.Errorf("got %d, want %d", day, tt.wantDay)
			}
			if month != tt.wantMonth {
				t.Errorf("got %d, want %d", month, tt.wantMonth)
			}
			if ok != tt.ok {
				t.Errorf("got %t, want %t", ok, tt.ok)
			}
		})
	}
}
