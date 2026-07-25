package metals

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, goldHandler, fxHandler http.HandlerFunc) *Client {
	t.Helper()
	gold := httptest.NewServer(goldHandler)
	fx := httptest.NewServer(fxHandler)
	t.Cleanup(gold.Close)
	t.Cleanup(fx.Close)
	return NewClient(5*time.Second, WithEndpoints(gold.URL+"/price", fx.URL))
}

func TestFetchSuccess(t *testing.T) {
	client := newTestClient(t,
		func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.HasSuffix(r.URL.Path, "/XAU"):
				_, _ = w.Write([]byte(`{"price":4053.7,"symbol":"XAU"}`))
			case strings.HasSuffix(r.URL.Path, "/XAG"):
				_, _ = w.Write([]byte(`{"price":58.28,"symbol":"XAG"}`))
			default:
				http.NotFound(w, r)
			}
		},
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"result":"success","rates":{"USD":1,"EGP":51.3,"TRY":47.3}}`))
		},
	)
	prices, err := client.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prices.GoldUSDPerOunce != 4053.7 || prices.SilverUSDPerOunce != 58.28 {
		t.Fatalf("unexpected prices: %+v", prices)
	}
	if prices.Rates["EGP"] != 51.3 || prices.Rates["USD"] != 1 {
		t.Fatalf("unexpected rates: %+v", prices.Rates)
	}
	if !prices.Valid() {
		t.Fatal("prices should be valid")
	}
}

func TestFetchRejectsNonUSDBase(t *testing.T) {
	client := newTestClient(t,
		func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"price":1}`)) },
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"result":"success","rates":{"USD":0.9}}`))
		},
	)
	if _, err := client.Fetch(context.Background()); err == nil {
		t.Fatal("expected error for non-USD base rate table")
	}
}

func TestFetchRejectsNonPositivePrice(t *testing.T) {
	client := newTestClient(t,
		func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"price":0}`)) },
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"result":"success","rates":{"USD":1}}`))
		},
	)
	if _, err := client.Fetch(context.Background()); err == nil {
		t.Fatal("expected error for non-positive metal price")
	}
}

func TestFetchPropagatesUpstreamFailure(t *testing.T) {
	client := newTestClient(t,
		func(w http.ResponseWriter, r *http.Request) { http.Error(w, "boom", http.StatusBadGateway) },
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"result":"success","rates":{"USD":1}}`))
		},
	)
	if _, err := client.Fetch(context.Background()); err == nil {
		t.Fatal("expected error when the gold upstream fails")
	}
}
