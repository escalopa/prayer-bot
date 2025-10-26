package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/go-telegram/bot/models"
)

const (
	languagesPerRow = 2
	monthsPerRow    = 3
	remindPerRow    = 4
	daysPerRow      = 5

	buttonBack = "🔙"
)

func (h *Handler) languagesKeyboard() *models.InlineKeyboardMarkup {
	languages := h.lp.GetLanguages()
	rows, empty := layoutRowsInfo(len(languages), languagesPerRow)

	kb := &models.InlineKeyboardMarkup{InlineKeyboard: make([][]models.InlineKeyboardButton, rows)}

	row := 0
	for i, lang := range languages {
		if i%languagesPerRow == 0 && i != 0 {
			row++
		}
		kb.InlineKeyboard[row] = append(kb.InlineKeyboard[row], models.InlineKeyboardButton{
			Text:         lang.Name,
			CallbackData: fmt.Sprintf("%s%s", languageQuery, lang.Code),
		})
	}

	for i := 0; i < empty; i++ {
		kb.InlineKeyboard[row] = append(kb.InlineKeyboard[row], emptyButton())
	}

	return kb
}

func (h *Handler) monthsKeyboard(languageCode string) *models.InlineKeyboardMarkup {
	months := h.lp.GetText(languageCode).GetMonths()
	rows, empty := layoutRowsInfo(len(months), monthsPerRow)

	kb := &models.InlineKeyboardMarkup{InlineKeyboard: make([][]models.InlineKeyboardButton, rows)}

	row := 0
	for i, month := range months {
		if i%monthsPerRow == 0 && i != 0 {
			row++
		}
		kb.InlineKeyboard[row] = append(kb.InlineKeyboard[row], models.InlineKeyboardButton{
			Text:         month.Name,
			CallbackData: fmt.Sprintf("%s%d", monthQuery, month.ID),
		})
	}

	for i := 0; i < empty; i++ {
		kb.InlineKeyboard[row] = append(kb.InlineKeyboard[row], emptyButton())
	}

	return kb
}

func (h *Handler) daysKeyboard(now time.Time, month int) *models.InlineKeyboardMarkup {
	days := daysInMonth(time.Month(month), now.Year())
	rows, empty := layoutRowsInfo(days, daysPerRow)

	kb := &models.InlineKeyboardMarkup{InlineKeyboard: make([][]models.InlineKeyboardButton, rows)}

	row := 0
	for i := 0; i < days; i++ {
		if i%daysPerRow == 0 && i != 0 {
			row++
		}

		day := i + 1
		kb.InlineKeyboard[row] = append(kb.InlineKeyboard[row], models.InlineKeyboardButton{
			Text:         strconv.Itoa(day),
			CallbackData: fmt.Sprintf("%s%d%s%d", dayQuery, month, dataSplitterQuery, day),
		})
	}

	for i := 0; i < empty; i++ {
		kb.InlineKeyboard[row] = append(kb.InlineKeyboard[row], emptyButton())
	}

	return kb
}

func (h *Handler) remindMenuKeyboard(chat *domain.Chat) *models.InlineKeyboardMarkup {
	text := h.lp.GetText(chat.LanguageCode)

	// Calculate number of rows based on whether it's a group
	numRows := 4
	if isChatGroup(chat.ChatID) {
		numRows = 5
	}
	kb := &models.InlineKeyboardMarkup{InlineKeyboard: make([][]models.InlineKeyboardButton, numRows)}

	rowIndex := 0

	if chat.Subscribed {
		kb.InlineKeyboard[rowIndex] = []models.InlineKeyboardButton{
			{Text: text.RemindMenu.Disable, CallbackData: "remind:toggle|"},
		}
	} else {
		kb.InlineKeyboard[rowIndex] = []models.InlineKeyboardButton{
			{Text: text.RemindMenu.Enable, CallbackData: "remind:toggle|"},
		}
	}
	rowIndex++

	tomorrowOffset := domain.FormatDuration(chat.Reminder.Tomorrow.Offset)
	kb.InlineKeyboard[rowIndex] = []models.InlineKeyboardButton{
		{Text: fmt.Sprintf("%s (%s)", text.RemindMenu.Tomorrow, tomorrowOffset), CallbackData: "remind:edit:tomorrow|"},
	}
	rowIndex++

	soonOffset := domain.FormatDuration(chat.Reminder.Soon.Offset)
	kb.InlineKeyboard[rowIndex] = []models.InlineKeyboardButton{
		{Text: fmt.Sprintf("%s (%s)", text.RemindMenu.Soon, soonOffset), CallbackData: "remind:edit:soon|"},
	}
	rowIndex++

	// Only show Jamaat Settings for group chats
	if isChatGroup(chat.ChatID) {
		kb.InlineKeyboard[rowIndex] = []models.InlineKeyboardButton{
			{Text: text.RemindMenu.JamaatSettings, CallbackData: "remind:jamaat:menu|"},
		}
		rowIndex++
	}

	kb.InlineKeyboard[rowIndex] = []models.InlineKeyboardButton{
		{Text: text.RemindMenu.Close, CallbackData: "remind:close|"},
	}

	return kb
}

func (h *Handler) remindEditKeyboard(reminderType domain.ReminderType, languageCode string) *models.InlineKeyboardMarkup {
	kb := &models.InlineKeyboardMarkup{InlineKeyboard: make([][]models.InlineKeyboardButton, 3)}

	kb.InlineKeyboard[0] = []models.InlineKeyboardButton{
		{Text: "+1m", CallbackData: fmt.Sprintf("remind:adjust:%s:+1m|", reminderType)},
		{Text: "+10m", CallbackData: fmt.Sprintf("remind:adjust:%s:+10m|", reminderType)},
		{Text: "+1h", CallbackData: fmt.Sprintf("remind:adjust:%s:+1h|", reminderType)},
	}

	kb.InlineKeyboard[1] = []models.InlineKeyboardButton{
		{Text: "-1m", CallbackData: fmt.Sprintf("remind:adjust:%s:-1m|", reminderType)},
		{Text: "-10m", CallbackData: fmt.Sprintf("remind:adjust:%s:-10m|", reminderType)},
		{Text: "-1h", CallbackData: fmt.Sprintf("remind:adjust:%s:-1h|", reminderType)},
	}

	kb.InlineKeyboard[2] = []models.InlineKeyboardButton{
		{Text: buttonBack, CallbackData: "remind:back:menu|"},
	}

	return kb
}

func (h *Handler) jammatMenuKeyboard(chat *domain.Chat) *models.InlineKeyboardMarkup {
	text := h.lp.GetText(chat.LanguageCode)

	prayerIDs := []domain.PrayerID{
		domain.PrayerIDFajr,
		domain.PrayerIDDhuhr,
		domain.PrayerIDAsr,
		domain.PrayerIDMaghrib,
		domain.PrayerIDIsha,
	}

	kb := &models.InlineKeyboardMarkup{InlineKeyboard: make([][]models.InlineKeyboardButton, len(prayerIDs)+2)}

	if chat.Reminder.Jamaat.Enabled {
		kb.InlineKeyboard[0] = []models.InlineKeyboardButton{
			{Text: text.JamaatMenu.Disable, CallbackData: "remind:jamaat:toggle|"},
		}
	} else {
		kb.InlineKeyboard[0] = []models.InlineKeyboardButton{
			{Text: text.JamaatMenu.Enable, CallbackData: "remind:jamaat:toggle|"},
		}
	}

	for i, prayerID := range prayerIDs {
		delay := chat.Reminder.Jamaat.Delay.GetDelayByPrayerID(prayerID)
		kb.InlineKeyboard[i+1] = []models.InlineKeyboardButton{
			{
				Text:         fmt.Sprintf("%s (%s)", text.Prayer[int(prayerID)], domain.FormatDuration(delay)),
				CallbackData: fmt.Sprintf("remind:jamaat:edit:%s|", prayerID.String()),
			},
		}
	}

	kb.InlineKeyboard[len(prayerIDs)+1] = []models.InlineKeyboardButton{
		{Text: buttonBack, CallbackData: "remind:back:menu|"},
	}

	return kb
}

func (h *Handler) jammatEditKeyboard(prayerID domain.PrayerID, languageCode string) *models.InlineKeyboardMarkup {
	kb := &models.InlineKeyboardMarkup{InlineKeyboard: make([][]models.InlineKeyboardButton, 3)}

	prayerName := prayerID.String()

	kb.InlineKeyboard[0] = []models.InlineKeyboardButton{
		{Text: "+1m", CallbackData: fmt.Sprintf("remind:jamaat:adjust:%s:+1m|", prayerName)},
		{Text: "+10m", CallbackData: fmt.Sprintf("remind:jamaat:adjust:%s:+10m|", prayerName)},
		{Text: "+1h", CallbackData: fmt.Sprintf("remind:jamaat:adjust:%s:+1h|", prayerName)},
	}

	kb.InlineKeyboard[1] = []models.InlineKeyboardButton{
		{Text: "-1m", CallbackData: fmt.Sprintf("remind:jamaat:adjust:%s:-1m|", prayerName)},
		{Text: "-10m", CallbackData: fmt.Sprintf("remind:jamaat:adjust:%s:-10m|", prayerName)},
		{Text: "-1h", CallbackData: fmt.Sprintf("remind:jamaat:adjust:%s:-1h|", prayerName)},
	}

	kb.InlineKeyboard[2] = []models.InlineKeyboardButton{
		{Text: buttonBack, CallbackData: "remind:back:jamaat|"},
	}

	return kb
}

func emptyButton() models.InlineKeyboardButton {
	return models.InlineKeyboardButton{Text: " ", CallbackData: emptyQuery.String()}
}
