package notifier

import (
	app "github.com/escalopa/gopray/telegram/internal/application"
)

type Notifier struct {
	pr app.PrayerRepository
	sr app.SubscriberRepository
	lr app.LanguageRepository
}

func New(pr app.PrayerRepository, sr app.SubscriberRepository, lr app.LanguageRepository) *Notifier {
	return &Notifier{
		pr: pr,
		sr: sr,
		lr: lr,
	}
}

func (n *Notifier) Notify() {

}

func (n *Notifier) Subscribe(id int) error {
	return nil
}

func (n *Notifier) Unsubscribe(id int) error {
	return nil
}
