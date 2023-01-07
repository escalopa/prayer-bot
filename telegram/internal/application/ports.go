package application

import (
	p "github.com/escalopa/gopray/pkg/prayer"
)

type PrayerRepository interface {
	StorePrayer(times p.PrayerTimes) error
	GetPrayer(day, month int) (p.PrayerTimes, error)
}

type SubscriberRepository interface {
	StoreSubscriber(id int) error
	RemoveSubscribe(id int) error
	GetSubscribers() ([]int, error)
}

type LanguageRepository interface {
	GetLang(id int) (string, error)
	SetLang(id int, lang string) error
}

type Parser interface {
	ParseSchedule() error
}

type Notifier interface {
	NotifyPrayers(func(id []int, msg string)) error
	NotifyGomaa(func(id []int, msg string)) error
	Subscribe(id int) error
	Unsubscribe(id int) error
}
