package application

import (
	"context"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/escalopa/gopray/pkg/core"
	"github.com/pkg/errors"
)

type UseCase struct {
	n   Notifier
	sr  SubscriberRepository
	pr  PrayerRepository
	lr  LanguageRepository
	loc *time.Location
	ctx context.Context
}

func New(ctx context.Context, opts ...func(*UseCase)) *UseCase {
	uc := &UseCase{ctx: ctx}
	for _, opt := range opts {
		opt(uc)
	}
	return uc
}

func WithNotifier(n Notifier) func(*UseCase) {
	return func(uc *UseCase) {
		uc.n = n
	}
}

func WithTimeLocation(loc *time.Location) func(*UseCase) {
	return func(uc *UseCase) {
		uc.loc = loc
	}
}

func WithPrayerRepository(pr PrayerRepository) func(*UseCase) {
	return func(uc *UseCase) {
		uc.pr = pr
	}
}

func WithSubscriberRepository(sr SubscriberRepository) func(*UseCase) {
	return func(uc *UseCase) {
		uc.sr = sr
	}
}

func WithLanguageRepository(lr LanguageRepository) func(*UseCase) {
	return func(uc *UseCase) {
		uc.lr = lr
	}
}

func (uc *UseCase) GetPrayers() (core.PrayerTimes, error) {
	now := time.Now().In(uc.loc)
	p, err := uc.pr.GetPrayer(uc.ctx, now.Day(), int(now.Month()))
	if err != nil {
		return core.PrayerTimes{}, err
	}
	return p, nil
}

func (uc *UseCase) GetPrayersDate(date string) (core.PrayerTimes, error) {
	day, month, err := parseDate(date)
	if err != nil {
		return core.PrayerTimes{}, errors.New("invalid date")
	}

	p, err := uc.pr.GetPrayer(uc.ctx, day, month)
	return p, err
}

// Notify TODO: Handle message with date & translation before sending to users
func (uc *UseCase) Notify(
	notifySoon func(id int, prayer, time string),
	notifyNow func(id int, prayer string),
	notifyGomaa func(ids int, time string),
) {
	// Notify gomaa
	go func() {
		uc.n.NotifyGomaa(uc.ctx,
			func(ids []int, time string) {
				for _, id := range ids {
					notifyGomaa(id, time)
				}
			})
	}()
	// Notify prayers
	go func() {
		uc.n.NotifyPrayers(uc.ctx,
			func(ids []int, prayer, time string) {
				for _, id := range ids {
					notifySoon(id, prayer, time)
				}
			}, func(ids []int, time string) {
				for _, id := range ids {
					notifyNow(id, time)
				}
			})
	}()
}

func (uc *UseCase) Subscribe(ctx context.Context, id int) error {
	err := uc.sr.StoreSubscriber(ctx, id)
	return err
}

func (uc *UseCase) Unsubscribe(ctx context.Context, id int) error {
	err := uc.sr.RemoveSubscribe(ctx, id)
	return err
}

func (uc *UseCase) GetSubscribers(ctx context.Context) ([]int, error) {
	ids, err := uc.sr.GetSubscribers(ctx)
	return ids, err
}

func (uc *UseCase) SetLang(ctx context.Context, id int, lang string) error {
	err := uc.lr.SetLang(ctx, id, lang)
	return err
}

func (uc *UseCase) GetLang(ctx context.Context, id int) (string, error) {
	lang, err := uc.lr.GetLang(ctx, id)
	return lang, err
}

// parseDate parses the date
// @param date: The date to parse
// @return: The date in the format of DD/MM
// @return: An error if the date is invalid
func parseDate(date string) (day, month int, err error) {
	// Split the date by /, - or .
	re, err := regexp.Compile(`(\/|-|\.)`)
	if err != nil {
		log.Printf("failed to compile regex: %v", err)
		return 0, 0, err
	}
	nums := re.Split(date, -1)
	if len(nums) != 2 {
		return 0, 0, errors.New("invalid date format")
	}

	// Check if the day is valid and between 1 and 31
	day, err = strconv.Atoi(nums[0])
	if err != nil || day > 31 || day < 1 {
		return 0, 0, errors.New("invalid day")
	}
	// Check if the month is valid and between 1 and 12
	month, err = strconv.Atoi(nums[1])
	if err != nil || month > 12 || month < 1 {
		return 0, 0, errors.New("invalid month")
	}
	// Check if the days is in the correct range for the month
	if month == 2 && day > 28 {
		return 0, 0, errors.New("invalid day for february")
	} else if (month == 4 || month == 6 || month == 9 || month == 11) && day > 30 {
		return 0, 0, errors.New("invalid day for one of the months 4, 6, 9, 11")
	}
	return
}
