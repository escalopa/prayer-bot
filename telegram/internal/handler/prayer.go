package handler

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	objs "github.com/SakoDroid/telego/objects"
	"github.com/escalopa/gopray/pkg/core"
	"github.com/fbiville/markdown-table-formatter/pkg/markdown"
)

func (h *Handler) GetPrayers(u *objs.Update) {
	// Get the prayers for today
	prayers, err := h.u.GetPrayers()
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An error occurred while getting prayers. Please try again later.", 0)
		return
	}

	// Send the prayers to the user
	message := fmt.Sprintf("```%s```", prayrify(prayers))
	_, err = h.b.SendMessage(u.Message.Chat.Id, message, "MarkDownV2", 0, false, false)
	if err != nil {
		log.Printf("Error: %s, Failed to sned default prayers table", err)
	}
}

func (h *Handler) GetPrayersByDate(u *objs.Update) {
	chatID := strconv.Itoa(u.Message.Chat.Id)
	ch, err := h.b.AdvancedMode().RegisterChannel(chatID, "message")
	defer h.b.AdvancedMode().UnRegisterChannel(chatID, "message")
	if err != nil {
		return
	}

	// TODO:
	// 	Replace messages with buttons and inline keyboard.
	//  The keyboard should be deleted after the user sends the date.
	// Send a message to the user to ask for the date
	o, err := h.b.SendMessage(u.Message.Chat.Id, "Please insert date in the format of <u>DD/MM</u> or <u>DD-MM</u>.\nExample: <b>9/10</b>", "HTML", 0, false, false)
	if err != nil {
		log.Printf("Error: %s, Failed to send date format request", err)
		return
	}

	// Delete the message if the user sends the date or if the context times out
	//defer func(messageID int) {
	//	if err == nil {
	//		h.deleteMessage(u.Message.Chat.Id, messageID)
	//	}
	//}(o.Result.MessageId)

	// Wait for the user to send the date or timeout after 10 minutes
	ctx, cancel := context.WithTimeout(h.c, 10*time.Minute)
	defer cancel()

	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	case u = <-*ch:
	}

	// Get the prayers for the date
	prayers, err := h.u.GetPrayersDate(u.Message.Text)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An error occurred while getting prayers. Please try again.", 0)
		return
	}

	defer func() {
		if err == nil {
			h.deleteMessage(u.Message.Chat.Id, o.Result.MessageId)
		}
	}()

	// Send the prayers to the user
	message := fmt.Sprintf("```%s```", prayrify(prayers))
	_, err = h.b.SendMessage(u.Message.Chat.Id, message, "MarkDownV2", 0, false, false)
	if err != nil {
		log.Printf("Error: %s, Failed to send the date prayers table", err)
	}
}

const (
	// PrayerTimeFormat is the format of the prayer times
	prayerTimeFormat = "15:04"
)

// prayrify returns a string representation of the prayer times in a Markdown table format.
func prayrify(p core.PrayerTimes) string {
	// Get the day and monthName
	monthName := time.Month(p.Month)

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
		log.Printf("Error: %s, Failed to prayrify table", err)
	}

	return fmt.Sprintf("\nDay %d %s 🕌\n>\n%s>", p.Day, monthName, basicTable)
}
