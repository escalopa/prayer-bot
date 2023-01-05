package handler

import (
	"fmt"
	"log"
	"regexp"
	"strconv"

	objs "github.com/SakoDroid/telego/objects"
)

func (h *Handler) GetPrayers(u *objs.Update) {
	// Get the prayers for today
	prayers, err := h.ac.GetPrayers()
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An unexpected error occurred while getting prayers. Please try again later.", 0)
		return
	}

	// Send the prayers to the user
	message := fmt.Sprintf("---\n%s\n---", prayers.EnHTML())
	_, err = h.b.SendMessage(u.Message.Chat.Id, message, "HTML", 0, false, false)
	if err != nil {
		log.Println(err)
	}
}

func (h *Handler) GetPrayersByDate(u *objs.Update) {
	chatID := strconv.Itoa(u.Message.Chat.Id)
	ch, err := h.b.AdvancedMode().RegisterChannel(chatID, "message")
	defer h.b.AdvancedMode().UnRegisterChannel(chatID, "message")
	if err != nil {
		return
	}

	// Send a message to the user to ask for the date
	h.b.SendMessage(u.Message.Chat.Id, "Please send the date in the format of <u>DD/MM</u> or <u>DD-MM</u>", "HTML", 0, false, false)
	u = <-*ch
	date, ok := parseDate(u.Message.Text)
	if !ok {
		h.simpleSend(u.Message.Chat.Id, "Invalid date format. Please try again. /prayersByDate", 0)
		return
	}

	// Get the prayers for the date
	prayers, err := h.ac.GetPrayersByDate(date)
	if err != nil {
		h.simpleSend(u.Message.Chat.Id, "An unexpected error occurred while getting prayers. Please try again later.", 0)
		return
	}

	// Send the prayers to the user
	message := fmt.Sprintf("---\n%s\n---", prayers.EnHTML())
	_, err = h.b.SendMessage(u.Message.Chat.Id, message, "HTML", 0, false, false)
	if err != nil {
		log.Println(err)
	}
}

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
