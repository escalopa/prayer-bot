package application

import (
	"context"
	"time"

	"github.com/escalopa/gopray/telegram/internal/domain"
)

type UseCase struct {
	ctx context.Context

	loc *time.Location

	sc  Scheduler
	pr  PrayerRepository
	scr ScriptRepository
	hr  HistoryRepository
	lr  LanguageRepository
	sr  SubscriberRepository
}

func NewUseCase(
	ctx context.Context,
	loc *time.Location,
	sc Scheduler,
	prayerRepo PrayerRepository,
	scriptRepo ScriptRepository,
	historyRepo HistoryRepository,
	languageRepo LanguageRepository,
	subscriberRepo SubscriberRepository,
) *UseCase {
	uc := &UseCase{
		ctx: ctx,

		loc: loc,

		sc:  sc,
		pr:  prayerRepo,
		scr: scriptRepo,
		hr:  historyRepo,
		lr:  languageRepo,
		sr:  subscriberRepo,
	}

	return uc
}

func (uc *UseCase) GetPrayers() (*domain.PrayerTime, error) {
	return uc.pr.GetPrayer(uc.ctx, uc.now())
}

func (uc *UseCase) GetPrayersDate(day time.Time) (*domain.PrayerTime, error) {
	return uc.pr.GetPrayer(uc.ctx, day)
}

func (uc *UseCase) SchedulePrayers(notifier Notifier) {
	uc.sc.Run(uc.ctx, notifier)
}

func (uc *UseCase) Subscribe(ctx context.Context, chatID int) error {
	return uc.sr.StoreSubscriber(ctx, chatID)
}

func (uc *UseCase) Unsubscribe(ctx context.Context, chatID int) error {
	return uc.sr.RemoveSubscribe(ctx, chatID)
}

func (uc *UseCase) GetSubscribers(ctx context.Context) ([]int, error) {
	return uc.sr.GetSubscribers(ctx)
}

func (uc *UseCase) SetLang(ctx context.Context, chatID int, lang string) error {
	return uc.lr.SetLang(ctx, chatID, lang)
}

func (uc *UseCase) GetLang(ctx context.Context, chatID int) (string, error) {
	return uc.lr.GetLang(ctx, chatID)
}

func (uc *UseCase) GetPrayerMessageID(ctx context.Context, chatDI int) (int, error) {
	return uc.hr.GetPrayerMessageID(ctx, chatDI)
}

func (uc *UseCase) StorePrayerMessageID(ctx context.Context, chatDI int, messageID int) error {
	return uc.hr.StorePrayerMessageID(ctx, chatDI, messageID)
}

func (uc *UseCase) GetScript(ctx context.Context, language string) (*domain.Script, error) {
	return uc.scr.GetScript(ctx, language)
}

func (uc *UseCase) now() time.Time {
	return time.Now().In(uc.loc)
}
