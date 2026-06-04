package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDuration_JSON(t *testing.T) {
	tests := []struct {
		name      string
		value     Duration
		expect    string
		expectErr bool
		payload   string
	}{
		{
			name:   "marshal roundtrip",
			value:  Duration(90 * time.Minute),
			expect: "\"1h30m0s\"",
		},
		{
			name:      "invalid unmarshal",
			expectErr: true,
			payload:   `"not valid"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectErr {
				var got Duration
				if err := json.Unmarshal([]byte(tt.payload), &got); err == nil {
					t.Fatalf("expected error")
				}
				return
			}

			b, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("marshal duration: %v", err)
			}

			if got := string(b); got != tt.expect {
				t.Fatalf("unexpected marshal %s", got)
			}

			var decoded Duration
			if err := json.Unmarshal(b, &decoded); err != nil {
				t.Fatalf("unmarshal duration: %v", err)
			}

			if time.Duration(decoded) != time.Duration(tt.value) {
				t.Fatalf("decoded duration mismatch: want %v got %v", tt.value, decoded.Duration())
			}
		})
	}
}
