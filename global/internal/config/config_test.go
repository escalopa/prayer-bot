package config

import "testing"

func TestValidWebhookSecret(t *testing.T) {
	for _, value := range []string{"abc-123_DEF", "x"} {
		if !validWebhookSecret(value) {
			t.Fatalf("expected %q to be valid", value)
		}
	}
	for _, value := range []string{"", "contains space", "contains.dot"} {
		if validWebhookSecret(value) {
			t.Fatalf("expected %q to be invalid", value)
		}
	}
}
