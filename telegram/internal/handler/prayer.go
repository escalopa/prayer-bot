package handler

import (
	"context"
	"fmt"
	"log"
	"time"

	objs "github.com/SakoDroid/telego/objects"
	"github.com/escalopa/gopray/pkg/core"
	"github.com/fbiville/markdown-table-formatter/pkg/markdown"
)

func (h *Handler) GetPrayers(u *objs.Update) {
	// Get the prayers for today
	prayers, err := h.u.GetPrayers()
	if err != nil {
		log.Printf("failed to get prayers /today: %s", err)
		h.simpleSend(u.Message.Chat.Id, "An error occurred while getting prayers. Please try again later.", 0)
		return
	}

	// Send the prayers to the user
	formattedTable, err := prayrify(prayers)
	if err != nil {
		log.Printf("failed to format prayers table /today: %s", err)
		h.simpleSend(u.Message.Chat.Id, "An error occurred while getting prayers. Please try again later.", 0)
		return
	}
	message := fmt.Sprintf("```%s```", formattedTable)
	_, err = h.b.SendMessage(u.Message.Chat.Id, message, "MarkDownV2", 0, false, false)
	if err != nil {
		log.Printf("failed to send the prayers table: %s", err)
	}
}

func (h *Handler) GetPrayersByDate(u *objs.Update) {
	ctx, cancel := context.WithTimeout(h.userCtx[u.Message.Chat.Id].ctx, 3*time.Minute)

	var messageID int
	// Delete the message after 3 minutes. This is to avoid the message being stuck in the chat.
	go func() {
		<-ctx.Done()
		cancel()
		h.deleteMessage(u.Message.Chat.Id, messageID)
	}()

	kb := h.newCalendar(func(day, month int) {
		defer cancel()
		prayers, err := h.u.GetPrayersDate(day, month)
		if err != nil {
			log.Printf("failed to get prayers on /date: %s", err)
			h.simpleSend(u.Message.Chat.Id, "An error occurred while getting prayers. Please try again.", 0)
			return
		}

		// Send the prayers to the user
		formattedTable, err := prayrify(prayers)
		if err != nil {
			log.Printf("failed to format prayers table on /date: %s", err)
			h.simpleSend(u.Message.Chat.Id, "An error occurred while getting prayers. Please try again.", 0)
			return
		}
		message := fmt.Sprintf("```%s```", formattedTable)
		_, err = h.b.SendMessage(u.Message.Chat.Id, message, "MarkDownV2", 0, false, false)
		if err != nil {
			log.Printf("failed to send the prayers table /date: %s", err)
		}
	})

	// Send a message to the user to ask for the date
	r, err := h.b.AdvancedMode().ASendMessage(
		u.Message.Chat.Id,
		"Please choose date",
		"",
		0,
		false,
		false,
		nil,
		false,
		false,
		kb,
	)
	if err != nil {
		log.Printf("failed to send message /date: %s", err)
		return
	}
	messageID = r.Result.MessageId
}

const (
	// PrayerTimeFormat is the format of the prayer times
	prayerTimeFormat = "15:04"
)

// prayrify returns a string representation of the prayer times in a Markdown table format.
func prayrify(p core.PrayerTimes) (string, error) {
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
		log.Printf("failed to format the prayer times table: %s", err)
		return "", err
	}
	// Return the formatted table
	// Get the day and monthName
	monthName := time.Month(p.Month).String()
	formattedTable := fmt.Sprintf("\nDay %d %s 🕌\n>\n%s>", p.Day, monthName, basicTable)
	return formattedTable, nil
}
