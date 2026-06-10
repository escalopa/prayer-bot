package function

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebhookProxyForwardsPostRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("X-Telegram-Bot-Api-Secret-Token"); got != "secret-token" {
			t.Fatalf("secret token = %q, want %q", got, "secret-token")
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		if got := string(body); got != `{"ok":true}` {
			t.Fatalf("body = %q, want %q", got, `{"ok":true}`)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	t.Cleanup(upstream.Close)

	t.Setenv("YC_DISPATCHER_URL", upstream.URL)

	req := httptest.NewRequest(http.MethodPost, "https://example.com", strings.NewReader(`{"ok":true}`))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret-token")

	rr := httptest.NewRecorder()
	WebhookProxy(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}

	if got := rr.Body.String(); got != `{"status":"ok"}` {
		t.Fatalf("response body = %q, want %q", got, `{"status":"ok"}`)
	}
}

func TestWebhookProxyRejectsNonPost(t *testing.T) {
	t.Setenv("YC_DISPATCHER_URL", "https://example.com")

	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	rr := httptest.NewRecorder()

	WebhookProxy(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
	if got := rr.Header().Get("Allow"); got != http.MethodPost {
		t.Fatalf("allow header = %q, want POST", got)
	}
}
