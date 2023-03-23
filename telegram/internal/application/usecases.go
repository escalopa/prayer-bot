package application

import (
	"context"
	"time"

	"github.com/escalopa/gopray/pkg/core"
)

type UseCase struct {
	n   Notifier
	sr  SubscriberRepository
	pr  PrayerRepository
	lr  LanguageRepository
	hr  HistoryRepository
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

func WithHistoryRepository(hr HistoryRepository) func(*UseCase) {
	return func(uc *UseCase) {
		uc.hr = hr
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

func (uc *UseCase) GetPrayersDate(day, month int) (core.PrayerTimes, error) {
	p, err := uc.pr.GetPrayer(uc.ctx, day, month)
	return p, err
}

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

func (uc *UseCase) GetPrayerMessageID(ctx context.Context, userID int) (int, error) {
	id, err := uc.hr.GetPrayerMessageID(ctx, userID)
	return id, err
}

func (uc *UseCase) StorePrayerMessageID(ctx context.Context, userID int, messageID int) error {
	err := uc.hr.StorePrayerMessageID(ctx, userID, messageID)
	return err
}
