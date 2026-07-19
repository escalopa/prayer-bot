package i18n

import (
	"testing"

	"github.com/escalopa/prayer-bot/global/internal/occasions"
)

func TestEveryOccasionIsLocalized(t *testing.T) {
	for _, locale := range Supported() {
		for _, definition := range occasions.Catalog() {
			copy := locale.Occasion(definition.ID)
			if copy.Title == "" || copy.Summary == "" || copy.Action == "" {
				t.Errorf("%s is missing occasion copy for %s", locale.Code, definition.ID)
			}
			if locale.OccasionCategory(string(definition.Category)) == "" {
				t.Errorf("%s is missing category %s", locale.Code, definition.Category)
			}
		}
		for _, key := range []string{"title", "help", "disclaimer", "recommended", "sources", "major_reminders", "fasting_reminders", "observed_reminders", "schedule"} {
			if locale.OccasionUI(key) == "" {
				t.Errorf("%s is missing occasion UI key %s", locale.Code, key)
			}
		}
	}
}
