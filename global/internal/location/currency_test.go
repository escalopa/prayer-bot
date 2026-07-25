package location

import "testing"

func TestCurrencyForCountry(t *testing.T) {
	cases := map[string]string{
		"EG": "EGP", "eg": "EGP", " TR ": "TRY", "PK": "PKR",
		"UZ": "UZS", "US": "USD", "FR": "EUR", "DE": "EUR",
		"": "", "ZZ": "", "XX": "",
	}
	for input, want := range cases {
		if got := CurrencyForCountry(input); got != want {
			t.Errorf("CurrencyForCountry(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCurrencyForTimezone(t *testing.T) {
	cases := map[string]string{
		"Africa/Cairo":     "EGP",
		"Europe/Istanbul":  "TRY",
		"Asia/Tashkent":    "UZS",
		"Europe/Moscow":    "RUB",
		"America/New_York": "USD",
		"Antarctica/Troll": "",
		"":                 "",
	}
	for input, want := range cases {
		if got := CurrencyForTimezone(input); got != want {
			t.Errorf("CurrencyForTimezone(%q) = %q, want %q", input, got, want)
		}
	}
}
