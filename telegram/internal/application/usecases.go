package application

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/escalopa/gopray/pkg/prayer"
	"github.com/pkg/errors"
)

type UseCase struct {
	n  Notifier
	pr PrayerRepository
	lr LanguageRepository
}

func New(n Notifier, pr PrayerRepository, lr LanguageRepository) *UseCase {
	return &UseCase{n: n, pr: pr, lr: lr}
}

func (uc *UseCase) GetPrayers() (prayer.PrayerTimes, error) {
	format := fmt.Sprintf("%d/%d", time.Now().Day(), time.Now().Month())
	p, err := uc.pr.GetPrayer(format)
	if err != nil {
		return prayer.PrayerTimes{}, errors.Wrap(err, "failed to get prayer")
	}
	return p, nil
}

func (uc *UseCase) Getprayersdate(date string) (prayer.PrayerTimes, error) {
	date, ok := parseDate(date)
	if !ok {
		return prayer.PrayerTimes{}, errors.New("invalid date")
	}

	p, err := uc.pr.GetPrayer(date)
	if err != nil {
		return prayer.PrayerTimes{}, errors.Wrap(err, "failed to get prayer by date")
	}
	return p, nil
}

func (uc *UseCase) Notify(send func(id int, msg string)) {
	err := uc.n.Notify(func(ids []int, message string) {
		for _, id := range ids {
			send(id, message)
		}
	})
	log.Printf("Notifier stoped with error: %v", err)
}

func (uc *UseCase) Subscribe(id int) error {
	err := uc.n.Subscribe(id)
	if err != nil {
		return errors.Wrap(err, "failed to subscribe")
	}
	return nil
}

func (uc *UseCase) Unsubscribe(id int) error {
	err := uc.n.Unsubscribe(id)
	if err != nil {
		return errors.Wrap(err, "failed to unsubscribe")
	}
	return nil
}

func (uc *UseCase) SetLang(id int, lang string) error {
	err := uc.lr.SetLang(id, lang)
	if err != nil {
		return errors.Wrap(err, "failed to set language")
	}
	return nil
}

func (uc *UseCase) GetLang(id int) (string, error) {
	lang, err := uc.lr.GetLang(id)
	if err != nil {
		return "", errors.Wrap(err, "failed to get language")
	}
	return lang, nil
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
	// Check if the days is in the correct range for the month
	if month == 2 && day > 28 {
		return "", false
	} else if (month == 4 || month == 6 || month == 9 || month == 11) && day > 30 {
		return "", false
	}
	return fmt.Sprintf("%d/%d", day, month), true
}
