package application

import (
	"fmt"
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
	// TODO: Add language support
	return p, nil
}

func (uc *UseCase) GetPrayersByDate(date string) (prayer.PrayerTimes, error) {
	p, err := uc.pr.GetPrayer(date)
	if err != nil {
		return prayer.PrayerTimes{}, errors.Wrap(err, "failed to get prayer by date")
	}
	return p, nil
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
