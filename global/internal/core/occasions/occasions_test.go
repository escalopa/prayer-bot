package occasions

import (
	"net/url"
	"testing"
	"time"
)

func TestCatalogHasUniqueStableIDsAndValidSources(t *testing.T) {
	seen := map[string]bool{}
	for _, definition := range Catalog() {
		if definition.ID == "" || seen[definition.ID] {
			t.Fatalf("invalid or duplicate ID %q", definition.ID)
		}
		seen[definition.ID] = true
		if definition.Month < 1 || definition.Month > 12 ||
			definition.Day < 1 || definition.Day > 30 {
			t.Fatalf("invalid Hijri date for %q", definition.ID)
		}
		for _, source := range definition.Sources {
			if source.Label == "" || source.URL == "" {
				t.Fatalf("incomplete source for %q", definition.ID)
			}
			parsed, err := url.Parse(source.URL)
			if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
				t.Fatalf("invalid HTTPS source for %q: %q", definition.ID, source.URL)
			}
		}
	}
}

func TestBetweenRespectsHijriAdjustment(t *testing.T) {
	start := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	withoutAdjustment, err := Between(start, 400, 0)
	if err != nil {
		t.Fatal(err)
	}
	withAdjustment, err := Between(start, 400, 1)
	if err != nil {
		t.Fatal(err)
	}
	find := func(values []Occurrence, id string) time.Time {
		for _, value := range values {
			if value.Definition.ID == id {
				return value.Date
			}
		}
		return time.Time{}
	}
	base := find(withoutAdjustment, "arafah")
	adjusted := find(withAdjustment, "arafah")
	if base.IsZero() || adjusted.IsZero() || !adjusted.Equal(base.AddDate(0, 0, -1)) {
		t.Fatalf("adjustment did not move Arafah: base=%v adjusted=%v", base, adjusted)
	}
}

func TestNextFiltersCategory(t *testing.T) {
	start := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	occurrence, err := Next(start, 0, CategoryObserved)
	if err != nil {
		t.Fatal(err)
	}
	if occurrence.Definition.Category != CategoryObserved || occurrence.Date.Before(start) {
		t.Fatalf("unexpected occurrence: %+v", occurrence)
	}
}
