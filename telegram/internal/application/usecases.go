package application

import (
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/escalopa/gopray/pkg/core"
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

func (uc *UseCase) GetPrayers() (core.PrayerTimes, error) {
	p, err := uc.pr.GetPrayer(time.Now().Day(), int(time.Now().Month()))
	if err != nil {
		return core.PrayerTimes{}, errors.Wrap(err, "failed to get prayer")
	}
	return p, nil
}

func (uc *UseCase) Getprayersdate(date string) (core.PrayerTimes, error) {
	day, month, ok := parseDate(date)
	if !ok {
		return core.PrayerTimes{}, errors.New("invalid date")
	}

	p, err := uc.pr.GetPrayer(day, month)
	if err != nil {
		return core.PrayerTimes{}, errors.Wrap(err, "failed to get prayer by date")
	}
	return p, nil
}

func (uc *UseCase) Notify(send func(id int, msg string)) {
	// Notify gomaa
	go func() {
		err := uc.n.NotifyGomaa(func(ids []int, message string) {
			for _, id := range ids {
				send(id, message)
			}
		})
		if err != nil {
			log.Printf("Notifiy Gomma has stoped with error: %v", err)
		}
	}()
	// Notify prayers
	err := uc.n.NotifyPrayers(func(ids []int, message string) {
		for _, id := range ids {
			send(id, message)
		}
	})
	if err != nil {
		log.Printf("Notify Prayers has stoped with error: %v", err)
	}
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
func parseDate(date string) (day, month int, ok bool) {
	// Split the date by /, - or .
	re := regexp.MustCompile(`(\/|-|\.)`)
	nums := re.Split(date, -1)
	if len(nums) != 2 {
		return 0, 0, false
	}

	var err error
	// Check if the day is valid and between 1 and 31
	day, err = strconv.Atoi(nums[0])
	if err != nil || day > 31 || day < 1 {
		return 0, 0, false
	}
	// Check if the month is valid and between 1 and 12
	month, err = strconv.Atoi(nums[1])
	if err != nil || month > 12 || month < 1 {
		return 0, 0, false
	}
	// Check if the days is in the correct range for the month
	if month == 2 && day > 28 {
		return 0, 0, false
	} else if (month == 4 || month == 6 || month == 9 || month == 11) && day > 30 {
		return 0, 0, false
	}
	ok = true
	return
}
