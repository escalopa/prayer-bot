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

// maxConcurrentReminderSends bounds how many chats are processed in parallel
// within a single bot. It caps goroutine fan-out and smooths bursts against
// Telegram's per-bot rate limits when the subscriber count is large.
const maxConcurrentReminderSends = 32

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
		return nil, fmt.Errorf("bot config not found for bot_id %d", botID)
	}

	b, err := bot.New(botConfig.Token)
	if err != nil {
		return nil, fmt.Errorf("create bot for bot_id %d: %w", botID, err)
	}

	h.bots[botID] = b
	return b, nil
}

func (h *Handler) Handle(ctx context.Context, botID int64) error {
	chatIDs, err := h.db.GetSubscribers(ctx, botID)
	if err != nil {
		logReminder("Handle.getSubscribers", "db GetSubscribers failed", log.Err(err), log.BotID(botID))
		return domain.ErrInternal
	}

	if len(chatIDs) == 0 {
		return nil
	}

	chats, err := h.db.GetChatsByIDs(ctx, botID, chatIDs)
	if err != nil {
		logReminder("Handle.getChatsByIDs", "db GetChatsByIDs failed", log.Err(err), log.BotID(botID))
		return domain.ErrInternal
	}

	b, err := h.getBot(botID)
	if err != nil {
		logReminder("Handle.getBot", "failed to get telegram bot client", log.Err(err), log.BotID(botID))
		return domain.ErrInternal
	}

	cfg := h.cfg[botID]
	now := h.now(cfg.Location.V())
	y, m, d := now.Date()
	date := time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	prayerDay, err := h.db.GetPrayerDay(ctx, botID, date)
	if err != nil {
		logReminder("Handle.getPrayerDay", "db GetPrayerDay failed", log.Err(err), log.BotID(botID))
		return domain.ErrInternal
	}

	reminders := []ReminderType{
		&TomorrowReminder{
			lp:              h.lp,
			botConfig:       h.cfg,
			formatPrayerDay: h.formatPrayerDay,
		},
		&SoonReminder{lp: h.lp, botConfig: h.cfg},
		&ArriveReminder{lp: h.lp},
	}

	errG := &errgroup.Group{}
	errG.SetLimit(maxConcurrentReminderSends)
	for _, chat := range chats {
		chat := chat
		errG.Go(func() error {
			if chat.Reminder == nil { // cannot happen but just in case
				return nil
			}

			for _, reminder := range reminders {
				shouldSend, prayerID := reminder.ShouldTrigger(ctx, chat, prayerDay, now)
				if !shouldSend {
					continue
				}

				messageID, err := reminder.Send(ctx, b, chat, prayerID, prayerDay)
				if err != nil {
					if isBlockedErr(err) {
						h.deleteChat(ctx, chat)
						return nil
					}
					logReminder("Handle.sendReminder", "telegram send failed",
						log.Err(err),
						log.BotID(chat.BotID),
						log.ChatID(chat.ChatID),
						log.String("reminder_type", reminder.Name().String()),
					)
					continue
				}

				err = h.db.UpdateReminder(ctx, chat.BotID, chat.ChatID, reminder.Name(), messageID, now)
				if err != nil {
					logReminder("Handle.updateReminderState", "db UpdateReminder failed",
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
