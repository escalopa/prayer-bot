package internal

import (
	"fmt"
	"strconv"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/go-telegram/bot/models"
)

const (
	daysPerRow   = 5
	monthsPerRow = 3

	notifyPerRow = 5

	languagesPerRow = 2
)

func (h *Handler) languagesKeyboard() *models.InlineKeyboardMarkup {
	languages := h.lp.GetLanguages()
	rows, emptyButtons := rowsCount(len(languages), languagesPerRow)

	kb := &models.InlineKeyboardMarkup{InlineKeyboard: make([][]models.InlineKeyboardButton, rows)}

	row := 0
	for i, lang := range languages {
		if i%2 == 0 && i != 0 {
			row++
		}
		kb.InlineKeyboard[row] = append(kb.InlineKeyboard[row], models.InlineKeyboardButton{
			Text:         lang.Name,
			CallbackData: fmt.Sprintf("%s%s", callbackLanguage, lang.Code),
		})
	}

	for i := 0; i < emptyButtons; i++ {
		kb.InlineKeyboard[row] = append(kb.InlineKeyboard[row], models.InlineKeyboardButton{
			Text:         "",
			CallbackData: "",
		})
	}

	return kb
}

func (h *Handler) monthsKeyboard(languageCode string) *models.InlineKeyboardMarkup {
	months := h.lp.GetText(languageCode).GetMonths()
	rows, _ := rowsCount(len(months), languagesPerRow)

	kb := &models.InlineKeyboardMarkup{InlineKeyboard: make([][]models.InlineKeyboardButton, rows)}

	row := 0
	for i, month := range months {
		if i%2 == 0 && i != 0 {
			row++
		}
		kb.InlineKeyboard[row] = append(kb.InlineKeyboard[row], models.InlineKeyboardButton{
			Text:         month.Name,
			CallbackData: fmt.Sprintf("%s%d", callbackDateMonth, month.ID),
		})
	}

	return kb
}

func (h *Handler) daysKeyboard(now time.Time, month int) *models.InlineKeyboardMarkup {
	days := daysInMonth(month, now)
	rows, emptyButtons := rowsCount(days, daysPerRow)

	kb := &models.InlineKeyboardMarkup{InlineKeyboard: make([][]models.InlineKeyboardButton, rows)}

	row := 0
	for i := 0; i < days; i++ {
		if i%daysPerRow == 0 && i != 0 {
			row++
		}

		day := i + 1
		kb.InlineKeyboard[row] = append(kb.InlineKeyboard[row], models.InlineKeyboardButton{
			Text:         strconv.Itoa(day),
			CallbackData: fmt.Sprintf("%s%d%s%d", callbackDateDay, month, callbackDataSplitter, day),
			// CallbackData value example: date:day|12|20 => 20th December
		})
	}

	for i := 0; i < emptyButtons; i++ {
		kb.InlineKeyboard[row] = append(kb.InlineKeyboard[row], models.InlineKeyboardButton{
			Text:         "",
			CallbackData: "",
		})
	}

	return kb
}

func (h *Handler) notifyKeyboard() *models.InlineKeyboardMarkup {
	offsets := domain.NotifyOffsets()
	rows, emptyButtons := rowsCount(len(offsets), notifyPerRow)

	kb := &models.InlineKeyboardMarkup{InlineKeyboard: make([][]models.InlineKeyboardButton, rows)}

	row := 0
	for i, offset := range offsets {
		if i%notifyPerRow == 0 && i != 0 {
			row++
		}
		kb.InlineKeyboard[row] = append(kb.InlineKeyboard[row], models.InlineKeyboardButton{
			Text:         fmt.Sprintf("%dmin", offset),
			CallbackData: fmt.Sprintf("%s%d", callbackNotify, offset),
		})
	}

	for i := 0; i < emptyButtons; i++ {
		kb.InlineKeyboard[row] = append(kb.InlineKeyboard[row], models.InlineKeyboardButton{
			Text:         "",
			CallbackData: "",
		})
	}

	return kb
}
