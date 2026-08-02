package qibla

import (
	"math"
	"testing"
)

func TestCalculateKnownBearings(t *testing.T) {
	tests := []struct {
		name                string
		latitude, longitude float64
		wantBearing         float64
		wantDistance        float64
	}{
		{name: "Cairo", latitude: 30.0444, longitude: 31.2357, wantBearing: 136.1, wantDistance: 1287},
		{name: "London", latitude: 51.5074, longitude: -0.1278, wantBearing: 118.9, wantDistance: 4794},
		{name: "New York", latitude: 40.7128, longitude: -74.0060, wantBearing: 58.5, wantDistance: 10307},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := Calculate(test.latitude, test.longitude)
			if err != nil {
				t.Fatal(err)
			}
			if math.Abs(result.BearingDegrees-test.wantBearing) > 0.6 {
				t.Fatalf("bearing = %.2f, want approximately %.2f", result.BearingDegrees, test.wantBearing)
			}
			if math.Abs(result.DistanceKilometres-test.wantDistance) > 15 {
				t.Fatalf("distance = %.2f, want approximately %.2f", result.DistanceKilometres, test.wantDistance)
			}
		})
	}
}

func TestCalculateRejectsInvalidCoordinates(t *testing.T) {
	if _, err := Calculate(91, 0); err == nil {
		t.Fatal("expected invalid latitude to fail")
	}
	if _, err := Calculate(0, -181); err == nil {
		t.Fatal("expected invalid longitude to fail")
	}
}
