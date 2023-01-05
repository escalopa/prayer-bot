package application

import (
	p "github.com/escalopa/gopray/pkg/prayer"
)

type PrayerRepository interface {
	StorePrayer(date string, times p.PrayerTimes) error // Date format: dd:mm
	GetPrayer(date string) (p.PrayerTimes, error)       // Date format: dd:mm
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
	Subscribe(id int) error
	Unsubscribe(id int) error
}
