package location

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/domain"
)

type Resolved struct {
	Timezone    string
	PlaceID     string
	City        string
	CountryCode string
}

type Resolver interface {
	Resolve(context.Context, float64, float64) (Resolved, error)
}

type GoogleMaps struct {
	apiKey       string
	client       *http.Client
	timezoneURL  string
	geocodingURL string
}

func NewGoogleMaps(apiKey string, timeout time.Duration) *GoogleMaps {
	return &GoogleMaps{
		apiKey:       apiKey,
		client:       &http.Client{Timeout: timeout},
		timezoneURL:  "https://maps.googleapis.com/maps/api/timezone/json",
		geocodingURL: "https://maps.googleapis.com/maps/api/geocode/json",
	}
}

func (g *GoogleMaps) Resolve(ctx context.Context, latitude, longitude float64) (Resolved, error) {
	timezone, err := g.resolveTimezone(ctx, latitude, longitude)
	if err != nil {
		return Resolved{}, err
	}
	placeID, city, country, err := g.reverseGeocode(ctx, latitude, longitude)
	if err != nil {
		return Resolved{}, err
	}
	return Resolved{Timezone: timezone, PlaceID: placeID, City: city, CountryCode: country}, nil
}

func (g *GoogleMaps) resolveTimezone(ctx context.Context, latitude, longitude float64) (string, error) {
	values := url.Values{
		"location":  {coordinates(latitude, longitude)},
		"timestamp": {strconv.FormatInt(time.Now().Unix(), 10)},
		"key":       {g.apiKey},
	}
	var response struct {
		Status       string `json:"status"`
		ErrorMessage string `json:"errorMessage"`
		TimeZoneID   string `json:"timeZoneId"`
	}
	if err := g.getJSON(ctx, g.timezoneURL, values, &response); err != nil {
		return "", fmt.Errorf("resolve timezone: %w", err)
	}
	if response.Status != "OK" || response.TimeZoneID == "" {
		return "", fmt.Errorf("resolve timezone: Google status %s: %s", response.Status, response.ErrorMessage)
	}
	return response.TimeZoneID, nil
}

func (g *GoogleMaps) reverseGeocode(ctx context.Context, latitude, longitude float64) (string, string, string, error) {
	values := url.Values{
		"latlng":   {coordinates(latitude, longitude)},
		"language": {"en"},
		"key":      {g.apiKey},
	}
	var response struct {
		Status       string `json:"status"`
		ErrorMessage string `json:"error_message"`
		Results      []struct {
			PlaceID    string `json:"place_id"`
			Components []struct {
				LongName  string   `json:"long_name"`
				ShortName string   `json:"short_name"`
				Types     []string `json:"types"`
			} `json:"address_components"`
		} `json:"results"`
	}
	if err := g.getJSON(ctx, g.geocodingURL, values, &response); err != nil {
		return "", "", "", fmt.Errorf("reverse geocode: %w", err)
	}
	if response.Status != "OK" || len(response.Results) == 0 {
		return "", "", "", fmt.Errorf("reverse geocode: Google status %s: %s", response.Status, response.ErrorMessage)
	}

	result := response.Results[0]
	var city, fallback, country string
	for _, component := range result.Components {
		for _, kind := range component.Types {
			switch kind {
			case "locality", "postal_town":
				if city == "" {
					city = component.LongName
				}
			case "administrative_area_level_1":
				fallback = component.LongName
			case "country":
				country = strings.ToUpper(component.ShortName)
			}
		}
	}
	if city == "" {
		city = fallback
	}
	return result.PlaceID, city, country, nil
}

func (g *GoogleMaps) getJSON(ctx context.Context, endpoint string, values url.Values, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+values.Encode(), nil)
	if err != nil {
		return fmt.Errorf("create request")
	}
	response, err := g.client.Do(request)
	if err != nil {
		// net/http errors can include the full URL, which contains the API key.
		return fmt.Errorf("request failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status %d", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func coordinates(latitude, longitude float64) string {
	return strconv.FormatFloat(latitude, 'f', 6, 64) + "," + strconv.FormatFloat(longitude, 'f', 6, 64)
}

func RecommendedMethod(countryCode string) domain.Method {
	switch strings.ToUpper(countryCode) {
	case "EG":
		return domain.MethodEgyptian
	case "SA":
		return domain.MethodUmmAlQura
	case "PK", "IN", "BD", "AF":
		return domain.MethodKarachi
	case "TR":
		return domain.MethodDiyanet
	case "ID":
		return domain.MethodKemenag
	case "SG":
		return domain.MethodMUIS
	case "MY":
		return domain.MethodJAKIM
	case "US", "CA":
		return domain.MethodISNA
	default:
		return domain.MethodMWL
	}
}
