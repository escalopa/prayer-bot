package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Google Front End intercepts /healthz on run.app domains, so the externally
// reachable route is /health; /healthz stays for container-internal probes.
func TestHealthMuxServesBothRoutes(t *testing.T) {
	mux := http.NewServeMux()
	HealthMux(mux)
	for _, path := range []string{"/health", "/healthz"} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNoContent {
			t.Errorf("GET %s = %d, want %d", path, response.Code, http.StatusNoContent)
		}
	}
}
