package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
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
		CreateChat(ctx context.Context, botID int32, chatID int64, languageCode string, notifyOffset int32, state string) error
		GetChat(ctx context.Context, botID int32, chatID int64) (chat *domain.Chat, _ error)
		GetChats(ctx context.Context, botID int32, chatIDs []int64) (chats []*domain.Chat, _ error)
		GetAllChats(ctx context.Context, botID int32) (chats []*domain.Chat, _ error)

		SetState(ctx context.Context, botID int32, chatID int64, state string) error
		SetSubscribed(ctx context.Context, botID int32, chatID int64, subscribed bool) error
		SetLanguageCode(ctx context.Context, botID int32, chatID int64, languageCode string) error
		SetNotifyOffset(ctx context.Context, botID int32, chatID int64, notifyOffset int32) error
		SetNotifyMessageID(ctx context.Context, botID int32, chatID int64, notifyMessageID int32) error

		GetPrayerDay(ctx context.Context, botID int32, date time.Time) (*domain.PrayerDay, error)

		GetStats(ctx context.Context, botID int32) (*domain.Stats, error)
	}

	Handler struct {
		cfg map[int32]*domain.BotConfig
		lp  *languagesProvider
		db  DB

		bots   map[int32]*bot.Bot
		botsMu sync.Mutex
	}
)

func NewHandler(cfg map[int32]*domain.BotConfig, db DB) (*Handler, error) {
	lp, err := newLanguageProvider()
	if err != nil {
		return nil, fmt.Errorf("create language provider: %v", err)
	}

	return &Handler{
		cfg:  cfg,
		lp:   lp,
		db:   db,
		bots: make(map[int32]*bot.Bot),
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
		bot.WithMessageTextHandler(notifyCommand.String(), bot.MatchTypeCommand, h.errorH(h.notify)),
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

		bot.WithCallbackQueryDataHandler(callbackDateMonth.String(), bot.MatchTypePrefix, h.errorH(h.callbackDateMonth)),
		bot.WithCallbackQueryDataHandler(callbackDateDay.String(), bot.MatchTypePrefix, h.errorH(h.callbackDateDay)),
		bot.WithCallbackQueryDataHandler(callbackNotify.String(), bot.MatchTypePrefix, h.errorH(h.callbackNotify)),
		bot.WithCallbackQueryDataHandler(callbackLanguage.String(), bot.MatchTypePrefix, h.errorH(h.callbackLanguage)),
		bot.WithCallbackQueryDataHandler(callbackEmpty.String(), bot.MatchTypePrefix, h.errorH(h.emptyCallback)),
	}
}

func (h *Handler) emptyCallback(_ context.Context, _ *bot.Bot, _ *models.Update) error {
	return nil
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
	case string(chatStateBug):
		err = h.chatStateBug(ctx, b, update)
	case string(chatStateFeedback):
		err = h.chatStateFeedback(ctx, b, update)
	case string(chatStateReply):
		err = h.chatStateReply(ctx, b, update)
	case string(chatStateAnnounce):
		err = h.chatStateAnnounce(ctx, b, update)
	default:
		return h.help(ctx, b, update)
	}

	if err != nil {
		return fmt.Errorf("defaultHandler: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) chatStateBug(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("chatStateBug: get chat: %v", err)
	}

	_, err = b.ForwardMessage(ctx, &bot.ForwardMessageParams{
		ChatID:     h.cfg[chat.BotID].OwnerID,
		FromChatID: chat.ChatID,
		MessageID:  update.Message.ID,
	})
	if err != nil {
		return fmt.Errorf("chatStateBug: forward message: %v", err)
	}

	info := newReplyInfo(replyTypeBug, chat.ChatID, update.Message.ID, update.Message.From.Username)
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: h.cfg[chat.BotID].OwnerID,
		Text:   info.JSON(),
	})
	if err != nil {
		return fmt.Errorf("chatStateBug: send message: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Bug.Success,
	})
	if err != nil {
		return fmt.Errorf("chatStateBug: send message: %v", err)
	}

	return nil
}

func (h *Handler) chatStateFeedback(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("chatStateFeedback: get chat: %v", err)
	}

	_, err = b.ForwardMessage(ctx, &bot.ForwardMessageParams{
		ChatID:     h.cfg[chat.BotID].OwnerID,
		FromChatID: chat.ChatID,
		MessageID:  update.Message.ID,
	})
	if err != nil {
		return fmt.Errorf("chatStateFeedback: forward message: %v", err)
	}

	info := newReplyInfo(replyTypeFeedback, chat.ChatID, update.Message.ID, update.Message.From.Username)
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: h.cfg[chat.BotID].OwnerID,
		Text:   info.JSON(),
	})
	if err != nil {
		return fmt.Errorf("chatStateFeedback: send message: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Feedback.Success,
	})
	if err != nil {
		return fmt.Errorf("chatStateFeedback: send message: %v", err)
	}

	return nil
}

func (h *Handler) chatStateReply(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("chatStateReply: get chat: %v", err)
	}

	if update.Message.ReplyToMessage == nil {
		return fmt.Errorf("chatStateReply: reply to message is nil")
	}

	info := &replyInfo{}
	err = json.Unmarshal([]byte(update.Message.ReplyToMessage.Text), info)
	if err != nil {
		return fmt.Errorf("chatStateReply: unmarshal reply info: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: info.ChatID,
		Text:   update.Message.Text,
		ReplyParameters: &models.ReplyParameters{
			MessageID:                info.MessageID,
			AllowSendingWithoutReply: true,
		},
		Entities: update.Message.Entities,
	})
	if err != nil {
		return fmt.Errorf("chatStateReply: send message: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Reply.Success,
	})
	if err != nil {
		return fmt.Errorf("chatStateReply: send message: %v", err)
	}

	return nil
}

func (h *Handler) chatStateAnnounce(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("chatStateAnnounce: get chat: %v", err)
	}

	chats, err := h.db.GetAllChats(ctx, chat.BotID)
	if err != nil {
		return fmt.Errorf("chatStateAnnounce: get all chats: %v", err)
	}

	g := &errgroup.Group{}
	for _, c := range chats {
		c := c
		g.Go(func() error {
			_, err = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: c.ChatID,
				Text:   update.Message.Text,
			})
			return err
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("chatStateAnnounce: send message: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Announce.Success,
	})
	if err != nil {
		return fmt.Errorf("chatStateAnnounce: send message: %v", err)
	}

	return nil
}

func (h *Handler) getBot(botID int32) (*bot.Bot, error) {
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

func (h *Handler) Do(ctx context.Context, body string) error {
	payload := &domain.Payload{}
	if err := payload.Unmarshal([]byte(body)); err != nil {
		return fmt.Errorf("unmarshal payload: %v", err)
	}

	switch payload.Type {
	case domain.PayloadTypeHandler:
		return h.processHandel(ctx, payload.Data)
	case domain.PayloadTypeNotifier:
		return h.processNotify(ctx, payload.Data)
	default:
		fmt.Printf("unknown payload type: %s", payload.Type) // ignore message
		return nil
	}
}

func (h *Handler) processHandel(ctx context.Context, data interface{}) error {
	payload, err := domain.Unmarshal[domain.HandlerPayload](data)
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

func (h *Handler) start(ctx context.Context, b *bot.Bot, update *models.Update) error {
	return h.help(ctx, b, update)
}

func (h *Handler) help(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("help: get chat: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chat.ChatID,
		Text:      h.lp.GetText(chat.LanguageCode).Help,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		return fmt.Errorf("help: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) today(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("today: get chat: %v", err)
	}

	now := h.now(chat.BotID)
	prayerDay, err := h.db.GetPrayerDay(ctx, chat.BotID, now)
	if err != nil {
		return fmt.Errorf("today: get prayer day: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.formatPrayerDay(prayerDay, chat.LanguageCode),
	})
	if err != nil {
		return fmt.Errorf("today: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) date(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("date: get chat: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chat.ChatID,
		Text:        h.lp.GetText(chat.LanguageCode).PrayerDate,
		ReplyMarkup: h.monthsKeyboard(chat.LanguageCode),
	})
	if err != nil {
		return fmt.Errorf("date: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) next(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("next: get chat: %v", err)
	}

	date := h.now(chat.BotID)
	prayerDay, err := h.db.GetPrayerDay(ctx, chat.BotID, date)
	if err != nil {
		return fmt.Errorf("next: get prayer day: %v", err)
	}

	prayerID, duration := domain.PrayerIDUnknown, time.Duration(0)
	switch {
	case prayerDay.Fajr.After(date):
		prayerID, duration = domain.PrayerIDFajr, prayerDay.Fajr.Sub(date)
	case prayerDay.Shuruq.After(date):
		prayerID, duration = domain.PrayerIDShuruq, prayerDay.Shuruq.Sub(date)
	case prayerDay.Dhuhr.After(date):
		prayerID, duration = domain.PrayerIDDhuhr, prayerDay.Dhuhr.Sub(date)
	case prayerDay.Asr.After(date):
		prayerID, duration = domain.PrayerIDAsr, prayerDay.Asr.Sub(date)
	case prayerDay.Maghrib.After(date):
		prayerID, duration = domain.PrayerIDMaghrib, prayerDay.Maghrib.Sub(date)
	case prayerDay.Isha.After(date):
		prayerID, duration = domain.PrayerIDIsha, prayerDay.Isha.Sub(date)
	}

	// when no prayer time is found, return the first prayer of the next day
	if prayerID == domain.PrayerIDUnknown || duration == 0 {
		nextDate := domain.Date(date.Day()+1, date.Month(), date.Year(), date.Location())
		prayerDay, err = h.db.GetPrayerDay(ctx, chat.BotID, nextDate)
		if err != nil {
			return fmt.Errorf("next: get prayer day: %v", err)
		}
		prayerID, duration = domain.PrayerIDFajr, prayerDay.Fajr.Sub(date)
	}

	text := h.lp.GetText(chat.LanguageCode)
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chat.ChatID,
		Text:      fmt.Sprintf(text.PrayerSoon, text.Prayer[int(prayerID)], formatDuration(duration)),
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		return fmt.Errorf("next: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) notify(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("notify: get chat: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chat.ChatID,
		Text:        h.lp.GetText(chat.LanguageCode).NotifyOffset.Start,
		ReplyMarkup: h.notifyKeyboard(),
	})
	if err != nil {
		return fmt.Errorf("notify: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) bug(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("bug: get chat: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Bug.Start,
	})
	if err != nil {
		return fmt.Errorf("bug: send message: %v", err)
	}

	err = h.db.SetState(ctx, chat.BotID, chat.ChatID, chatStateBug.String())
	if err != nil {
		return fmt.Errorf("bug: set state: %v", err)
	}

	return nil
}

func (h *Handler) feedback(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("feedback: get chat: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Feedback.Start,
	})
	if err != nil {
		return fmt.Errorf("feedback: send message: %v", err)
	}

	err = h.db.SetState(ctx, chat.BotID, chat.ChatID, chatStateFeedback.String())
	if err != nil {
		return fmt.Errorf("feedback: set state: %v", err)
	}

	return nil
}

func (h *Handler) language(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("language: get chat: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chat.ChatID,
		Text:        h.lp.GetText(chat.LanguageCode).Language.Start,
		ReplyMarkup: h.languagesKeyboard(),
	})
	if err != nil {
		return fmt.Errorf("language: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) subscribe(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("subscribe: get chat: %v", err)
	}

	err = h.db.SetSubscribed(ctx, chat.BotID, chat.ChatID, true)
	if err != nil {
		return fmt.Errorf("subscribe: set subscribed: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).SubscriptionSuccess,
	})
	if err != nil {
		fmt.Printf("subscribe: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) unsubscribe(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("unsubscribe: get chat: %v", err)
	}

	err = h.db.SetSubscribed(ctx, chat.BotID, chat.ChatID, false)
	if err != nil {
		return fmt.Errorf("unsubscribe: set subscribed: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).UnsubscriptionSuccess,
	})
	if err != nil {
		return fmt.Errorf("unsubscribe: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) admin(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("admin: get chat: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chat.ChatID,
		Text:      h.lp.GetText(chat.LanguageCode).HelpAdmin,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		return fmt.Errorf("admin: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) reply(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("reply: get chat: %v", err)
	}

	err = h.db.SetState(ctx, chat.BotID, chat.ChatID, string(chatStateReply))
	if err != nil {
		return fmt.Errorf("reply: set state: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Reply.Start,
	})
	if err != nil {
		return fmt.Errorf("reply: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) stats(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("stats: get chat: %v", err)
	}

	stats, err := h.db.GetStats(ctx, chat.BotID)
	if err != nil {
		return fmt.Errorf("stats: get stats: %v", err)
	}

	languagesStats := &strings.Builder{}
	for _, lang := range h.lp.GetLanguages() {
		row := fmt.Sprintf("%s: %d\n", lang.Code, stats.LanguagesGrouped[lang.Code])
		languagesStats.WriteString(row)
	}

	message := fmt.Sprintf(h.lp.GetText(chat.LanguageCode).Stats,
		stats.Users,
		stats.Subscribed,
		stats.Unsubscribed,
		languagesStats.String(),
	)
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   message,
	})
	if err != nil {
		return fmt.Errorf("stats: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) announce(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("announce: get chat: %v", err)
	}

	err = h.db.SetState(ctx, chat.BotID, chat.ChatID, string(chatStateAnnounce))
	if err != nil {
		return fmt.Errorf("announce: set state: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Announce.Start,
	})
	if err != nil {
		return fmt.Errorf("announce: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) cancel(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("cancel: get chat: %v", err)
	}

	if chat.State == string(chatStateDefault) {
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chat.ChatID,
			Text:   h.lp.GetText(chat.LanguageCode).Noop,
		})
		if err != nil {
			return fmt.Errorf("cancel: send message: %v", err)
		}
		return nil
	}

	err = h.db.SetState(ctx, chat.BotID, chat.ChatID, string(chatStateDefault))
	if err != nil {
		return fmt.Errorf("cancel: set state: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Cancel,
	})
	if err != nil {
		return fmt.Errorf("cancel: send message: %v", err)
	}

	return nil
}

func (h *Handler) resetState(ctx context.Context, chat *domain.Chat) {
	if chat.State == string(chatStateDefault) {
		return
	}

	err := h.db.SetState(ctx, chat.BotID, chat.ChatID, string(chatStateDefault))
	if err != nil {
		fmt.Printf("reset state: bot_id: %d chat_id: %d err: %v", chat.BotID, chat.ChatID, err)
	}
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

	err = h.db.CreateChat(ctx, botID, chatID, languageCode, int32(domain.NotifyOffset0m), string(chatStateDefault))
	if err != nil {
		fmt.Printf("create chat: bot_id: %d chat_id: %d err: %v", botID, chatID, err)
		return nil, fmt.Errorf("create chat: %v", err)
	}

	chat = &domain.Chat{
		BotID:           botID,
		ChatID:          chatID,
		State:           string(chatStateDefault),
		LanguageCode:    languageCode,
		NotifyMessageID: 0, // no message to delete
	}

	return chat, nil
}

func (h *Handler) processNotify(ctx context.Context, data interface{}) error {
	payload, err := domain.Unmarshal[domain.NotifierPayload](data)
	if err != nil {
		return err
	}

	b, err := h.getBot(payload.BotID)
	if err != nil {
		return fmt.Errorf("get bot: %v", err)
	}

	chats, err := h.db.GetChats(ctx, payload.BotID, payload.ChatIDs)
	if err != nil {
		return fmt.Errorf("get chats: %v", err)
	}

	errG := &errgroup.Group{}
	for _, chat := range chats {
		chat := chat
		errG.Go(func() error {
			err := h.notifyBotUser(ctx, b, chat, payload.PrayerID, payload.NotifyOffset)
			if err != nil {
				fmt.Printf("processNotify: send message: %v", err)
			}
			return nil
		})
	}

	_ = errG.Wait()
	return nil
}

func (h *Handler) notifyBotUser(
	ctx context.Context,
	b *bot.Bot,
	chat *domain.Chat,
	prayerID domain.PrayerID,
	notifyOffset domain.NotifyOffset,
) error {
	var (
		text            = h.lp.GetText(chat.LanguageCode)
		duration        = time.Duration(notifyOffset) * time.Minute
		prayer, message = text.Prayer[int(prayerID)], ""
	)

	switch {
	case duration == 0:
		message = fmt.Sprintf(text.PrayerArrived, prayer)
	default:
		message = fmt.Sprintf(text.PrayerSoon, prayer, formatDuration(duration))
	}

	res, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chat.ChatID,
		Text:      message,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		return fmt.Errorf("send message: bot_id: %d chat_id: %d err: %v", chat.BotID, chat.ChatID, err)
	}

	if chat.NotifyMessageID == 0 { // no message to delete
		return nil
	}

	_, err = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chat.ChatID,
		MessageID: int(chat.NotifyMessageID),
	})
	if err != nil {
		return fmt.Errorf("delete message: bot_id: %d chat_id: %d err: %v", chat.BotID, chat.ChatID, err)
	}

	err = h.db.SetNotifyMessageID(ctx, chat.BotID, chat.ChatID, int32(res.ID))
	if err != nil {
		return fmt.Errorf("set notify message id: bot_id: %d chat_id: %d err: %v", chat.BotID, chat.ChatID, err)
	}

	return nil
}

func (h *Handler) callbackDateMonth(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("callbackDateMonth: get chat: %v", err)
	}

	month, _ := strconv.Atoi(strings.TrimPrefix(update.CallbackQuery.Data, callbackDateMonth.String()))

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chat.ChatID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		Text:        h.lp.GetText(chat.LanguageCode).PrayerDate,
		ReplyMarkup: h.daysKeyboard(h.now(chat.BotID), month),
	})
	if err != nil {
		return fmt.Errorf("callbackDateMonth: edit message: %v", err)
	}

	_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	if err != nil {
		return fmt.Errorf("callbackDateMonth: answer callback query: %v", err)
	}

	return nil
}

func (h *Handler) callbackDateDay(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("callbackDateDay: get chat: %v", err)
	}

	parts := strings.Split(update.CallbackQuery.Data, "|")
	month, _ := strconv.Atoi(parts[1])
	day, _ := strconv.Atoi(parts[2])

	date := h.now(chat.BotID)
	date = domain.Date(day, time.Month(month), date.Year(), time.UTC)

	prayerDay, err := h.db.GetPrayerDay(ctx, chat.BotID, date)
	if err != nil {
		return fmt.Errorf("callbackDateDay: get prayer day: %v", err)
	}

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chat.ChatID,
		MessageID: update.CallbackQuery.Message.Message.ID,
		Text:      h.formatPrayerDay(prayerDay, chat.LanguageCode),
	})
	if err != nil {
		return fmt.Errorf("callbackDateDay: edit message: %v", err)
	}

	_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	if err != nil {
		return fmt.Errorf("callbackDateDay: answer callback query: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) callbackNotify(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("callbackNotify: get chat: %v", err)
	}

	notifyOffset, _ := strconv.Atoi(strings.TrimPrefix(update.CallbackQuery.Data, callbackNotify.String()))

	err = h.db.SetNotifyOffset(ctx, chat.BotID, chat.ChatID, int32(notifyOffset))
	if err != nil {
		return fmt.Errorf("callbackNotify: set notify offset: %v", err)
	}

	message := fmt.Sprintf(h.lp.GetText(chat.LanguageCode).NotifyOffset.Success, notifyOffset)
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chat.ChatID,
		MessageID: update.CallbackQuery.Message.Message.ID,
		Text:      message,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		return fmt.Errorf("callbackNotify: edit message: %v", err)
	}

	_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	if err != nil {
		return fmt.Errorf("callbackNotify: answer callback query: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) callbackLanguage(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		return fmt.Errorf("callbackLanguage: get chat: %v", err)
	}

	languageCode := strings.TrimPrefix(update.CallbackQuery.Data, callbackLanguage.String())
	if !h.lp.IsSupportedCode(languageCode) {
		return fmt.Errorf("callbackLanguage: unsupported language code: %s", languageCode)
	}

	err = h.db.SetLanguageCode(ctx, chat.BotID, chat.ChatID, languageCode)
	if err != nil {
		return fmt.Errorf("callbackLanguage: set language code: %v", err)
	}

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chat.ChatID,
		MessageID: update.CallbackQuery.Message.Message.ID,
		Text:      fmt.Sprintf(h.lp.GetText(languageCode).Language.Success, languageCode),
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		return fmt.Errorf("callbackLanguage: send message: %v", err)
	}

	_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	if err != nil {
		return fmt.Errorf("callbackLanguage: answer callback query: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}
