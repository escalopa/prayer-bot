// Package port declares the hexagon's ports: the interfaces through which the
// core and the driving adapters reach driven technology (persistence, prayer
// calculation, geocoding, market data). Every signature speaks in domain
// types only; adapters under internal/adapter implement them, and the
// composition roots in cmd wire concrete implementations in.
package port

import (
	"context"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/domain"
)

// Calculator computes one local prayer day for a profile.
// Implemented by core/prayertime.LocalCalculator.
type Calculator interface {
	Day(context.Context, time.Time, domain.PrayerProfile) (domain.DaySchedule, error)
}

// LocationResolver turns raw coordinates into a timezone and approximate
// place. Implemented by adapter/out/location.GoogleMaps.
type LocationResolver interface {
	Resolve(context.Context, float64, float64) (domain.ResolvedLocation, error)
}

// MetalSource fetches the daily gold/silver spot prices and USD rate table
// backing the Zakat niSab. Implemented by adapter/out/metals.Client.
type MetalSource interface {
	Fetch(context.Context) (domain.MetalPrices, error)
}

// Store is the persistence port. Implemented by adapter/out/store.Store; a
// missing record is reported as domain.ErrNotFound. Narrower role interfaces
// (for example the reminder services' SenderStore) remain subsets of this
// port, so the same adapter satisfies them all.
type Store interface {
	// Webhook update idempotency.
	AcquireUpdate(ctx context.Context, updateID int64) (bool, error)
	CompleteUpdate(ctx context.Context, updateID int64) error
	FailUpdate(ctx context.Context, updateID int64, cause error) error

	// Chats.
	UpsertChat(ctx context.Context, chat domain.Chat) error
	Chat(ctx context.Context, chatID int64) (domain.Chat, error)
	SetLanguage(ctx context.Context, chatID int64, languageCode string) error
	SetJamaatPoll(ctx context.Context, chatID int64, enabled bool) error
	DeleteChat(ctx context.Context, chatID int64) error

	// Prayer profiles.
	Profile(ctx context.Context, chatID int64) (domain.PrayerProfile, error)
	UpsertProfile(ctx context.Context, profile domain.PrayerProfile) (domain.PrayerProfile, error)

	// Reminder rules and schedules.
	EnableDefaultRules(ctx context.Context, chatID int64) error
	ConfigurePrayerRules(ctx context.Context, chatID int64, enabled bool, beforeMinutes int) error
	DisableRules(ctx context.Context, chatID int64) error
	SetWeeklyRule(ctx context.Context, chatID int64, kind domain.ReminderKind, enabled bool) error
	SetWhiteDaysRule(ctx context.Context, chatID int64, enabled bool) error
	SetOccasionRule(ctx context.Context, chatID int64, kind domain.ReminderKind, enabled bool) error
	EnabledRules(ctx context.Context, chatID int64) ([]domain.ReminderRule, error)
	Rule(ctx context.Context, ruleID int64) (domain.ReminderRule, error)
	UpsertSchedule(ctx context.Context, schedule domain.ReminderSchedule) (domain.ReminderSchedule, error)
	Schedule(ctx context.Context, scheduleID int64) (domain.ReminderSchedule, error)

	// Dispatch and delivery.
	ClaimDue(ctx context.Context, now time.Time, limit int) (int, error)
	PendingOutbox(ctx context.Context, limit int) ([]domain.OutboxItem, error)
	MarkOutboxEnqueued(ctx context.Context, id int64) error
	Cleanup(ctx context.Context, now time.Time, limit int) (int64, error)
	AcquireDelivery(ctx context.Context, task domain.DeliveryTask) (bool, error)
	CompleteDelivery(ctx context.Context, task domain.DeliveryTask, messageID int64, next domain.ReminderSchedule, category string, expiresAt time.Time) (int64, error)
	ClearNotificationMessage(ctx context.Context, chatID, messageID int64) error
	MarkDeliveryStale(ctx context.Context, deliveryKey string) error
	FailDelivery(ctx context.Context, deliveryKey string, cause error) error

	// Calendar subscriptions.
	CalendarSubscription(ctx context.Context, chatID int64) (domain.CalendarSubscription, error)
	CalendarSubscriptionByToken(ctx context.Context, feedToken string) (domain.CalendarSubscription, error)
	EnableCalendarSubscription(ctx context.Context, chatID int64, feedToken, uidNamespace string) (domain.CalendarSubscription, error)
	DisableCalendarSubscription(ctx context.Context, chatID int64) error

	// Cached market data and owner metrics.
	MetalPrices(ctx context.Context) (domain.MetalPrices, error)
	UpsertMetalPrices(ctx context.Context, prices domain.MetalPrices) error
	AdminMetrics(ctx context.Context) (domain.AdminDashboard, error)
}
