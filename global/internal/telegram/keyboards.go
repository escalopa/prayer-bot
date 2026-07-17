package telegram

import (
	"fmt"

	"github.com/go-telegram/bot/models"

	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
)

func mainKeyboard(locale i18n.Locale) *models.ReplyKeyboardMarkup {
	return &models.ReplyKeyboardMarkup{
		Keyboard: [][]models.KeyboardButton{
			{{Text: locale.Button(i18n.ActionToday)}, {Text: locale.Button(i18n.ActionTomorrow)}},
			{{Text: locale.Button(i18n.ActionNext)}, {Text: locale.Button(i18n.ActionLocation)}},
			{{Text: locale.Button(i18n.ActionSettings)}, {Text: locale.Button(i18n.ActionReminders)}},
			{{Text: locale.Button(i18n.ActionLanguage)}, {Text: locale.Button(i18n.ActionHelp)}},
		},
		IsPersistent:   true,
		ResizeKeyboard: true,
	}
}

func settingsKeyboard(locale i18n.Locale) *models.InlineKeyboardMarkup {
	return inlineKeyboard(
		[]models.InlineKeyboardButton{callbackButton(locale.Button("method"), "settings:method")},
		[]models.InlineKeyboardButton{callbackButton(locale.Button("madhab"), "settings:madhab")},
		[]models.InlineKeyboardButton{callbackButton(locale.Button("highlat"), "settings:highlat")},
		[]models.InlineKeyboardButton{callbackButton(locale.Button("adjustments"), "settings:adjustments")},
		[]models.InlineKeyboardButton{callbackButton(locale.Button("hijri"), "settings:hijri")},
		[]models.InlineKeyboardButton{callbackButton(locale.Button("close"), "close")},
	)
}

func methodKeyboard(current domain.Method, locale i18n.Locale) *models.InlineKeyboardMarkup {
	rows := make([][]models.InlineKeyboardButton, 0, len(domain.SupportedMethods())+1)
	for _, method := range domain.SupportedMethods() {
		rows = append(rows, []models.InlineKeyboardButton{callbackButton(
			selectedLabel(locale.Method(method), method == current), "method:"+string(method),
		)})
	}
	rows = append(rows, []models.InlineKeyboardButton{callbackButton(locale.Button("back"), "settings")})
	return inlineKeyboard(rows...)
}

func madhabKeyboard(current domain.Madhab, locale i18n.Locale) *models.InlineKeyboardMarkup {
	return inlineKeyboard(
		[]models.InlineKeyboardButton{callbackButton(selectedLabel(locale.Madhab(domain.MadhabShafii), current == domain.MadhabShafii), "madhab:shafii")},
		[]models.InlineKeyboardButton{callbackButton(selectedLabel(locale.Madhab(domain.MadhabHanafi), current == domain.MadhabHanafi), "madhab:hanafi")},
		[]models.InlineKeyboardButton{callbackButton(locale.Button("back"), "settings")},
	)
}

func highLatitudeKeyboard(current domain.HighLatitudeRule, locale i18n.Locale) *models.InlineKeyboardMarkup {
	rules := []domain.HighLatitudeRule{
		domain.HighLatitudeAngleBased, domain.HighLatitudeMiddleNight, domain.HighLatitudeSeventhNight,
	}
	rows := make([][]models.InlineKeyboardButton, 0, len(rules)+1)
	for _, rule := range rules {
		rows = append(rows, []models.InlineKeyboardButton{callbackButton(
			selectedLabel(locale.HighLatitudeRule(rule), rule == current), "highlat:"+string(rule),
		)})
	}
	rows = append(rows, []models.InlineKeyboardButton{callbackButton(locale.Button("back"), "settings")})
	return inlineKeyboard(rows...)
}

func adjustmentKeyboard(profile domain.PrayerProfile, locale i18n.Locale) *models.InlineKeyboardMarkup {
	rows := make([][]models.InlineKeyboardButton, 0, 4)
	prayers := allPrayers()
	for index := 0; index < len(prayers); index += 2 {
		row := make([]models.InlineKeyboardButton, 0, 2)
		for offset := 0; offset < 2 && index+offset < len(prayers); offset++ {
			prayer := prayers[index+offset]
			row = append(row, callbackButton(
				fmt.Sprintf("%s %+d", locale.Prayer(prayer), adjustmentValue(profile.Adjustments, prayer)),
				"adjust:"+string(prayer),
			))
		}
		rows = append(rows, row)
	}
	rows = append(rows, []models.InlineKeyboardButton{callbackButton(locale.Button("back"), "settings")})
	return inlineKeyboard(rows...)
}

func adjustmentDetailKeyboard(prayer domain.Prayer, locale i18n.Locale) *models.InlineKeyboardMarkup {
	return inlineKeyboard(
		[]models.InlineKeyboardButton{
			callbackButton("−5", "adjust_delta:"+string(prayer)+":-5"),
			callbackButton("−1", "adjust_delta:"+string(prayer)+":-1"),
			callbackButton("+1", "adjust_delta:"+string(prayer)+":1"),
			callbackButton("+5", "adjust_delta:"+string(prayer)+":5"),
		},
		[]models.InlineKeyboardButton{
			callbackButton("↺ 0", "adjust_set:"+string(prayer)+":0"),
			callbackButton(locale.Button("back"), "settings:adjustments"),
		},
	)
}

func hijriKeyboard(current int, locale i18n.Locale) *models.InlineKeyboardMarkup {
	row := make([]models.InlineKeyboardButton, 0, 5)
	for value := -2; value <= 2; value++ {
		label := fmt.Sprintf("%+d", value)
		row = append(row, callbackButton(selectedLabel(label, value == current), fmt.Sprintf("hijri:%d", value)))
	}
	return inlineKeyboard(
		row,
		[]models.InlineKeyboardButton{callbackButton(locale.Button("back"), "settings")},
	)
}

func remindersKeyboard(state reminderState, locale i18n.Locale) *models.InlineKeyboardMarkup {
	toggle := func(label, kind string, enabled bool) models.InlineKeyboardButton {
		action, prefix := "on", "○ "
		if enabled {
			action, prefix = "off", "✓ "
		}
		return callbackButton(prefix+label, "reminders:"+kind+":"+action)
	}
	return inlineKeyboard(
		[]models.InlineKeyboardButton{toggle(locale.Button("prayer_reminders"), "prayer", state.Prayer)},
		[]models.InlineKeyboardButton{toggle(locale.Button("fasting_reminders"), "fasting", state.Fasting)},
		[]models.InlineKeyboardButton{toggle(locale.Button("kahf_reminders"), "kahf", state.Kahf)},
		[]models.InlineKeyboardButton{callbackButton(locale.Button("close"), "close")},
	)
}

func languageKeyboard(current string) *models.InlineKeyboardMarkup {
	locales := i18n.Supported()
	rows := make([][]models.InlineKeyboardButton, 0, (len(locales)+1)/2)
	for index := 0; index < len(locales); index += 2 {
		row := make([]models.InlineKeyboardButton, 0, 2)
		for offset := 0; offset < 2 && index+offset < len(locales); offset++ {
			locale := locales[index+offset]
			row = append(row, callbackButton(selectedLabel(locale.NativeName, locale.Code == current), "language:"+locale.Code))
		}
		rows = append(rows, row)
	}
	return inlineKeyboard(rows...)
}

func inlineKeyboard(rows ...[]models.InlineKeyboardButton) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func callbackButton(text, data string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{Text: text, CallbackData: data}
}

func selectedLabel(label string, selected bool) string {
	if selected {
		return "✓ " + label
	}
	return label
}
