package handler

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/escalopa/prayer-bot/log"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"golang.org/x/sync/errgroup"
)

type (
	DB interface {
		GetChatsByIDs(ctx context.Context, botID int64, chatIDs []int64) (chats []*domain.Chat, _ error)
		GetSubscribers(ctx context.Context, botID int64) (chatIDs []int64, _ error)
		GetSubscribersByOffset(ctx context.Context, botID int64, offset int32) (chatIDs []int64, _ error)
		GetPrayerDay(ctx context.Context, botID int64, date time.Time) (prayerDay *domain.PrayerDay, _ error)
		SetReminderMessageID(ctx context.Context, botID int64, chatID int64, reminderMessageID int32) error
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
	lp, err := newLanguageProvider()
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
	cfg := h.cfg[botID]
	prayerID, reminderOffset, err := h.getPrayer(ctx, botID, cfg.Location.V())
	if err != nil {
		log.Error("get prayer", log.Err(err), log.BotID(botID))
		return fmt.Errorf("get prayer: %v", err)
	}

	var chatIDs []int64
	switch {
	case reminderOffset == 0:
		chatIDs, err = h.db.GetSubscribers(ctx, botID)
	case slices.Contains(domain.ReminderOffsets(), reminderOffset):
		chatIDs, err = h.db.GetSubscribersByOffset(ctx, botID, reminderOffset)
	}

	if err != nil {
		log.Error("get subscribers", log.Err(err), log.BotID(botID))
		return fmt.Errorf("get subscribers: %v", err)
	}
	if len(chatIDs) == 0 {
		return nil
	}

	err = h.remindUsers(ctx, botID, chatIDs, prayerID, reminderOffset)
	if err != nil {
		return fmt.Errorf("remindUsers: %v", err)
	}

	return nil
}

func (h *Handler) getPrayer(ctx context.Context, botID int64, loc *time.Location) (domain.PrayerID, int32, error) {
	date := h.now(loc)
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

func (h *Handler) remindUsers(
	ctx context.Context,
	botID int64,
	chatIDs []int64,
	prayerID domain.PrayerID,
	reminderOffset int32,
) error {
	chats, err := h.db.GetChatsByIDs(ctx, botID, chatIDs)
	if err != nil {
		log.Error("get chats", log.Err(err), log.BotID(botID))
		return fmt.Errorf("get chats: %v", err)
	}

	b, err := h.getBot(botID)
	if err != nil {
		log.Error("get bot", log.Err(err), log.BotID(botID))
		return fmt.Errorf("get bot: %v", err)
	}

	errG := &errgroup.Group{}
	for _, chat := range chats {
		chat := chat
		errG.Go(func() error {
			err := h.remindUser(ctx, b, chat, prayerID, reminderOffset)
			if err != nil {
				log.Error("remindUsers",
					log.Err(err),
					log.BotID(chat.BotID),
					log.ChatID(chat.ChatID),
					log.String("prayer_id", strconv.Itoa(int(prayerID))),
					log.String("reminder_offset", strconv.Itoa(int(reminderOffset))),
				)
			}
			return nil
		})
	}

	_ = errG.Wait()
	return nil
}

func (h *Handler) remindUser(
	ctx context.Context,
	b *bot.Bot,
	chat *domain.Chat,
	prayerID domain.PrayerID,
	reminderOffset int32,
) error {
	var (
		text            = h.lp.GetText(chat.LanguageCode)
		duration        = time.Duration(reminderOffset) * time.Minute
		prayer, message = text.Prayer[int(prayerID)], ""
	)

	switch {
	case duration == 0:
		message = fmt.Sprintf(text.PrayerArrived, prayer)
	default:
		message = fmt.Sprintf(text.PrayerSoon, prayer, domain.FormatDuration(duration))
	}

	res, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chat.ChatID,
		Text:      message,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Error("remindUser: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("remindUser: send message: %v", err)
	}

	err = h.db.SetReminderMessageID(ctx, chat.BotID, chat.ChatID, int32(res.ID))
	if err != nil {
		log.Error("remindUser: set remind_message_id", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("remindUser: set remind_message_id: %v", err)
	}

	if chat.ReminderMessageID == 0 { // no message to delete
		return nil
	}

	_, err = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chat.ChatID,
		MessageID: int(chat.ReminderMessageID),
	})
	if err != nil {
		log.Error("remindUser: delete message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("remindUser: delete message: %v", err)
	}

	return nil
}

func (h *Handler) now(loc *time.Location) time.Time {
	now := time.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, loc)
}
