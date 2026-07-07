package domain

import (
	"testing"
	"time"
)

func TestDurationJSONRoundTrip(t *testing.T) {
	t.Parallel()

	orig := Duration(90 * time.Minute)

	data, err := orig.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	if want := `"1h30m0s"`; string(data) != want {
		t.Fatalf("MarshalJSON() = %s, want %s", data, want)
	}

	var got Duration
	if err := got.UnmarshalJSON(data); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if got != orig {
		t.Fatalf("round trip = %v, want %v", got.Duration(), orig.Duration())
	}
}

func TestDurationUnmarshalInvalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
	}{
		{"invalid duration string", `"not-a-duration"`},
		{"non-string payload", `123`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var d Duration
			if err := d.UnmarshalJSON([]byte(tt.in)); err == nil {
				t.Fatalf("UnmarshalJSON(%s) expected error, got nil", tt.in)
			}
		})
	}
}
