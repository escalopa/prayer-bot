package location

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/domain"
)

func TestResolve(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/timezone" {
			_, _ = w.Write([]byte(`{"status":"OK","timeZoneId":"Africa/Cairo"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"OK","results":[{"place_id":"place-1","address_components":[{"long_name":"Cairo","short_name":"Cairo","types":["locality"]},{"long_name":"Egypt","short_name":"EG","types":["country"]}]}]}`))
	}))
	defer server.Close()

	client := NewGoogleMaps("key", time.Second)
	client.timezoneURL = server.URL + "/timezone"
	client.geocodingURL = server.URL + "/geocode"
	resolved, err := client.Resolve(context.Background(), 30.044, 31.236)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Timezone != "Africa/Cairo" || resolved.City != "Cairo" || resolved.PlaceID != "place-1" {
		t.Fatalf("unexpected result: %+v", resolved)
	}
	if got := domain.RecommendedMethod(resolved.CountryCode); got != domain.MethodEgyptian {
		t.Fatalf("unexpected method: %s", got)
	}
}

func TestSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("address"); got != "Istanbul" {
			t.Errorf("address = %q, want Istanbul", got)
		}
		if got := r.URL.Query().Get("language"); got != "tr" {
			t.Errorf("language = %q, want tr", got)
		}
		_, _ = w.Write([]byte(`{"status":"OK","results":[
			{"formatted_address":"İstanbul, Türkiye","geometry":{"location":{"lat":41.008,"lng":28.978}}},
			{"formatted_address":"Istanbul, OH, USA","geometry":{"location":{"lat":41.393,"lng":-82.933}}}
		]}`))
	}))
	defer server.Close()

	client := NewGoogleMaps("key", time.Second)
	client.geocodingURL = server.URL + "/geocode"
	candidates, err := client.Search(context.Background(), "Istanbul", "tr")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 {
		t.Fatalf("got %d candidates, want 2", len(candidates))
	}
	first := candidates[0]
	if first.Label != "İstanbul, Türkiye" || first.Latitude != 41.008 || first.Longitude != 28.978 {
		t.Fatalf("unexpected first candidate: %+v", first)
	}
}

func TestSearchZeroResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ZERO_RESULTS","results":[]}`))
	}))
	defer server.Close()

	client := NewGoogleMaps("key", time.Second)
	client.geocodingURL = server.URL + "/geocode"
	candidates, err := client.Search(context.Background(), "xxxxxx", "en")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected no candidates, got %+v", candidates)
	}
}
