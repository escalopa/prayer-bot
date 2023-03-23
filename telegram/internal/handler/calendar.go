package handler

import (
	"log"
	"strconv"
	"time"

	"github.com/SakoDroid/telego"
	objs "github.com/SakoDroid/telego/objects"
)

// newCalendar creates a new calendar. The callback function is called when the user selects a date.
// The date is passed as two integers, day and month.
func (h *Handler) newCalendar(callBack func(int, int)) telego.MarkUps {
	kb := h.b.CreateInlineKeyboard()
	for i := 1; i <= 12; i++ {
		row := (i-1)/3 + 1 // 3 buttons(months) per row.
		kb.AddCallbackButtonHandler(time.Month(i).String(), strconv.Itoa(i), row, func(u1 *objs.Update) {
			// Sets the language.
			kb = h.b.CreateInlineKeyboard()
			month, _ := strconv.Atoi(u1.CallbackQuery.Data)
			daysInMonth := daysIn(time.Month(month), time.Now().Year())
			for j := 1; j <= daysInMonth; j++ {
				row := (j-1)/7 + 1 // 7 buttons(days) per row.
				kb.AddCallbackButtonHandler(strconv.Itoa(j), strconv.Itoa(j), row, func(u2 *objs.Update) {
					day, _ := strconv.Atoi(u2.CallbackQuery.Data)
					callBack(day, month)
				})
			}
			editor := h.b.GetMsgEditor(u1.CallbackQuery.Message.Chat.Id)
			_, err := editor.EditText(
				u1.CallbackQuery.Message.MessageId,
				"Please choose date",
				"",
				"",
				nil,
				false,
				kb,
			)
			if err != nil {
				log.Printf("failed to edit message in calendar /date : %s", err)
				callBack(0, 0) // Cancel the calendar.
			}
		})
	}
	return kb
}

func daysIn(m time.Month, year int) int {
	return time.Date(year, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
