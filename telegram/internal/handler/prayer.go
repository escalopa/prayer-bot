package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	objs "github.com/SakoDroid/telego/objects"
	log "github.com/catalystgo/logger/cli"
	"github.com/escalopa/gopray/telegram/internal/domain"
	"github.com/olekukonko/tablewriter"
)

const (
	// PrayerTimeFormat is the format of the prayer times
	prayerTimeFormat = "15:04"
)

func (h *Handler) GetPrayers(u *objs.Update) {
	chatID := getChatID(u)

	prayers, err := h.uc.GetPrayers()
	if err != nil {
		log.Errorf("Handler.GetPrayers: [%d] => %v", chatID, err)
		h.simpleSend(chatID, h.getChatScript(chatID).PrayerFail)
		return
	}

	// Send the prayers to the user
	table := h.prayrify(chatID, prayers)
	_, err = h.bot.SendMessage(chatID, table, "MarkDownV2", 0, false, false)
	if err != nil {
		log.Errorf("Handler.GetPrayers: [%d] => %v", chatID, err)
	}
}

func (h *Handler) GetPrayersByDate(u *objs.Update) {
	var (
		chatID = getChatID(u)

		ctx    = h.getChatCtx(chatID)
		script = h.getChatScript(chatID)
	)

	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)

	var messageID int

	// Delete the message after 3 minutes. This is to avoid the message being stuck in the chat.
	go func() {
		<-ctx.Done()
		cancel()
		h.deleteMessage(chatID, messageID)
	}()

	kb := h.newCalendar(chatID, func(day time.Time) {
		defer cancel()
		if day.IsZero() {
			h.simpleSend(chatID, script.PrayerFail)
			return
		}

		prayers, err := h.uc.GetPrayersDate(day)
		if err != nil {
			log.Errorf("Handler.GetPrayersByDate: [%d] => %v", chatID, err)
			h.simpleSend(chatID, script.PrayerFail)
			return
		}

		// Send the prayers to the user
		table := h.prayrify(chatID, prayers)
		_, err = h.bot.SendMessage(chatID, table, "MarkDownV2", 0, false, false)
		if err != nil {
			log.Errorf("Handler.GetPrayersByDate: [%d] => %v", chatID, err)
		}
	})

	// Send a message to the user to ask for the date
	r, err := h.bot.AdvancedMode().ASendMessage(
		chatID,
		script.DatePickerStart,
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
		log.Errorf("Handler.GetPrayersByDate: [%d] => %v", chatID, err)
		return
	}
	messageID = r.Result.MessageId
}

// prayrify returns a string representation of the prayer times in a Markdown prayerTable format.
// Example output:
// Day 9 November 🕌
//
// | Fajr    | 04:58 |
// | Sunrise | 07:26 |
// | Dhuhr   | 11:27 |
// | Asr     | 13:55 |
// | Maghrib | 15:47 |
// | Isha    | 17:34 |
func (h *Handler) prayrify(chatID int, p *domain.PrayerTime) string {
	script := h.getChatScript(chatID)

	// Create a Markdown prayerTable with the prayer times
	writer := &strings.Builder{}
	tw := tablewriter.NewWriter(writer)

	data := [][]string{
		{script.Fajr, p.Fajr.Format(prayerTimeFormat)},
		{script.Dohaa, p.Dohaa.Format(prayerTimeFormat)},
		{script.Dhuhr, p.Dhuhr.Format(prayerTimeFormat)},
		{script.Asr, p.Asr.Format(prayerTimeFormat)},
		{script.Maghrib, p.Maghrib.Format(prayerTimeFormat)},
		{script.Isha, p.Isha.Format(prayerTimeFormat)},
	}

	// header := []string{h.chatScript[chatID].PrayrifyTablePrayer, h.chatScript[chatID].PrayrifyTableTime}
	//tw.SetHeader(header)

	tw.AppendBulk(data)
	tw.SetBorders(tablewriter.Border{Left: true, Right: true})
	tw.SetCenterSeparator("|")
	tw.Render()

	formattedTable := fmt.Sprintf(prayerText,
		p.Day.Day(),
		script.GetMonthNames()[p.Day.Month()-1],
		writer.String(),
	)

	return formattedTable
}
