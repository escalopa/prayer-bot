package handler

import (
	"context"
	"fmt"
	"strings"
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

	// Get chat details
	chats, err := h.db.GetChatsByIDs(ctx, botID, chatIDs)
	if err != nil {
		log.Error("get chats", log.Err(err), log.BotID(botID))
		return domain.ErrInternal
	}

	// Get bot
	b, err := h.getBot(botID)
	if err != nil {
		log.Error("get bot", log.Err(err), log.BotID(botID))
		return domain.ErrInternal
	}

	// Get current day prayer times
	cfg := h.cfg[botID]
	now := time.Now().In(cfg.Location.V())
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	prayerDay, err := h.db.GetPrayerDay(ctx, botID, date)
	if err != nil {
		log.Error("get prayer day", log.Err(err), log.BotID(botID))
		return domain.ErrInternal
	}

	// Initialize reminder types
	reminders := []ReminderType{
		&TodayReminder{
			lp:              h.lp,
			botConfig:       h.cfg,
			formatPrayerDay: h.formatPrayerDay,
		},
		&SoonReminder{lp: h.lp},
		&ArriveReminder{lp: h.lp},
		&JamaatReminder{lp: h.lp},
	}

	// Process each chat
	errG := &errgroup.Group{}
	for _, chat := range chats {
		chat := chat
		errG.Go(func() error {
			// Skip chats without reminder config
			if chat.Reminder == nil {
				return nil
			}

			// Check each reminder type
			for _, reminder := range reminders {
				shouldSend, prayerID := reminder.Check(ctx, chat, prayerDay, now)
				if !shouldSend {
					continue
				}

				// Send reminder
				err := reminder.Send(ctx, b, chat, prayerID, prayerDay)
				if err != nil {
					if h.isBlockedErr(err) {
						h.deleteChat(ctx, chat)
						return nil
					}
					log.Error("send reminder",
						log.Err(err),
						log.BotID(chat.BotID),
						log.ChatID(chat.ChatID),
						log.String("reminder_type", reminder.Name()),
					)
					continue
				}

				// Update state
				var messageID int
				switch reminder.Name() {
				case "today":
					messageID = chat.Reminder.Today.MessageID
				case "soon":
					messageID = chat.Reminder.Soon.MessageID
				case "arrive":
					messageID = chat.Reminder.Arrive.MessageID
				case "jamaat":
					// Jamaat is stateless, no message ID to track
					messageID = 0
				}

				err = reminder.UpdateState(ctx, h.db, chat, messageID, now)
				if err != nil {
					log.Error("update reminder state",
						log.Err(err),
						log.BotID(chat.BotID),
						log.ChatID(chat.ChatID),
						log.String("reminder_type", reminder.Name()),
					)
				}
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

	switch duration {
	case 0:
		message = fmt.Sprintf(text.PrayerArrived, prayer)
	default:
		message = fmt.Sprintf(text.PrayerSoon, prayer, domain.FormatDuration(duration))
	}

	params := &bot.SendMessageParams{
		ChatID:    chat.ChatID,
		Text:      message,
		ParseMode: models.ParseModeMarkdown,
	}

	res, err := b.SendMessage(ctx, params)

	if err != nil {
		if h.isBlockedErr(err) {
			h.deleteChat(ctx, chat)
			return nil
		}
		log.Error("remindUser: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	err = h.db.SetReminderMessageID(ctx, chat.BotID, chat.ChatID, int32(res.ID))
	if err != nil {
		log.Error("remindUser: set remind_message_id", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	h.deleteMessages(ctx, b, chat, chat.ReminderMessageID, chat.JamaatMessageID)
	return nil
}

func (h *Handler) remindUserJamaat(
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
		hasArrived      = false
	)

	switch duration {
	case 0:
		hasArrived = true
		message = fmt.Sprintf(text.PrayerArrived, prayer)
	default:
		message = fmt.Sprintf("%s\n%s",
			fmt.Sprintf(text.PrayerSoon, prayer, domain.FormatDuration(duration)),
			fmt.Sprintf(text.PrayerJamaat, domain.FormatDuration(duration+jamaatDelay)),
		)
	}

	var (
		res *models.Message
		err error
	)

	if hasArrived {
		res, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chat.ChatID,
			Text:      message,
			ParseMode: models.ParseModeMarkdown,
			ReplyParameters: &models.ReplyParameters{
				MessageID:                int(chat.JamaatMessageID),
				ChatID:                   chat.ChatID,
				AllowSendingWithoutReply: true,
			},
		})
	} else {
		isAnonymous := false
		message = strings.ReplaceAll(message, "*", "") // remove markdown syntax cuz doesn't work on poll
		res, err = b.SendPoll(ctx, &bot.SendPollParams{
			ChatID:   chat.ChatID,
			Question: message,
			Options: []models.InputPollOption{
				{
					Text:          text.PrayerJoin,
					TextParseMode: models.ParseModeMarkdown,
				},
				{
					Text:          text.PrayerJoinDelay,
					TextParseMode: models.ParseModeMarkdown,
				},
			},
			IsAnonymous: &isAnonymous,
		})
	}

	if err != nil {
		if h.isBlockedErr(err) {
			h.deleteChat(ctx, chat)
			return nil
		}
		log.Error("remindUserJamaat: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	if hasArrived {
		err = h.db.SetReminderMessageID(ctx, chat.BotID, chat.ChatID, int32(res.ID))
		if err != nil {
			log.Error("remindUserJamaat: set remind_message_id", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
			return domain.ErrInternal
		}
		h.deleteMessages(ctx, b, chat, chat.ReminderMessageID) // prevent duplicates
		return nil                                             // no need to continue further
	}

	err = h.db.SetJamaatMessageID(ctx, chat.BotID, chat.ChatID, int32(res.ID))
	if err != nil {
		log.Error("remindUserJamaat: set jamaat_message_id", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	h.deleteMessages(ctx, b, chat, chat.ReminderMessageID, chat.JamaatMessageID)
	return nil
}
// formatPrayerDay formats the domain.PrayerDay into a string (copied from dispatcher service)

func (h *Handler) deleteChat(ctx context.Context, chat *domain.Chat) {
	err := h.db.DeleteChat(ctx, chat.BotID, chat.ChatID)
	if err != nil {
		log.Error("remindUserJamaat: delete chat", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return
	}
	log.Warn("remindUserJamaat: deleted chat", log.BotID(chat.BotID), log.ChatID(chat.ChatID))
}

func (h *Handler) deleteMessages(ctx context.Context, b *bot.Bot, chat *domain.Chat, ids ...int32) {
	messageIDs := make([]int, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		messageIDs = append(messageIDs, int(id))
	}

	if len(messageIDs) == 0 {
		return // nothing to do
	}

	_, err := b.DeleteMessages(ctx, &bot.DeleteMessagesParams{
		ChatID:     chat.ChatID,
		MessageIDs: messageIDs,
	})
	if err != nil {
		log.Error("delete messages", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID), "ids", ids)
	}
}

func (h *Handler) isBlockedErr(err error) bool {
	return strings.HasPrefix(err.Error(), bot.ErrorForbidden.Error())
}

func (h *Handler) now(loc *time.Location) time.Time {
	now := time.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, loc)
}
