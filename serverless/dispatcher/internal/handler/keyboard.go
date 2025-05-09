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
	days := daysInMonth(time.Month(month), now)
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

func (h *Handler) remindKeyboard() *models.InlineKeyboardMarkup {
	reminderOffsets := domain.ReminderOffsets()
	rows, empty := layoutRowsInfo(len(reminderOffsets), remindPerRow)

	kb := &models.InlineKeyboardMarkup{InlineKeyboard: make([][]models.InlineKeyboardButton, rows)}

	row := 0
	for i, offset := range reminderOffsets {
		if i%remindPerRow == 0 && i != 0 {
			row++
		}
		kb.InlineKeyboard[row] = append(kb.InlineKeyboard[row], models.InlineKeyboardButton{
			Text:         strconv.Itoa(int(offset)),
			CallbackData: fmt.Sprintf("%s%d", remindQuery, offset),
		})
	}

	for i := 0; i < empty; i++ {
		kb.InlineKeyboard[row] = append(kb.InlineKeyboard[row], emptyButton())
	}

	return kb
}

func emptyButton() models.InlineKeyboardButton {
	return models.InlineKeyboardButton{Text: " ", CallbackData: emptyQuery.String()}
}
