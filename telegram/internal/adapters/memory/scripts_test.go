package memory

import (
	"context"
	"reflect"
	"testing"

	"github.com/escalopa/gopray/telegram/internal/domain"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"
)

func TestScripts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		lang   string
		script *domain.Script
	}{
		{
			name:   "en",
			lang:   "en",
			script: randomScript(),
		},
		{
			name:   "es",
			lang:   "es",
			script: randomScript(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				ctx = context.Background()
				src = NewScriptRepository()
			)

			// Get script
			script, err := src.GetScript(ctx, tt.lang)
			require.Error(t, err)
			require.Nil(t, script)

			// Store script
			err = src.StoreScript(ctx, tt.lang, tt.script)
			require.NoError(t, err)

			// Get script
			script, err = src.GetScript(ctx, tt.lang)
			require.NoError(t, err)
			require.True(t, reflect.DeepEqual(tt.script, script))
		})
	}
}

func randomScript() *domain.Script {
	return &domain.Script{
		DatePickerStart: gofakeit.InputName(),

		January:   gofakeit.InputName(),
		February:  gofakeit.InputName(),
		March:     gofakeit.InputName(),
		April:     gofakeit.InputName(),
		May:       gofakeit.InputName(),
		June:      gofakeit.InputName(),
		July:      gofakeit.InputName(),
		August:    gofakeit.InputName(),
		September: gofakeit.InputName(),
		October:   gofakeit.InputName(),
		November:  gofakeit.InputName(),
		December:  gofakeit.InputName(),

		LanguageSelectionStart:   gofakeit.InputName(),
		LanguageSelectionSuccess: gofakeit.InputName(),
		LanguageSelectionFail:    gofakeit.InputName(),

		Fajr:    gofakeit.InputName(),
		Dohaa:   gofakeit.InputName(),
		Dhuhr:   gofakeit.InputName(),
		Asr:     gofakeit.InputName(),
		Maghrib: gofakeit.InputName(),
		Isha:    gofakeit.InputName(),

		PrayrifyTableDay:    gofakeit.InputName(),
		PrayrifyTablePrayer: gofakeit.InputName(),
		PrayrifyTableTime:   gofakeit.InputName(),
		PrayerFail:          gofakeit.InputName(),

		SubscriptionSuccess: gofakeit.InputName(),
		SubscriptionError:   gofakeit.InputName(),

		UnsubscriptionSuccess: gofakeit.InputName(),
		UnsubscriptionError:   gofakeit.InputName(),

		PrayerSoon:    gofakeit.InputName(),
		PrayerArrived: gofakeit.InputName(),
		GomaaDay:      gofakeit.InputName(),

		Help: gofakeit.InputName(),

		FeedbackStart:   gofakeit.InputName(),
		FeedbackSuccess: gofakeit.InputName(),
		FeedbackFail:    gofakeit.InputName(),

		BugReportStart:   gofakeit.InputName(),
		BugReportSuccess: gofakeit.InputName(),
		BugReportFail:    gofakeit.InputName(),
	}
}
