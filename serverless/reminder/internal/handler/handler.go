package handler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/escalopa/prayer-bot/log"
	"github.com/go-telegram/bot"
	"golang.org/x/sync/errgroup"
)

type (
	DB interface {
		GetChatsByIDs(ctx context.Context, botID int64, chatIDs []int64) (chats []*domain.Chat, _ error)
		GetSubscribers(ctx context.Context, botID int64) (chatIDs []int64, _ error)
		GetPrayerDay(ctx context.Context, botID int64, date time.Time) (prayerDay *domain.PrayerDay, _ error)
		DeleteChat(ctx context.Context, botID int64, chatID int64) error
		UpdateReminder(
			ctx context.Context,
			botID int64,
			chatID int64,
			reminderType domain.ReminderType,
			messageID int,
			lastAt time.Time,
		) error
	}

	Handler struct {
		cfg map[int64]*domain.BotConfig
		db  DB
		lp  *languagesProvider

		bots   map[int64]*bot.Bot
		botsMu sync.Mutex
	}
)

func New(cfg map[int64]*domain.BotConfig, db DB) (*Handler, error) {
	lp, err := newLanguagesProvider()
	if err != nil {
		return nil, err
	}

	h := &Handler{
		cfg:  cfg,
		db:   db,
		lp:   lp,
		bots: make(map[int64]*bot.Bot),
	}

	return h, nil
}

func (h *Handler) getBot(botID int64) (*bot.Bot, error) {
	h.botsMu.Lock()
	defer h.botsMu.Unlock()

	b, ok := h.bots[botID]
	if ok {
		return b, nil
	}

	botConfig, ok := h.cfg[botID]
	if !ok {
		return nil, fmt.Errorf("bot config not found")
	}

	b, err := bot.New(botConfig.Token)
	if err != nil {
		return nil, fmt.Errorf("create bot: %v", err)
	}

	h.bots[botID] = b
	return b, nil
}

func (h *Handler) Handel(ctx context.Context, botID int64) error {
	chatIDs, err := h.db.GetSubscribers(ctx, botID)
	if err != nil {
		log.Error("get subscribers", log.Err(err), log.BotID(botID))
		return domain.ErrInternal
	}

	if len(chatIDs) == 0 {
		return nil
	}

	chats, err := h.db.GetChatsByIDs(ctx, botID, chatIDs)
	if err != nil {
		log.Error("get chats", log.Err(err), log.BotID(botID))
		return domain.ErrInternal
	}

	b, err := h.getBot(botID)
	if err != nil {
		log.Error("get bot", log.Err(err), log.BotID(botID))
		return domain.ErrInternal
	}

	cfg := h.cfg[botID]
	now := h.now(cfg.Location.V())
	date := now.Truncate(24 * time.Hour)
	prayerDay, err := h.db.GetPrayerDay(ctx, botID, date)
	if err != nil {
		log.Error("get prayer day", log.Err(err), log.BotID(botID))
		return domain.ErrInternal
	}

	reminders := []ReminderType{
		&TodayReminder{
			lp:              h.lp,
			botConfig:       h.cfg,
			formatPrayerDay: h.formatPrayerDay,
		},
		&SoonReminder{lp: h.lp},
		&ArriveReminder{lp: h.lp},
	}

	errG := &errgroup.Group{}
	for _, chat := range chats {
		chat := chat
		errG.Go(func() error {
			if chat.Reminder == nil { // cannot happen but just in case
				return nil
			}

			for _, reminder := range reminders {
				shouldSend, prayerID := reminder.Check(ctx, chat, prayerDay, now)
				if !shouldSend {
					continue
				}

				messageID, err := reminder.Send(ctx, b, chat, prayerID, prayerDay)
				if err != nil {
					if isBlockedErr(err) {
						h.deleteChat(ctx, chat)
						return nil
					}
					log.Error("send reminder",
						log.Err(err),
						log.BotID(chat.BotID),
						log.ChatID(chat.ChatID),
						log.String("reminder_type", reminder.Name().String()),
					)
					continue
				}

				now = now.Add(1 * time.Minute) // increment by 1 minute to avoid sending the same reminder multiple times
				err = h.db.UpdateReminder(ctx, chat.BotID, chat.ChatID, reminder.Name(), messageID, now)
				if err != nil {
					log.Error("update reminder state",
						log.Err(err),
						log.BotID(chat.BotID),
						log.ChatID(chat.ChatID),
						log.String("reminder_type", reminder.Name().String()),
					)
				}
			}
			return nil
		})
	}

	_ = errG.Wait()
	return nil
}
