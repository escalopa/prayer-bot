package telegram

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	botapi "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/escalopa/prayer-bot/global/internal/assets"
	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
	"github.com/escalopa/prayer-bot/global/internal/location"
	"github.com/escalopa/prayer-bot/global/internal/prayertime"
	"github.com/escalopa/prayer-bot/global/internal/reminders"
	"github.com/escalopa/prayer-bot/global/internal/store"
)

type Bot interface {
	SendMessage(context.Context, *botapi.SendMessageParams) (*models.Message, error)
	SendPhoto(context.Context, *botapi.SendPhotoParams) (*models.Message, error)
	EditMessageText(context.Context, *botapi.EditMessageTextParams) (*models.Message, error)
	AnswerCallbackQuery(context.Context, *botapi.AnswerCallbackQueryParams) (bool, error)
	GetChatMember(context.Context, *botapi.GetChatMemberParams) (*models.ChatMember, error)
}

type Handler struct {
	bot        Bot
	store      *store.Store
	resolver   location.Resolver
	calculator prayertime.Calculator
	planner    *reminders.Planner
	ownerID    int64
	now        func() time.Time
}

func NewHandler(bot Bot, storage *store.Store, resolver location.Resolver, calculator prayertime.Calculator, planner *reminders.Planner, ownerID int64) *Handler {
	return &Handler{bot: bot, store: storage, resolver: resolver, calculator: calculator, planner: planner, ownerID: ownerID, now: time.Now}
}

func (h *Handler) Handle(ctx context.Context, update models.Update) error {
	if update.CallbackQuery != nil {
		return h.handleCallback(ctx, update.CallbackQuery)
	}
	message := update.Message
	if message == nil || message.Chat.Type == models.ChatTypeChannel {
		return nil
	}
	languageHint := "en"
	if message.From != nil && message.From.LanguageCode != "" {
		languageHint = message.From.LanguageCode
	}
	if err := h.store.UpsertChat(ctx, domain.Chat{
		TelegramChatID: message.Chat.ID,
		Type:           string(message.Chat.Type),
		LanguageCode:   i18n.Resolve(languageHint).Code,
	}); err != nil {
		return fmt.Errorf("save chat: %w", err)
	}
	locale, err := h.chatLocale(ctx, message.Chat.ID, languageHint)
	if err != nil {
		return err
	}

	if message.Location != nil {
		return h.handleLocation(ctx, message, locale)
	}
	command, argument := parseCommand(message.Text)
	if command == "" {
		command = i18n.ActionForText(message.Text)
	}
	if command == "" {
		if message.Chat.Type == models.ChatTypePrivate && strings.TrimSpace(message.Text) != "" {
			return h.send(ctx, message.Chat.ID, locale.Message("unknown"), mainKeyboard(locale))
		}
		return nil
	}

	switch command {
	case "start":
		if ok, err := h.canConfigure(ctx, message, locale); err != nil || !ok {
			return err
		}
		if err := h.sendWelcome(ctx, message.Chat.ID, locale); err != nil {
			return err
		}
		if _, err := h.store.Profile(ctx, message.Chat.ID); store.IsNotFound(err) {
			return h.requestLocation(ctx, message.Chat, locale)
		} else {
			return err
		}
	case i18n.ActionLocation:
		if ok, err := h.canConfigure(ctx, message, locale); err != nil || !ok {
			return err
		}
		return h.requestLocation(ctx, message.Chat, locale)
	case i18n.ActionToday:
		return h.sendSchedule(ctx, message.Chat.ID, h.now(), locale.Message("today_title"), locale)
	case i18n.ActionTomorrow:
		return h.sendSchedule(ctx, message.Chat.ID, h.now().AddDate(0, 0, 1), locale.Message("tomorrow_title"), locale)
	case i18n.ActionNext:
		return h.sendNext(ctx, message.Chat.ID, locale)
	case i18n.ActionSettings:
		return h.sendSettings(ctx, message.Chat.ID, locale)
	case i18n.ActionReminders, "remind":
		if argument == "" {
			return h.sendReminders(ctx, message.Chat.ID, locale)
		}
		if ok, err := h.canConfigure(ctx, message, locale); err != nil || !ok {
			return err
		}
		return h.setReminders(ctx, message.Chat.ID, argument, locale)
	case i18n.ActionLanguage:
		return h.send(ctx, message.Chat.ID, locale.Message("choose_language"), languageKeyboard(locale.Code))
	case "method", "madhab", "highlat", "adjust", "delete_me":
		ok, err := h.canConfigure(ctx, message, locale)
		if err != nil || !ok {
			return err
		}
		switch command {
		case "method":
			return h.setMethod(ctx, message.Chat.ID, argument, locale)
		case "madhab":
			return h.setMadhab(ctx, message.Chat.ID, argument, locale)
		case "highlat":
			return h.setHighLatitude(ctx, message.Chat.ID, argument, locale)
		case "adjust":
			return h.setAdjustment(ctx, message.Chat.ID, argument, locale)
		default:
			return h.deleteChat(ctx, message.Chat.ID, locale)
		}
	case "privacy":
		return h.send(ctx, message.Chat.ID, locale.Message("privacy"), mainKeyboard(locale))
	case i18n.ActionHelp:
		return h.send(ctx, message.Chat.ID, locale.Message("help"), mainKeyboard(locale))
	case "status":
		if h.ownerID == 0 || message.From == nil || message.From.ID != h.ownerID {
			return nil
		}
		stats, err := h.store.Stats(ctx)
		if err != nil {
			return err
		}
		return h.send(ctx, message.Chat.ID, fmt.Sprintf(
			"<b>Global bot status</b>\nChats: %d\nProfiles: %d\nEnabled reminder rules: %d\nPending schedules: %d",
			stats.Chats, stats.Profiles, stats.EnabledRules, stats.PendingSchedules), nil)
	default:
		return h.send(ctx, message.Chat.ID, locale.Message("unknown"), mainKeyboard(locale))
	}
}

func (h *Handler) chatLocale(ctx context.Context, chatID int64, fallback string) (i18n.Locale, error) {
	chat, err := h.store.Chat(ctx, chatID)
	if store.IsNotFound(err) {
		return i18n.Resolve(fallback), nil
	}
	if err != nil {
		return i18n.Locale{}, fmt.Errorf("load chat language: %w", err)
	}
	return i18n.Resolve(chat.LanguageCode), nil
}

func (h *Handler) sendWelcome(ctx context.Context, chatID int64, locale i18n.Locale) error {
	_, err := h.bot.SendPhoto(ctx, &botapi.SendPhotoParams{
		ChatID: chatID,
		Photo: &models.InputFileUpload{
			Filename: "welcome.jpg",
			Data:     bytes.NewReader(assets.WelcomePhoto),
		},
		Caption:     locale.Message("welcome"),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: mainKeyboard(locale),
	})
	if err != nil {
		return fmt.Errorf("Telegram welcome send failed")
	}
	return nil
}

func (h *Handler) canConfigure(ctx context.Context, message *models.Message, locale i18n.Locale) (bool, error) {
	return h.canConfigureActor(ctx, message.Chat, message.From, locale)
}

func (h *Handler) canConfigureActor(ctx context.Context, chat models.Chat, user *models.User, locale i18n.Locale) (bool, error) {
	if chat.Type == models.ChatTypePrivate {
		return true, nil
	}
	if user == nil {
		return false, h.send(ctx, chat.ID, locale.Message("admin_only"), nil)
	}
	member, err := h.bot.GetChatMember(ctx, &botapi.GetChatMemberParams{ChatID: chat.ID, UserID: user.ID})
	if err != nil {
		return false, fmt.Errorf("Telegram administrator check failed")
	}
	if member.Type != models.ChatMemberTypeOwner && member.Type != models.ChatMemberTypeAdministrator {
		return false, h.send(ctx, chat.ID, locale.Message("admin_only"), nil)
	}
	return true, nil
}

func (h *Handler) send(ctx context.Context, chatID int64, text string, markup models.ReplyMarkup) error {
	_, err := h.bot.SendMessage(ctx, &botapi.SendMessageParams{
		ChatID: chatID, Text: text, ParseMode: models.ParseModeHTML, ReplyMarkup: markup,
	})
	if err != nil {
		return fmt.Errorf("Telegram send failed")
	}
	return nil
}

func (h *Handler) edit(ctx context.Context, chatID int64, messageID int, text string, markup models.ReplyMarkup) error {
	_, err := h.bot.EditMessageText(ctx, &botapi.EditMessageTextParams{
		ChatID: chatID, MessageID: messageID, Text: text,
		ParseMode: models.ParseModeHTML, ReplyMarkup: markup,
	})
	if err != nil {
		return fmt.Errorf("Telegram message edit failed")
	}
	return nil
}

func escape(value string) string { return html.EscapeString(value) }

func parseCommand(text string) (string, string) {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "/") {
		return "", ""
	}
	command := strings.TrimPrefix(fields[0], "/")
	if index := strings.IndexByte(command, '@'); index >= 0 {
		command = command[:index]
	}
	argument := ""
	if len(fields) > 1 {
		argument = strings.Join(fields[1:], " ")
	}
	return strings.ToLower(command), argument
}
