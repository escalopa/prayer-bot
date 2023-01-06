package handler

import (
	"testing"
)

func TestPrayer_ParserDate(t *testing.T) {
	tests := []struct {
		name string
		date string
		want string
		ok   bool
	}{
		{
			name: "Valid date with /",
			date: "25/10",
			want: "25/10",
			ok:   true,
		}, {
			name: "Valid date with -",
			date: "09-10",
			want: "9/10",
			ok:   true,
		}, {
			name: "Valid date with .",
			date: "09.10",
			want: "9/10",
			ok:   true,
		}, {
			name: "Inalid date with |",
			date: "09|10",
			want: "",
			ok:   false,
		}, {
			name: "Invalid date with //",
			date: "//",
			want: "",
			ok:   false,
		}, {
			name: "Invalid date",
			date: "09/10/2020",
			want: "",
			ok:   false,
		}, {
			name: "Invalid day 0",
			date: "00-10",
			want: "",
			ok:   false,
		}, {
			name: "Invalid day 32",
			date: "32-10",
			want: "",
			ok:   false,
		}, {
			name: "Invalid month 0",
			date: "09-00",
			want: "",
			ok:   false,
		}, {
			name: "Invalid month 13",
			date: "09-13",
			want: "",
			ok:   false,
		}, {
			name: "Invalid day and month",
			date: "32-13",
			want: "",
			ok:   false,
		}, {
			name: "Invalid combination between day and month on May",
			date: "31-04",
			want: "",
			ok:   false,
		}, {
			name: "Invalid combination between day and month on June",
			date: "31-06",
			want: "",
			ok:   false,
		}, {
			name: "Invalid combination between day and month on September",
			date: "31-09",
			want: "",
			ok:   false,
		}, {
			name: "Invalid combination between day and month on November",
			date: "31-11",
			want: "",
			ok:   false,
		}, {
			name: "Invalid combination between day and month on February",
			date: "29-02",
			want: "",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseDate(tt.date)
			if got != tt.want {
				t.Errorf("parseDate() got = %v, want %v", got, tt.want)
			}
			if ok != tt.ok {
				t.Errorf("parseDate() ok = %v, want %v", ok, tt.ok)
			}
		})
	}
}
