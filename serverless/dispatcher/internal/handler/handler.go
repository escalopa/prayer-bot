package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/escalopa/prayer-bot/log"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type (
	DB interface {
		CreateChat(ctx context.Context, botID int64, chatID int64, languageCode string, reminderOffset int32, state string, jamaat bool) error
		GetChat(ctx context.Context, botID int64, chatID int64) (chat *domain.Chat, _ error)
		GetChats(ctx context.Context, botID int64) (chats []*domain.Chat, _ error)
		SetState(ctx context.Context, botID int64, chatID int64, state string) error

		GetStats(ctx context.Context, botID int64) (*domain.Stats, error)
		GetPrayerDay(ctx context.Context, botID int64, date time.Time) (*domain.PrayerDay, error)

		SetSubscribed(ctx context.Context, botID int64, chatID int64, subscribed bool) error
		SetLanguageCode(ctx context.Context, botID int64, chatID int64, languageCode string) error
		SetReminderOffset(ctx context.Context, botID int64, chatID int64, reminderOffset int32) error
	}

	Handler struct {
		cfg map[int64]*domain.BotConfig
		lp  *languagesProvider
		db  DB

		bots   map[int64]*bot.Bot
		botsMu sync.Mutex
	}
)

func New(cfg map[int64]*domain.BotConfig, db DB) (*Handler, error) {
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
		bot.WithDefaultHandler(h.errorH(h.chatH(h.defaultHandler))),

		bot.WithMessageTextHandler(startCommand.String(), bot.MatchTypeCommandStartOnly, h.errorH(h.chatH(h.start))),
		bot.WithMessageTextHandler(helpCommand.String(), bot.MatchTypeCommand, h.errorH(h.chatH(h.help))),
		bot.WithMessageTextHandler(todayCommand.String(), bot.MatchTypeCommand, h.errorH(h.chatH(h.today))),
		bot.WithMessageTextHandler(dateCommand.String(), bot.MatchTypeCommand, h.errorH(h.chatH(h.date))),
		bot.WithMessageTextHandler(nextCommand.String(), bot.MatchTypeCommand, h.errorH(h.chatH(h.next))),
		bot.WithMessageTextHandler(remindCommand.String(), bot.MatchTypeCommand, h.errorH(h.chatH(h.remind))),
		bot.WithMessageTextHandler(bugCommand.String(), bot.MatchTypeCommand, h.errorH(h.chatH(h.bug))),
		bot.WithMessageTextHandler(feedbackCommand.String(), bot.MatchTypeCommand, h.errorH(h.chatH(h.feedback))),
		bot.WithMessageTextHandler(languageCommand.String(), bot.MatchTypeCommand, h.errorH(h.chatH(h.language))),
		bot.WithMessageTextHandler(subscribeCommand.String(), bot.MatchTypeCommand, h.errorH(h.chatH(h.subscribe))),
		bot.WithMessageTextHandler(unsubscribeCommand.String(), bot.MatchTypeCommand, h.errorH(h.chatH(h.unsubscribe))),
		bot.WithMessageTextHandler(cancelCommand.String(), bot.MatchTypeCommand, h.errorH(h.chatH(h.cancel))),

		bot.WithMessageTextHandler(adminCommand.String(), bot.MatchTypeCommand, h.errorH(h.chatH(h.authorizeH(h.admin)))),
		bot.WithMessageTextHandler(replyCommand.String(), bot.MatchTypeCommand, h.errorH(h.chatH(h.authorizeH(h.reply)))),
		bot.WithMessageTextHandler(statsCommand.String(), bot.MatchTypeCommand, h.errorH(h.chatH(h.authorizeH(h.stats)))),
		bot.WithMessageTextHandler(announceCommand.String(), bot.MatchTypeCommand, h.errorH(h.chatH(h.authorizeH(h.announce)))),

		bot.WithCallbackQueryDataHandler(monthQuery.String(), bot.MatchTypePrefix, h.errorH(h.chatH(h.monthQuery))),
		bot.WithCallbackQueryDataHandler(dayQuery.String(), bot.MatchTypePrefix, h.errorH(h.chatH(h.dayQuery))),
		bot.WithCallbackQueryDataHandler(remindQuery.String(), bot.MatchTypePrefix, h.errorH(h.chatH(h.remindQuery))),
		bot.WithCallbackQueryDataHandler(languageQuery.String(), bot.MatchTypePrefix, h.errorH(h.chatH(h.languageQuery))),
		bot.WithCallbackQueryDataHandler(emptyQuery.String(), bot.MatchTypePrefix, h.errorH(h.chatH(h.emptyQuery))),
	}
}

func (h *Handler) errorH(fn func(ctx context.Context, b *bot.Bot, update *models.Update) error) func(ctx context.Context, b *bot.Bot, update *models.Update) {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("recovered from panic",
					log.String("stack", string(debug.Stack())),
					log.String("err", fmt.Sprintf("%+v", r)),
				)
			}
		}()

		err := fn(ctx, b, update)
		if err != nil {
			log.Error("handler error", log.Err(err))
		}
	}
}

func (h *Handler) chatH(fn func(ctx context.Context, b *bot.Bot, update *models.Update) error) func(ctx context.Context, b *bot.Bot, update *models.Update) error {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) error {
		chat, err := h.getChat(ctx, update)
		if err != nil {
			if errors.Is(err, domain.ErrInvalidArgument) {
				return nil
			}
			log.Error("chatH: get chat", log.Err(err))
		}
		ctx = setContextChat(ctx, chat)
		return fn(ctx, b, update)
	}
}

func (h *Handler) authorizeH(fn func(ctx context.Context, b *bot.Bot, update *models.Update) error) func(ctx context.Context, b *bot.Bot, update *models.Update) error {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) error {
		chat := getContextChat(ctx)

		var (
			isAdminUser = h.cfg[chat.BotID].OwnerID == update.Message.From.ID
			isAdminChat = h.cfg[chat.BotID].OwnerID == update.Message.Chat.ID
		)

		if isAdminUser && isAdminChat {
			return fn(ctx, b, update)
		}

		return h.help(ctx, b, update)
	}
}

func (h *Handler) Authenticate(headers map[string]string) (int64, error) {
	const telegramBotAPISecretTokenHeader = "X-Telegram-Bot-Api-Secret-Token"

	secretToken := headers[telegramBotAPISecretTokenHeader]
	if secretToken == "" {
		return 0, fmt.Errorf("empty secret token header")
	}

	for _, botConfig := range h.cfg {
		if botConfig.Secret == secretToken {
			return botConfig.BotID, nil
		}
	}

	return 0, fmt.Errorf("secret token mismatch")
}

func (h *Handler) Handel(ctx context.Context, botID int64, data string) error {
	b, err := h.getBot(botID)
	if err != nil {
		log.Error("get bot", log.Err(err), log.BotID(botID), log.String("payload", data))
		return err
	}

	var update models.Update
	err = json.Unmarshal([]byte(data), &update)
	if err != nil {
		log.Error("unmarshal update", log.Err(err), log.String("payload", data))
		return nil
	}

	ctx = setContextBotID(ctx, botID)
	b.ProcessUpdate(ctx, &update)
	return nil
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
		log.Error("bot not found", log.BotID(botID))
		return nil, domain.ErrNotFound
	}

	b, err := bot.New(botConfig.Token, h.opts()...)
	if err != nil {
		log.Error("create bot", log.BotID(botID), log.Err(err))
		return nil, domain.ErrInternal
	}

	h.bots[botID] = b
	return b, nil
}

func (h *Handler) getChat(ctx context.Context, update *models.Update) (*domain.Chat, error) {
	botID := getContextBotID(ctx)
	chatID := int64(0)
	jamaat := false

	switch {
	case update.Message != nil:
		chatID = update.Message.Chat.ID
		jamaat = isJamaat(update.Message.Chat)
	case update.CallbackQuery != nil:
		chatID = update.CallbackQuery.Message.Message.Chat.ID
		jamaat = isJamaat(update.CallbackQuery.Message.Message.Chat)
	default:
		bytes, _ := json.Marshal(update)
		log.Info("ignore message", log.BotID(botID), log.String("update", string(bytes)))
		return nil, domain.ErrInvalidArgument // ignore message
	}

	chat, err := h.db.GetChat(ctx, botID, chatID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		log.Error("get chat", log.Err(err), log.BotID(botID), log.ChatID(chatID), "update", update)
		return nil, domain.ErrInternal
	}

	if chat != nil { // chat found
		return chat, nil
	}

	// chat not found then create it

	languageCode := defaultLanguageCode
	if h.lp.IsSupportedCode(update.Message.From.LanguageCode) {
		languageCode = update.Message.From.LanguageCode
	}

	err = h.db.CreateChat(ctx, botID, chatID, languageCode, int32(0), string(defaultState), jamaat)
	if err != nil {
		log.Error("create chat", log.Err(err), log.BotID(botID), log.ChatID(chatID), "update", update)
		return nil, domain.ErrInternal
	}

	chat = &domain.Chat{
		BotID:             botID,
		ChatID:            chatID,
		State:             string(defaultState),
		LanguageCode:      languageCode,
		ReminderMessageID: 0,
		Jamaat:            jamaat,
		JamaatMessageID:   0,
	}

	return chat, nil
}
