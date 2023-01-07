package handler

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	objs "github.com/SakoDroid/telego/objects"
	"github.com/escalopa/gopray/pkg/prayer"
	"github.com/fbiville/markdown-table-formatter/pkg/markdown"
)

func (h *Handler) GetPrayers(u *objs.Update) {
	// Get the prayers for today
	prayers, err := h.ac.GetPrayers()
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An error occurred while getting prayers. Please try again later.", 0)
		return
	}

	// Send the prayers to the user
	message := fmt.Sprintf("```%s```", prayrify(prayers))
	_, err = h.b.SendMessage(u.Message.Chat.Id, message, "MarkDownV2", 0, false, false)
	if err != nil {
		log.Println(err)
	}
}

func (h *Handler) Getprayersdate(u *objs.Update) {
	chatID := strconv.Itoa(u.Message.Chat.Id)
	ch, err := h.b.AdvancedMode().RegisterChannel(chatID, "message")
	defer h.b.AdvancedMode().UnRegisterChannel(chatID, "message")
	if err != nil {
		return
	}

	// Send a message to the user to ask for the date
	_, err = h.b.SendMessage(u.Message.Chat.Id, "Please insert date in the format of <u>DD/MM</u> or <u>DD-MM</u>.\nExample: <b>9/10</b>", "HTML", 0, false, false)
	if err != nil {
		log.Println(err)
		return
	}
	u = <-*ch

	// Get the prayers for the date
	prayers, err := h.ac.Getprayersdate(u.Message.Text)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An error occurred while getting prayers. Please try again later.", 0)
		return
	}

	// Send the prayers to the user
	message := fmt.Sprintf("```%s```", prayrify(prayers))
	_, err = h.b.SendMessage(u.Message.Chat.Id, message, "MarkDownV2", 0, false, false)
	if err != nil {
		log.Println(err)
	}
}

const (
	// PrayerTimeFormat is the format of the prayer times
	prayerTimeFormat = "15:04"
)

// prayrify returns a string representation of the prayer times in a Markdown table format.
func prayrify(p prayer.PrayerTimes) string {
	// Get the day and monthName
	s := strings.Split(p.Date, "/")
	day, _ := strconv.Atoi(s[0])
	month, _ := strconv.Atoi(s[1])
	monthName := time.Month(month)

	// Create a Markdown table with the prayer times
	basicTable, err := markdown.NewTableFormatterBuilder().
		WithPrettyPrint().
		Build("Prayer", "Time").
		Format([][]string{
			{"Fajr", p.Fajr.Format(prayerTimeFormat)},
			{"Sunrise", p.Sunrise.Format(prayerTimeFormat)},
			{"Dhuhr", p.Dhuhr.Format(prayerTimeFormat)},
			{"Asr", p.Asr.Format(prayerTimeFormat)},
			{"Maghrib", p.Maghrib.Format(prayerTimeFormat)},
			{"Isha", p.Isha.Format(prayerTimeFormat)},
		})
	if err != nil {
		log.Println(err)
	}

	return fmt.Sprintf("\nDay %d %s 🕌\n>\n%s>", day, monthName, basicTable)
}
