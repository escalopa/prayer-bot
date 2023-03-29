package handler

import (
	"context"
	"fmt"
	"log"
	"time"

	objs "github.com/SakoDroid/telego/objects"
	"github.com/escalopa/gopray/pkg/core"
	"github.com/olekukonko/tablewriter"
)

func (h *Handler) GetPrayers(u *objs.Update) {
	// Get the prayers for today
	chatID := u.Message.Chat.Id
	prayers, err := h.u.GetPrayers()
	if err != nil {
		log.Printf("failed to get prayers /today: %s", err)
		h.simpleSend(chatID, h.userScript[chatID].PrayerFail, 0)
		return
	}

	// Send the prayers to the user
	table := h.prayrify(chatID, prayers)
	_, err = h.b.SendMessage(chatID, table, "MarkDownV2", 0, false, false)
	if err != nil {
		log.Printf("failed to send the prayers prayerTable: %s", err)
	}
}

func (h *Handler) GetPrayersByDate(u *objs.Update) {
	chatID := u.Message.Chat.Id
	ctx, cancel := context.WithTimeout(h.userCtx[chatID].ctx, 3*time.Minute)

	var messageID int
	// Delete the message after 3 minutes. This is to avoid the message being stuck in the chat.
	go func() {
		<-ctx.Done()
		cancel()
		h.deleteMessage(chatID, messageID)
	}()

	kb := h.newCalendar(chatID, func(day, month int) {
		defer cancel()
		prayers, err := h.u.GetPrayersDate(day, month)
		if err != nil {
			log.Printf("failed to get prayers on /date: %s", err)
			h.simpleSend(chatID, h.userScript[chatID].PrayerFail, 0)
			return
		}

		// Send the prayers to the user
		table := h.prayrify(chatID, prayers)
		_, err = h.b.SendMessage(chatID, table, "MarkDownV2", 0, false, false)
		if err != nil {
			log.Printf("failed to send the prayers prayerTable /date: %s", err)
		}
	})

	// Send a message to the user to ask for the date
	r, err := h.b.AdvancedMode().ASendMessage(
		chatID,
		h.userScript[chatID].DatePickerStart,
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

type prayerTable string

func (t *prayerTable) Write(p []byte) (n int, err error) {
	*t += prayerTable(p)
	return len(p), nil
}

// prayrify returns a string representation of the prayer times in a Markdown prayerTable format.
func (h *Handler) prayrify(chatID int, p core.PrayerTimes) string {
	// Create a Markdown prayerTable with the prayer times
	t := new(prayerTable)
	tw := tablewriter.NewWriter(t)

	// Define prayerTable headers and data
	//header := []string{h.userScript[chatID].PrayrifyTablePrayer, h.userScript[chatID].PrayrifyTableTime}
	data := [][]string{
		{h.userScript[chatID].Fajr, p.Fajr.Format(prayerTimeFormat)},
		{h.userScript[chatID].Sunrise, p.Sunrise.Format(prayerTimeFormat)},
		{h.userScript[chatID].Dhuhr, p.Dhuhr.Format(prayerTimeFormat)},
		{h.userScript[chatID].Asr, p.Asr.Format(prayerTimeFormat)},
		{h.userScript[chatID].Maghrib, p.Maghrib.Format(prayerTimeFormat)},
		{h.userScript[chatID].Isha, p.Isha.Format(prayerTimeFormat)},
	}
	//tw.SetHeader(header)
	tw.AppendBulk(data)
	tw.SetBorders(tablewriter.Border{Left: true, Right: true})
	tw.SetCenterSeparator("|")
	tw.Render()

	formattedTable := fmt.Sprintf("```\n%s %d %s 🕌\n\n%s```\n/help",
		h.userScript[chatID].PrayrifyTableDay,
		p.Day,
		h.userScript[chatID].GetMonthNames()[p.Month-1],
		string(*t),
	)
	return formattedTable
}
