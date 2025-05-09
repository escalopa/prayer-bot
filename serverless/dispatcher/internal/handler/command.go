package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/escalopa/prayer-bot/log"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	defaultLanguageCode = "en"

	prayerDayFormat  = "02.01.2006"
	prayerTimeFormat = "15:04"
	prayerText       = `
🗓 %s,  %s

🕊 %s — %s  
🌤 %s — %s  
☀️ %s — %s  
🌇 %s — %s  
🌅 %s — %s  
🌙 %s — %s
`
)

type command string

const (
	// user commands

	startCommand       command = "start"
	helpCommand        command = "help"
	todayCommand       command = "today"
	dateCommand        command = "date" // 2 stages
	nextCommand        command = "next"
	remindCommand      command = "remind"   // 1 stage
	bugCommand         command = "bug"      // 1 stage
	feedbackCommand    command = "feedback" // 1 stage
	languageCommand    command = "language" // 1 stage
	subscribeCommand   command = "subscribe"
	unsubscribeCommand command = "unsubscribe"
	cancelCommand      command = "cancel"

	// admin commands

	adminCommand    command = "admin"
	replyCommand    command = "reply" // 1 stage
	statsCommand    command = "stats"
	announceCommand command = "announce" // 1 stage
)

func (c command) String() string {
	return string(c)
}

func (h *Handler) start(ctx context.Context, b *bot.Bot, update *models.Update) error {
	return h.help(ctx, b, update)
}

func (h *Handler) help(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("help: get chat", log.Err(err))
		return fmt.Errorf("help: get chat: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chat.ChatID,
		Text:      h.lp.GetText(chat.LanguageCode).Help,
		ParseMode: models.ParseModeMarkdown,
	})

	if err != nil {
		log.Error("help: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("help: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) today(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("today: get chat", log.Err(err))
		return fmt.Errorf("today: get chat: %v", err)
	}

	prayerDay, err := h.db.GetPrayerDay(ctx, chat.BotID, h.now(chat.BotID))
	if err != nil {
		log.Error("today: get prayer day", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("today: get prayer day: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.formatPrayerDay(chat.BotID, prayerDay, chat.LanguageCode),
	})
	if err != nil {
		log.Error("today: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("today: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) date(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("date: get chat", log.Err(err))
		return fmt.Errorf("date: get chat: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chat.ChatID,
		Text:        h.lp.GetText(chat.LanguageCode).PrayerDate,
		ReplyMarkup: h.monthsKeyboard(chat.LanguageCode),
	})
	if err != nil {
		log.Error("date: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("date: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) next(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("next: get chat", log.Err(err))
		return fmt.Errorf("next: get chat: %v", err)
	}

	date := h.now(chat.BotID)
	prayerDay, err := h.db.GetPrayerDay(ctx, chat.BotID, date)
	if err != nil {
		log.Error("next: get prayer day", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
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
		prayerDay, err = h.db.GetPrayerDay(ctx, chat.BotID, date.AddDate(0, 0, 1))
		if err != nil {
			log.Error("next: get prayer day", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
			return fmt.Errorf("next: get prayer day: %v", err)
		}
		prayerID, duration = domain.PrayerIDFajr, prayerDay.Fajr.Sub(date)
	}

	text := h.lp.GetText(chat.LanguageCode)
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chat.ChatID,
		Text:      fmt.Sprintf(text.PrayerSoon, text.Prayer[int(prayerID)], domain.FormatDuration(duration)),
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Error("next: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("next: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) remind(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("remind: get chat", log.Err(err))
		return fmt.Errorf("remind: get chat: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chat.ChatID,
		Text:        h.lp.GetText(chat.LanguageCode).Remind.Start,
		ReplyMarkup: h.remindKeyboard(),
	})
	if err != nil {
		log.Error("remind: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("remind: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) bug(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("bug: get chat", log.Err(err))
		return fmt.Errorf("bug: get chat: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Bug.Start,
	})
	if err != nil {
		log.Error("bug: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("bug: send message: %v", err)
	}

	err = h.db.SetState(ctx, chat.BotID, chat.ChatID, bugState.String())
	if err != nil {
		log.Error("bug: set state", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("bug: set state: %v", err)
	}

	return nil
}

func (h *Handler) feedback(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("feedback: get chat", log.Err(err))
		return fmt.Errorf("feedback: get chat: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Feedback.Start,
	})
	if err != nil {
		log.Error("feedback: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("feedback: send message: %v", err)
	}

	err = h.db.SetState(ctx, chat.BotID, chat.ChatID, feedbackState.String())
	if err != nil {
		log.Error("feedback: set state", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("feedback: set state: %v", err)
	}

	return nil
}

func (h *Handler) language(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("language: get chat", log.Err(err))
		return fmt.Errorf("language: get chat: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chat.ChatID,
		Text:        h.lp.GetText(chat.LanguageCode).Language.Start,
		ReplyMarkup: h.languagesKeyboard(),
	})
	if err != nil {
		log.Error("language: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("language: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) subscribe(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("subscribe: get chat", log.Err(err))
		return fmt.Errorf("subscribe: get chat: %v", err)
	}

	err = h.db.SetSubscribed(ctx, chat.BotID, chat.ChatID, true)
	if err != nil {
		log.Error("subscribe: set subscribed", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("subscribe: set subscribed: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).SubscriptionSuccess,
	})
	if err != nil {
		log.Error("subscribe: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("subscribe: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) unsubscribe(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("unsubscribe: get chat", log.Err(err))
		return fmt.Errorf("unsubscribe: get chat: %v", err)
	}

	err = h.db.SetSubscribed(ctx, chat.BotID, chat.ChatID, false)
	if err != nil {
		log.Error("unsubscribe: set subscribed", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("unsubscribe: set subscribed: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).UnsubscriptionSuccess,
	})
	if err != nil {
		log.Error("unsubscribe: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("unsubscribe: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) admin(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("admin: get chat", log.Err(err))
		return fmt.Errorf("admin: get chat: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chat.ChatID,
		Text:      h.lp.GetText(chat.LanguageCode).HelpAdmin,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Error("admin: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("admin: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) reply(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("reply: get chat", log.Err(err))
		return fmt.Errorf("reply: get chat: %v", err)
	}

	err = h.db.SetState(ctx, chat.BotID, chat.ChatID, string(replyState))
	if err != nil {
		log.Error("reply: set state", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("reply: set state: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Reply.Start,
	})
	if err != nil {
		log.Error("reply: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("reply: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) stats(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("stats: get chat", log.Err(err))
		return fmt.Errorf("stats: get chat: %v", err)
	}

	stats, err := h.db.GetStats(ctx, chat.BotID)
	if err != nil {
		log.Error("stats: get stats", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
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
		log.Error("stats: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("stats: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) announce(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("announce: get chat", log.Err(err))
		return fmt.Errorf("announce: get chat: %v", err)
	}

	err = h.db.SetState(ctx, chat.BotID, chat.ChatID, string(announceState))
	if err != nil {
		log.Error("announce: set state", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("announce: set state: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Announce.Start,
	})
	if err != nil {
		log.Error("announce: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("announce: send message: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) cancel(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("cancel: get chat", log.Err(err))
		return fmt.Errorf("cancel: get chat: %v", err)
	}

	if chat.State == string(defaultState) {
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chat.ChatID,
			Text:   h.lp.GetText(chat.LanguageCode).Noop,
		})
		if err != nil {
			log.Error("cancel: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
			return fmt.Errorf("cancel: send message: %v", err)
		}
		return nil
	}

	err = h.db.SetState(ctx, chat.BotID, chat.ChatID, string(defaultState))
	if err != nil {
		log.Error("cancel: set state", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("cancel: set state: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Cancel,
	})
	if err != nil {
		log.Error("cancel: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("cancel: send message: %v", err)
	}

	return nil
}

func (h *Handler) defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("defaultHandler: get chat", log.Err(err))
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
		log.Error("defaultHandler: get chat",
			log.Err(err),
			log.BotID(chat.BotID),
			log.ChatID(chat.ChatID),
			log.String("state", chat.State),
		)
		return fmt.Errorf("defaultHandler: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) resetState(ctx context.Context, chat *domain.Chat) {
	if chat.State == string(defaultState) {
		return
	}
	err := h.db.SetState(ctx, chat.BotID, chat.ChatID, string(defaultState))
	if err != nil {
		fmt.Printf("reset state: %v", err)
	}
}
