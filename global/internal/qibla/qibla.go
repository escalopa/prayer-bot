package qibla

import (
	"fmt"
	"math"
)

const (
	kaabaLatitude  = 21.4225
	kaabaLongitude = 39.8262
	earthRadiusKM  = 6371.0088
)

type Result struct {
	BearingDegrees     float64
	DistanceKilometres float64
}

// Calculate returns the initial great-circle bearing from the supplied
// coordinates to the Kaaba and the corresponding surface distance.
func Calculate(latitude, longitude float64) (Result, error) {
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return Result{}, fmt.Errorf("invalid coordinates")
	}

	latitudeRadians := degreesToRadians(latitude)
	kaabaLatitudeRadians := degreesToRadians(kaabaLatitude)
	longitudeDelta := degreesToRadians(kaabaLongitude - longitude)

	y := math.Sin(longitudeDelta) * math.Cos(kaabaLatitudeRadians)
	x := math.Cos(latitudeRadians)*math.Sin(kaabaLatitudeRadians) -
		math.Sin(latitudeRadians)*math.Cos(kaabaLatitudeRadians)*math.Cos(longitudeDelta)
	bearing := math.Mod(radiansToDegrees(math.Atan2(y, x))+360, 360)

	latitudeDelta := kaabaLatitudeRadians - latitudeRadians
	haversine := math.Sin(latitudeDelta/2)*math.Sin(latitudeDelta/2) +
		math.Cos(latitudeRadians)*math.Cos(kaabaLatitudeRadians)*
			math.Sin(longitudeDelta/2)*math.Sin(longitudeDelta/2)
	haversine = math.Max(0, math.Min(1, haversine))
	distance := 2 * earthRadiusKM * math.Asin(math.Sqrt(haversine))

	return Result{BearingDegrees: bearing, DistanceKilometres: distance}, nil
}

func degreesToRadians(value float64) float64 { return value * math.Pi / 180 }
func radiansToDegrees(value float64) float64 { return value * 180 / math.Pi }
