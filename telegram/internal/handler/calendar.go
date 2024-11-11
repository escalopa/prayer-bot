package handler

import (
	log "github.com/catalystgo/logger/cli"
	"strconv"
	"time"

	"github.com/SakoDroid/telego"
	objs "github.com/SakoDroid/telego/objects"
	"github.com/escalopa/gopray/telegram/internal/domain"
)

// newCalendar creates a new calendar. The callback function is called when the user selects a date.
// The date is passed as two integers, day and month.
func (h *Handler) newCalendar(chatID int, callBack func(time.Time)) telego.MarkUps {
	kb := h.bot.CreateInlineKeyboard()
	months := h.getChatScript(chatID).GetMonthNames()

	for i, month := range months {
		row := (i / 3) + 1 // 3 buttons(months) per row.
		kb.AddCallbackButtonHandler(month, strconv.Itoa(i+1), row, h.getCalendarKeyboardCallback(chatID, callBack))
	}

	return kb
}

func (h *Handler) getCalendarKeyboardCallback(chatID int, callBack func(time.Time)) func(update *objs.Update) {
	return func(u *objs.Update) {
		var (
			kb = h.bot.CreateInlineKeyboard()

			monthDigit, _ = strconv.Atoi(u.CallbackQuery.Data)
			month         = time.Month(monthDigit)
			year          = time.Now().In(domain.GetLocation()).Year()

			daysInMonth = daysIn(month, year)
		)

		var (
			j   int
			row int
		)

		for j = range daysInMonth {
			row = (j / 5) + 1 // 5 buttons(days) per row.
			kb.AddCallbackButtonHandler(strconv.Itoa(j+1), strconv.Itoa(j+1), row, func(u1 *objs.Update) {
				day, _ := strconv.Atoi(u1.CallbackQuery.Data)
				callBack(domain.Time(day, month, year))
			})
		}

		// Add empty callback buttons
		for (j+1)%5 != 0 {
			kb.AddCallbackButtonHandler(" ", " ", row, func(_ *objs.Update) { /* empty button to fill row */ })
			j++
		}

		editor := h.bot.GetMsgEditor(u.CallbackQuery.Message.Chat.Id)
		_, err := editor.EditText(
			u.CallbackQuery.Message.MessageId,
			h.getChatScript(chatID).DatePickerStart,
			"",
			"",
			nil,
			false,
			kb,
		)
		if err != nil {
			log.Errorf("Handler.getCalendarKeyboardCallback: [%d] => %v", chatID, err)
			callBack(time.Time{}) // Cancel the calendar.
		}
	}
}

// daysIn returns the number of days in a month.
// Month is incremented by 1 because to get the last day of the previous month.
func daysIn(m time.Month, year int) int {
	return time.Date(year, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
