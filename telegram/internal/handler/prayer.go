package handler

import (
	"fmt"
	"log"
	"regexp"
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
	h.b.SendMessage(u.Message.Chat.Id, "Please insert date in the format of <u>DD/MM</u> or <u>DD-MM</u>", "HTML", 0, false, false)
	u = <-*ch
	date, ok := parseDate(u.Message.Text)
	if !ok {
		h.simpleSend(u.Message.Chat.Id, "Invalid date format. Please try again. /prayersdate", 0)
		return
	}

	// Get the prayers for the date
	prayers, err := h.ac.Getprayersdate(date)
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
			{"Fajr", fmt.Sprintf("%d:%d", p.Fajr.Hour(), p.Fajr.Minute())},
			{"Sunrise", fmt.Sprintf("%d:%d", p.Sunrise.Hour(), p.Sunrise.Minute())},
			{"Dhuhr", fmt.Sprintf("%d:%d", p.Dhuhr.Hour(), p.Dhuhr.Minute())},
			{"Asr", fmt.Sprintf("%d:%d", p.Asr.Hour(), p.Asr.Minute())},
			{"Maghrib", fmt.Sprintf("%d:%d", p.Maghrib.Hour(), p.Maghrib.Minute())},
			{"Isha", fmt.Sprintf("%d:%d", p.Isha.Hour(), p.Isha.Minute())},
		})
	if err != nil {
		log.Println(err)
	}

	return fmt.Sprintf("-\nDay %d %s 🕌\n>\n%s>", day, monthName, basicTable)
}

// // prayrify returns a string representation of the prayer times in html human ready format.
// func prayrifyHTML(p prayer.PrayerTimes) string {
// 	s := strings.Split(p.Date, "/")
// 	day, _ := strconv.Atoi(s[0])
// 	month, _ := strconv.Atoi(s[1])
// 	monthName := time.Month(month)

// 	return fmt.Sprintf(`
// 	<b>Date</b>:   %d %s

// 	<b>Fajr</b>:              %d:%d
// 	<b>Sunrise</b>:       %d:%d
// 	<b>Dhuhr</b>:       %d:%d
// 	<b>Asr</b>:             %d:%d
// 	<b>Maghrib</b>:   %d:%d
// 	<b>Isha</b>:           %d:%d
// 	`, day, monthName,
// 		p.Fajr.Hour(), p.Fajr.Minute(),
// 		p.Sunrise.Hour(), p.Sunrise.Minute(),
// 		p.Dhuhr.Hour(), p.Dhuhr.Minute(),
// 		p.Asr.Hour(), p.Asr.Minute(),
// 		p.Maghrib.Hour(), p.Maghrib.Minute(),
// 		p.Isha.Hour(), p.Isha.Minute())

// }

// parseDate parses the date
// @param date: The date to parse
// @return: The date in the format of DD/MM
// @return: True if the date is valid, false otherwise
func parseDate(date string) (string, bool) {
	// Split the date by /, - or .
	re := regexp.MustCompile(`(\/|-|\.)`)
	nums := re.Split(date, -1)
	if len(nums) != 2 {
		return "", false
	}
	// Check if the day is valid and between 1 and 31
	day, err := strconv.Atoi(nums[0])
	if err != nil || day > 31 || day < 1 {
		return "", false
	}
	// Check if the month is valid and between 1 and 12
	month, err := strconv.Atoi(nums[1])
	if err != nil || month > 12 || month < 1 {
		return "", false
	}
	return fmt.Sprintf("%d/%d", day, month), true
}
