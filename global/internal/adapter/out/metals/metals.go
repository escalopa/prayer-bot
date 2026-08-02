// Package metals fetches the daily gold/silver spot price and USD currency
// rate table used to localize the Zakat niSab. Both upstreams are free and
// require no API key, so no new secret is introduced. The maintenance job calls
// Fetch once a day and caches the result; user requests never reach these APIs.
package metals

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/port"
)

const (
	// defaultGoldURL returns gold (XAU) and silver (XAG) spot prices in USD per
	// troy ounce, e.g. GET {defaultGoldURL}/XAU. No key required.
	defaultGoldURL = "https://api.gold-api.com/price"
	// defaultFXURL returns every supported currency rate with base USD in one
	// response. gold-api.com only covers a handful of major currencies, so this
	// separate feed provides the currencies the bot's audience actually uses.
	defaultFXURL = "https://open.er-api.com/v6/latest/USD"
)

// Client fetches metal prices and currency rates. Endpoints are overridable so
// tests can point at an httptest server.
type Client struct {
	http    *http.Client
	goldURL string
	fxURL   string
}

// Option customizes a Client.
type Option func(*Client)

// WithEndpoints overrides the upstream URLs (used in tests).
func WithEndpoints(goldURL, fxURL string) Option {
	return func(c *Client) {
		c.goldURL = goldURL
		c.fxURL = fxURL
	}
}

// NewClient returns a metals client with the given per-request timeout.
func NewClient(timeout time.Duration, opts ...Option) *Client {
	c := &Client{
		http:    &http.Client{Timeout: timeout},
		goldURL: defaultGoldURL,
		fxURL:   defaultFXURL,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Fetch retrieves gold and silver spot prices plus the USD currency rate table.
// It returns an error if any upstream fails or returns unusable data; callers
// should keep the previously cached value on error rather than overwriting it.
func (c *Client) Fetch(ctx context.Context) (domain.MetalPrices, error) {
	gold, err := c.metalPrice(ctx, "XAU")
	if err != nil {
		return domain.MetalPrices{}, fmt.Errorf("fetch gold price: %w", err)
	}
	silver, err := c.metalPrice(ctx, "XAG")
	if err != nil {
		return domain.MetalPrices{}, fmt.Errorf("fetch silver price: %w", err)
	}
	rates, err := c.currencyRates(ctx)
	if err != nil {
		return domain.MetalPrices{}, fmt.Errorf("fetch currency rates: %w", err)
	}
	prices := domain.MetalPrices{
		GoldUSDPerOunce:   gold,
		SilverUSDPerOunce: silver,
		Rates:             rates,
	}
	if !prices.Valid() {
		return domain.MetalPrices{}, fmt.Errorf("fetched metal prices are invalid")
	}
	return prices, nil
}

func (c *Client) metalPrice(ctx context.Context, symbol string) (float64, error) {
	var payload struct {
		Price float64 `json:"price"`
	}
	if err := c.getJSON(ctx, c.goldURL+"/"+symbol, &payload); err != nil {
		return 0, err
	}
	if payload.Price <= 0 {
		return 0, fmt.Errorf("non-positive price for %s", symbol)
	}
	return payload.Price, nil
}

func (c *Client) currencyRates(ctx context.Context) (map[string]float64, error) {
	var payload struct {
		Result string             `json:"result"`
		Rates  map[string]float64 `json:"rates"`
	}
	if err := c.getJSON(ctx, c.fxURL, &payload); err != nil {
		return nil, err
	}
	if payload.Result != "success" {
		return nil, fmt.Errorf("currency feed returned result %q", payload.Result)
	}
	if payload.Rates["USD"] != 1 {
		return nil, fmt.Errorf("currency feed is not USD-based")
	}
	return payload.Rates, nil
}

func (c *Client) getJSON(ctx context.Context, url string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close() //nolint:errcheck
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

var _ port.MetalSource = (*Client)(nil)
