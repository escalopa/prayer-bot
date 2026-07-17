package botprofile

import (
	"testing"
	"unicode/utf8"

	"github.com/escalopa/prayer-bot/global/internal/i18n"
)

func TestLocalizedCommandsAreCompleteAndWithinTelegramLimits(t *testing.T) {
	for _, locale := range i18n.Supported() {
		items := commands(locale)
		if len(items) != 10 {
			t.Fatalf("%s has %d commands, want 10", locale.Code, len(items))
		}
		seen := make(map[string]bool)
		for _, item := range items {
			if seen[item.Command] {
				t.Errorf("%s duplicates /%s", locale.Code, item.Command)
			}
			seen[item.Command] = true
			length := utf8.RuneCountInString(item.Description)
			if length < 1 || length > 256 {
				t.Errorf("%s /%s description has invalid length %d", locale.Code, item.Command, length)
			}
		}
	}
}
