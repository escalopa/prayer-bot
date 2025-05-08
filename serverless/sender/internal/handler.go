package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/escalopa/prayer-bot/service"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"golang.org/x/sync/errgroup"
)

type (
	DB interface {
		CreateChat(ctx context.Context, botID int64, chatID int64, languageCode string, reminderOffset int32, state string) error
		GetChat(ctx context.Context, botID int64, chatID int64) (chat *domain.Chat, _ error)
		GetChatsByIDs(ctx context.Context, botID int64, chatIDs []int64) (chats []*domain.Chat, _ error)
		GetChats(ctx context.Context, botID int64) (chats []*domain.Chat, _ error)

		SetState(ctx context.Context, botID int64, chatID int64, state string) error
		SetSubscribed(ctx context.Context, botID int64, chatID int64, subscribed bool) error
		SetLanguageCode(ctx context.Context, botID int64, chatID int64, languageCode string) error
		SetReminderOffset(ctx context.Context, botID int64, chatID int64, reminderOffset int32) error
		SetReminderMessageID(ctx context.Context, botID int64, chatID int64, reminderMessageID int32) error

		GetPrayerDay(ctx context.Context, botID int64, date time.Time) (*domain.PrayerDay, error)

		GetStats(ctx context.Context, botID int64) (*domain.Stats, error)
	}

	Handler struct {
		cfg map[int64]*domain.BotConfig
		lp  *languagesProvider
		db  DB

		bots   map[int64]*bot.Bot
		botsMu sync.Mutex
	}
)

func NewHandler(cfg map[int64]*domain.BotConfig, db DB) (*Handler, error) {
	lp, err := newLanguageProvider()
	if err != nil {
		return nil, fmt.Errorf("create language provider: %v", err)
	}

	return &Handler{
		cfg:  cfg,
		lp:   lp,
		db:   db,
		bots: make(map[int64]*bot.Bot),
	}, nil
}

func (h *Handler) opts() []bot.Option {
	return []bot.Option{
		bot.WithDefaultHandler(h.errorH(h.defaultHandler)),

		bot.WithMessageTextHandler(startCommand.String(), bot.MatchTypeCommandStartOnly, h.errorH(h.start)),
		bot.WithMessageTextHandler(helpCommand.String(), bot.MatchTypeCommand, h.errorH(h.help)),
		bot.WithMessageTextHandler(todayCommand.String(), bot.MatchTypeCommand, h.errorH(h.today)),
		bot.WithMessageTextHandler(dateCommand.String(), bot.MatchTypeCommand, h.errorH(h.date)),
		bot.WithMessageTextHandler(nextCommand.String(), bot.MatchTypeCommand, h.errorH(h.next)),
		bot.WithMessageTextHandler(remindCommand.String(), bot.MatchTypeCommand, h.errorH(h.remind)),
		bot.WithMessageTextHandler(bugCommand.String(), bot.MatchTypeCommand, h.errorH(h.bug)),
		bot.WithMessageTextHandler(feedbackCommand.String(), bot.MatchTypeCommand, h.errorH(h.feedback)),
		bot.WithMessageTextHandler(languageCommand.String(), bot.MatchTypeCommand, h.errorH(h.language)),
		bot.WithMessageTextHandler(subscribeCommand.String(), bot.MatchTypeCommand, h.errorH(h.subscribe)),
		bot.WithMessageTextHandler(unsubscribeCommand.String(), bot.MatchTypeCommand, h.errorH(h.unsubscribe)),
		bot.WithMessageTextHandler(cancelCommand.String(), bot.MatchTypeCommand, h.errorH(h.cancel)),

		bot.WithMessageTextHandler(adminCommand.String(), bot.MatchTypeCommand, h.errorH(h.authorize(h.admin))),
		bot.WithMessageTextHandler(replyCommand.String(), bot.MatchTypeCommand, h.errorH(h.authorize(h.reply))),
		bot.WithMessageTextHandler(statsCommand.String(), bot.MatchTypeCommand, h.errorH(h.authorize(h.stats))),
		bot.WithMessageTextHandler(announceCommand.String(), bot.MatchTypeCommand, h.errorH(h.authorize(h.announce))),

		bot.WithCallbackQueryDataHandler(monthQuery.String(), bot.MatchTypePrefix, h.errorH(h.monthQuery)),
		bot.WithCallbackQueryDataHandler(dayQuery.String(), bot.MatchTypePrefix, h.errorH(h.dayQuery)),
		bot.WithCallbackQueryDataHandler(remindQuery.String(), bot.MatchTypePrefix, h.errorH(h.remindQuery)),
		bot.WithCallbackQueryDataHandler(languageQuery.String(), bot.MatchTypePrefix, h.errorH(h.languageQuery)),
		bot.WithCallbackQueryDataHandler(emptyQuery.String(), bot.MatchTypePrefix, h.errorH(h.emptyQuery)),
	}
}

func (h *Handler) errorH(fn func(ctx context.Context, b *bot.Bot, update *models.Update) error) func(ctx context.Context, b *bot.Bot, update *models.Update) {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("panic recovered: %v", r)
			}
		}()

		err := fn(ctx, b, update)
		if err != nil {
			fmt.Printf("error: %v\n", err)
		}
	}
}

func (h *Handler) authorize(fn func(ctx context.Context, b *bot.Bot, update *models.Update) error) func(ctx context.Context, b *bot.Bot, update *models.Update) error {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) error {
		chat, err := h.getChat(ctx, update)
		if err != nil {
			return fmt.Errorf("authorize: get chat: %v", err)
		}

		if h.cfg[chat.BotID].OwnerID == update.Message.From.ID { // isAdmin
			return fn(ctx, b, update)
		}

		return h.help(ctx, b, update)
	}
}

func (h *Handler) defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("defaultHandler: get chat: %v", err)
	}

	switch chat.State {
	case string(bugState):
		err = h.bugState(ctx, b, update)
	case string(feedbackState):
		err = h.feedbackState(ctx, b, update)
	case string(replyState):
		err = h.replyState(ctx, b, update)
	case string(announceState):
		err = h.announceState(ctx, b, update)
	default:
		return h.help(ctx, b, update)
	}

	if err != nil {
		return fmt.Errorf("defaultHandler: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) Do(ctx context.Context, body string) error {
	payload := &domain.Payload{}
	if err := payload.Unmarshal([]byte(body)); err != nil {
		return fmt.Errorf("unmarshal payload: %v", err)
	}

	switch payload.Type {
	case domain.PayloadTypeDispatcher:
		return h.processDispatcher(ctx, payload.Data)
	case domain.PayloadTypeReminder:
		return h.processReminder(ctx, payload.Data)
	default:
		fmt.Printf("unknown payload type: %s", payload.Type) // ignore message
		return nil
	}
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

	b, err := bot.New(botConfig.Token, h.opts()...)
	if err != nil {
		return nil, fmt.Errorf("create bot: %v", err)
	}

	h.bots[botID] = b
	return b, nil
}

func (h *Handler) processDispatcher(ctx context.Context, data interface{}) error {
	payload, err := unmarshalPayload[domain.DispatcherPayload](data)
	if err != nil {
		return err
	}

	b, err := h.getBot(payload.BotID)
	if err != nil {
		return fmt.Errorf("get bot: %v", err)
	}

	var update models.Update
	err = json.Unmarshal([]byte(payload.Data), &update)
	if err != nil {
		return fmt.Errorf("unmarshal update: %v", err)
	}

	ctx = setContextBotID(ctx, payload.BotID)
	b.ProcessUpdate(ctx, &update)
	return nil
}

func (h *Handler) processReminder(ctx context.Context, data interface{}) error {
	payload, err := unmarshalPayload[domain.ReminderPayload](data)
	if err != nil {
		return err
	}

	b, err := h.getBot(payload.BotID)
	if err != nil {
		return fmt.Errorf("get bot: %v", err)
	}

	chats, err := h.db.GetChatsByIDs(ctx, payload.BotID, payload.ChatIDs)
	if err != nil {
		return fmt.Errorf("get chats: %v", err)
	}

	errG := &errgroup.Group{}
	for _, chat := range chats {
		chat := chat
		errG.Go(func() error {
			err := h.remindUser(ctx, b, chat, payload.PrayerID, payload.ReminderOffset)
			if err != nil {
				fmt.Printf("processReminder: send message: %v", err)
			}
			return nil
		})
	}

	_ = errG.Wait()
	return nil
}

func (h *Handler) getChat(ctx context.Context, update *models.Update) (*domain.Chat, error) {
	botID := getContextBotID(ctx)
	chatID := int64(0)

	switch {
	case update.Message != nil:
		chatID = update.Message.Chat.ID
	case update.CallbackQuery != nil:
		chatID = update.CallbackQuery.Message.Message.Chat.ID
	default:
		return nil, fmt.Errorf("cannot extract chat_id: bot_id: %d update: %+v", botID, update)
	}

	chat, err := h.db.GetChat(ctx, botID, chatID)
	if err != nil && !errors.Is(err, service.ErrNotFound) {
		fmt.Printf("get chat: bot_id: %d chat_id: %d err: %v", botID, chatID, err)
		return nil, err
	}

	if chat != nil { // chat found
		return chat, nil
	}

	// chat not found then create it

	languageCode := defaultLanguageCode
	if h.lp.IsSupportedCode(update.Message.From.LanguageCode) {
		languageCode = update.Message.From.LanguageCode
	}

	err = h.db.CreateChat(ctx, botID, chatID, languageCode, int32(0), string(defaultState))
	if err != nil {
		fmt.Printf("create chat: bot_id: %d chat_id: %d err: %v", botID, chatID, err)
		return nil, fmt.Errorf("create chat: %v", err)
	}

	chat = &domain.Chat{
		BotID:             botID,
		ChatID:            chatID,
		State:             string(defaultState),
		LanguageCode:      languageCode,
		ReminderMessageID: 0, // no message to delete
	}

	return chat, nil
}
