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
	if got := RecommendedMethod(resolved.CountryCode); got != domain.MethodEgyptian {
		t.Fatalf("unexpected method: %s", got)
	}
}
