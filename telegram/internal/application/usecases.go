package application

import (
	"context"
	"time"

	"github.com/escalopa/gopray/pkg/language"

	"github.com/escalopa/gopray/pkg/core"
)

type UseCase struct {
	n   Notifier
	sr  SubscriberRepository
	pr  PrayerRepository
	lr  LanguageRepository
	hr  HistoryRepository
	scr ScriptRepository
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

func WithScriptRepository(scr ScriptRepository) func(*UseCase) {
	return func(uc *UseCase) {
		uc.scr = scr
	}
}

func (uc *UseCase) GetPrayers() (core.PrayerTime, error) {
	now := time.Now().In(core.GetLocation())
	p, err := uc.pr.GetPrayer(uc.ctx, now)
	if err != nil {
		return core.PrayerTime{}, err
	}
	return p, nil
}

func (uc *UseCase) GetPrayersDate(day time.Time) (core.PrayerTime, error) {
	p, err := uc.pr.GetPrayer(uc.ctx, day)
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
	return uc.sr.StoreSubscriber(ctx, id)
}

func (uc *UseCase) Unsubscribe(ctx context.Context, id int) error {
	return uc.sr.RemoveSubscribe(ctx, id)
}

func (uc *UseCase) GetSubscribers(ctx context.Context) ([]int, error) {
	return uc.sr.GetSubscribers(ctx)
}

func (uc *UseCase) SetLang(ctx context.Context, id int, lang string) error {
	return uc.lr.SetLang(ctx, id, lang)
}

func (uc *UseCase) GetLang(ctx context.Context, id int) (string, error) {
	return uc.lr.GetLang(ctx, id)
}

func (uc *UseCase) GetPrayerMessageID(ctx context.Context, userID int) (int, error) {
	return uc.hr.GetPrayerMessageID(ctx, userID)
}

func (uc *UseCase) StorePrayerMessageID(ctx context.Context, userID int, messageID int) error {
	return uc.hr.StorePrayerMessageID(ctx, userID, messageID)
}

func (uc *UseCase) GetScript(ctx context.Context, language string) (*language.Script, error) {
	return uc.scr.GetScript(ctx, language)
}
