package application

import (
	"context"
	"time"

	"github.com/escalopa/gopray/telegram/internal/domain"
)

type Scheduler interface {
	Run(ctx context.Context, notifier Notifier)
}

type Notifier interface {
	PrayerSoon(ctx context.Context, chatIDs []int, prayer string, time string)
	PrayerNow(ctx context.Context, chatIDs []int, prayer string)
	PrayerJummah(ctx context.Context, chatIDs []int, time string)
}

type PrayerRepository interface {
	StorePrayer(ctx context.Context, times *domain.PrayerTime) error
	GetPrayer(ctx context.Context, day time.Time) (*domain.PrayerTime, error)
}

type ScriptRepository interface {
	StoreScript(ctx context.Context, language string, script *domain.Script) error
	GetScript(ctx context.Context, language string) (*domain.Script, error)
}

type HistoryRepository interface {
	GetPrayerMessageID(ctx context.Context, chatID int) (int, error)
	StorePrayerMessageID(ctx context.Context, chatID int, messageID int) error
}

type LanguageRepository interface {
	GetLang(ctx context.Context, chatID int) (string, error)
	SetLang(ctx context.Context, chatID int, lang string) error
}

type SubscriberRepository interface {
	StoreSubscriber(ctx context.Context, chatID int) error
	RemoveSubscribe(ctx context.Context, chatID int) error
	GetSubscribers(ctx context.Context) (chatIDs []int, err error)
}
