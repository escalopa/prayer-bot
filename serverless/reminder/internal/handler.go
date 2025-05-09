package internal

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/escalopa/prayer-bot/log"

	"github.com/escalopa/prayer-bot/domain"
)

type (
	DB interface {
		GetPrayerDay(ctx context.Context, botID int64, date time.Time) (*domain.PrayerDay, error)
		GetSubscribers(ctx context.Context, botID int64) ([]int64, error)
		GetSubscribersByOffset(ctx context.Context, botID int64, offset int32) ([]int64, error)
	}

	Queue interface {
		Enqueue(ctx context.Context, payload *domain.Payload) error
	}

	Handler struct {
		cfg   map[int64]*domain.BotConfig
		db    DB
		queue Queue
	}
)

func NewHandler(cfg map[int64]*domain.BotConfig, db DB, queue Queue) *Handler {
	return &Handler{
		cfg:   cfg,
		db:    db,
		queue: queue,
	}

}

func (h *Handler) Do(ctx context.Context, botID int64) error {
	prayerID, left, err := h.getPrayer(ctx, botID, h.cfg[botID].Location.V())
	if err != nil {
		return err
	}

	var chatIDs []int64
	switch {
	case left == 0:
		chatIDs, err = h.db.GetSubscribers(ctx, botID)
	case slices.Contains(domain.ReminderOffsets(), left):
		chatIDs, err = h.db.GetSubscribersByOffset(ctx, botID, left)
	}

	if err != nil {
		return err
	}
	if len(chatIDs) == 0 {
		return nil
	}

	payload := &domain.Payload{
		Type: domain.PayloadTypeReminder,
		Data: &domain.ReminderPayload{
			BotID:          botID,
			ChatIDs:        chatIDs,
			PrayerID:       prayerID,
			ReminderOffset: left,
		},
	}

	err = h.queue.Enqueue(ctx, payload)
	if err != nil {
		return fmt.Errorf("enqueue reminder payload: %v", err)
	}

	return nil
}

func (h *Handler) getPrayer(ctx context.Context, botID int64, loc *time.Location) (domain.PrayerID, int32, error) {
	date := domain.Now(loc)
	prayerDay, err := h.db.GetPrayerDay(ctx, botID, date)
	if err != nil {
		log.Error("get prayer day",
			log.Err(err),
			log.BotID(botID),
			log.String("date", date.String()),
			log.String("location", loc.String()),
		)
		return 0, 0, fmt.Errorf("get prayer day: %v", err)
	}

	switch {
	case prayerDay.Fajr.After(date) || prayerDay.Fajr.Equal(date):
		return domain.PrayerIDFajr, int32(prayerDay.Fajr.Sub(date).Minutes()), nil
	case prayerDay.Shuruq.After(date) || prayerDay.Shuruq.Equal(date):
		return domain.PrayerIDShuruq, int32(prayerDay.Shuruq.Sub(date).Minutes()), nil
	case prayerDay.Dhuhr.After(date) || prayerDay.Dhuhr.Equal(date):
		return domain.PrayerIDDhuhr, int32(prayerDay.Dhuhr.Sub(date).Minutes()), nil
	case prayerDay.Asr.After(date) || prayerDay.Asr.Equal(date):
		return domain.PrayerIDAsr, int32(prayerDay.Asr.Sub(date).Minutes()), nil
	case prayerDay.Maghrib.After(date) || prayerDay.Maghrib.Equal(date):
		return domain.PrayerIDMaghrib, int32(prayerDay.Maghrib.Sub(date).Minutes()), nil
	case prayerDay.Isha.After(date) || prayerDay.Isha.Equal(date):
		return domain.PrayerIDIsha, int32(prayerDay.Isha.Sub(date).Minutes()), nil
	}

	// if no prayer time is found, return the first prayer of the next day
	prayerDay, err = h.db.GetPrayerDay(ctx, botID, date.AddDate(0, 0, 1))
	if err != nil {
		log.Error("get next prayer day",
			log.Err(err),
			log.BotID(botID),
			log.String("date", date.String()),
			log.String("location", loc.String()),
		)
		return 0, 0, fmt.Errorf("get next prayer day: %v", err)
	}

	return domain.PrayerIDFajr, int32(prayerDay.Fajr.Sub(date).Minutes()), nil
}
