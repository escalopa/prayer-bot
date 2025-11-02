package handler

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/escalopa/prayer-bot/log"
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
	regexCommandTmpl command = "^/[a-zA-Z_]+@%s$" // example: /next@global_prayer_bot

	// user commands

	startCommand       command = "start"
	helpCommand        command = "help"
	todayCommand       command = "today"
	tomorrowCommand    command = "tomorrow"
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
	infoCommand     command = "info"
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

func (h *Handler) help(ctx context.Context, b *bot.Bot, _ *models.Update) error {
	chat := getContextChat(ctx)

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chat.ChatID,
		Text:      h.lp.GetText(chat.LanguageCode).Help,
		ParseMode: models.ParseModeMarkdown,
	})

	if err != nil {
		log.Error("help: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) today(ctx context.Context, b *bot.Bot, _ *models.Update) error {
	chat := getContextChat(ctx)

	prayerDay, err := h.db.GetPrayerDay(ctx, chat.BotID, h.nowDateUTC(chat.BotID))
	if err != nil {
		log.Error("today: get prayer day", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.formatPrayerDay(chat.BotID, prayerDay, chat.LanguageCode),
	})
	if err != nil {
		log.Error("today: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) tomorrow(ctx context.Context, b *bot.Bot, _ *models.Update) error {
	chat := getContextChat(ctx)

	tomorrow := h.nowDateUTC(chat.BotID).Add(24 * time.Hour)
	prayerDay, err := h.db.GetPrayerDay(ctx, chat.BotID, tomorrow)
	if err != nil {
		log.Error("tomorrow: get prayer day", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.formatPrayerDay(chat.BotID, prayerDay, chat.LanguageCode),
	})
	if err != nil {
		log.Error("tomorrow: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) date(ctx context.Context, b *bot.Bot, _ *models.Update) error {
	chat := getContextChat(ctx)

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chat.ChatID,
		Text:        h.lp.GetText(chat.LanguageCode).PrayerDate,
		ReplyMarkup: h.monthsKeyboard(chat.LanguageCode),
	})
	if err != nil {
		log.Error("date: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) next(ctx context.Context, b *bot.Bot, _ *models.Update) error {
	chat := getContextChat(ctx)

	var (
		now  = h.now(chat.BotID)
		date = h.nowDateUTC(chat.BotID)
	)

	prayerDay, err := h.db.GetPrayerDay(ctx, chat.BotID, date)
	if err != nil {
		log.Error("next: get prayer day", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	prayerID, duration := domain.PrayerIDUnknown, time.Duration(0)
	switch {
	case prayerDay.Fajr.After(now):
		prayerID, duration = domain.PrayerIDFajr, prayerDay.Fajr.Sub(now)
	case prayerDay.Shuruq.After(now):
		prayerID, duration = domain.PrayerIDShuruq, prayerDay.Shuruq.Sub(now)
	case prayerDay.Dhuhr.After(now):
		prayerID, duration = domain.PrayerIDDhuhr, prayerDay.Dhuhr.Sub(now)
	case prayerDay.Asr.After(now):
		prayerID, duration = domain.PrayerIDAsr, prayerDay.Asr.Sub(now)
	case prayerDay.Maghrib.After(now):
		prayerID, duration = domain.PrayerIDMaghrib, prayerDay.Maghrib.Sub(now)
	case prayerDay.Isha.After(now):
		prayerID, duration = domain.PrayerIDIsha, prayerDay.Isha.Sub(now)
	case prayerDay.NextDay.Fajr.After(now):
		prayerID, duration = domain.PrayerIDFajr, prayerDay.NextDay.Fajr.Sub(now)
	case prayerDay.NextDay.Shuruq.After(now):
		prayerID, duration = domain.PrayerIDShuruq, prayerDay.NextDay.Shuruq.Sub(now)
	default:
		log.Error("next: no prayer time found",
			log.BotID(chat.BotID),
			log.ChatID(chat.ChatID),
			log.String("date", date.String()),
			log.String("prayer_day", fmt.Sprintf("%+v", prayerDay)),
			log.String("now", now.String()),
		)
		return nil
	}

	text := h.lp.GetText(chat.LanguageCode)
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chat.ChatID,
		Text:      fmt.Sprintf(text.PrayerSoon, text.Prayer[int(prayerID)], domain.FormatDuration(duration)),
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Error("next: send message",
			log.Err(err),
			log.BotID(chat.BotID),
			log.ChatID(chat.ChatID),
			log.String("date", date.String()),
			log.String("prayer_day", fmt.Sprintf("%+v", prayerDay)),
		)
		return domain.ErrInternal
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) remind(ctx context.Context, b *bot.Bot, _ *models.Update) error {
	chat := getContextChat(ctx)
	text := h.lp.GetText(chat.LanguageCode)

	var messageText string
	if chat.Subscribed {
		messageText = text.RemindMenu.TitleEnabled
	} else {
		messageText = text.RemindMenu.TitleDisabled
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chat.ChatID,
		Text:        messageText,
		ReplyMarkup: h.remindMenuKeyboard(chat),
	})
	if err != nil {
		log.Error("remind: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) bug(ctx context.Context, b *bot.Bot, _ *models.Update) error {
	chat := getContextChat(ctx)

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Bug.Start,
	})
	if err != nil {
		log.Error("bug: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	err = h.db.SetState(ctx, chat.BotID, chat.ChatID, bugState.String())
	if err != nil {
		log.Error("bug: set state", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	return nil
}

func (h *Handler) feedback(ctx context.Context, b *bot.Bot, _ *models.Update) error {
	chat := getContextChat(ctx)

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Feedback.Start,
	})
	if err != nil {
		log.Error("feedback: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	err = h.db.SetState(ctx, chat.BotID, chat.ChatID, feedbackState.String())
	if err != nil {
		log.Error("feedback: set state", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	return nil
}

func (h *Handler) language(ctx context.Context, b *bot.Bot, _ *models.Update) error {
	chat := getContextChat(ctx)

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chat.ChatID,
		Text:        h.lp.GetText(chat.LanguageCode).Language.Start,
		ReplyMarkup: h.languagesKeyboard(),
	})
	if err != nil {
		log.Error("language: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) admin(ctx context.Context, b *bot.Bot, _ *models.Update) error {
	chat := getContextChat(ctx)

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chat.ChatID,
		Text:      h.lp.GetText(chat.LanguageCode).HelpAdmin,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Error("admin: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) info(ctx context.Context, b *bot.Bot, _ *models.Update) error {
	chat := getContextChat(ctx)
	text := h.lp.GetText(chat.LanguageCode)

	chatType := text.Info.Type.Private
	if isChatGroup(chat.ChatID) {
		chatType = text.Info.Type.Group
	}

	subscriptionStatus := text.Info.Disabled
	if chat.Subscribed {
		subscriptionStatus = text.Info.Enabled
	}

	jamaatInfo := ""
	if isChatGroup(chat.ChatID) {
		jamaatStatus := text.Info.Disabled
		if chat.Reminder.Jamaat.Enabled {
			jamaatStatus = text.Info.Enabled
		}
		jamaatInfo = fmt.Sprintf(text.Info.Jamaat,
			jamaatStatus,
			domain.FormatDuration(chat.Reminder.Jamaat.Delay.Fajr.Duration()),
			domain.FormatDuration(chat.Reminder.Jamaat.Delay.Dhuhr.Duration()),
			domain.FormatDuration(chat.Reminder.Jamaat.Delay.Asr.Duration()),
			domain.FormatDuration(chat.Reminder.Jamaat.Delay.Maghrib.Duration()),
			domain.FormatDuration(chat.Reminder.Jamaat.Delay.Isha.Duration()),
		)
	}

	message := fmt.Sprintf(text.Info.Default,
		fmt.Sprintf("%d", chat.ChatID),
		chatType,
		chat.LanguageCode,
		chat.State,
		subscriptionStatus,
		domain.FormatDuration(chat.Reminder.Tomorrow.Offset.Duration()),
		domain.FormatDuration(chat.Reminder.Soon.Offset.Duration()),
		jamaatInfo,
	)

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chat.ChatID,
		Text:      message,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Error("info: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) reply(ctx context.Context, b *bot.Bot, _ *models.Update) error {
	chat := getContextChat(ctx)

	err := h.db.SetState(ctx, chat.BotID, chat.ChatID, string(replyState))
	if err != nil {
		log.Error("reply: set state", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Reply.Start,
	})
	if err != nil {
		log.Error("reply: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) stats(ctx context.Context, b *bot.Bot, _ *models.Update) error {
	chat := getContextChat(ctx)

	stats, err := h.db.GetStats(ctx, chat.BotID)
	if err != nil {
		log.Error("stats: get stats", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
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
		return domain.ErrInternal
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) announce(ctx context.Context, b *bot.Bot, _ *models.Update) error {
	chat := getContextChat(ctx)

	err := h.db.SetState(ctx, chat.BotID, chat.ChatID, string(announceState))
	if err != nil {
		log.Error("announce: set state", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Announce.Start,
	})
	if err != nil {
		log.Error("announce: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) cancel(ctx context.Context, b *bot.Bot, _ *models.Update) error {
	chat := getContextChat(ctx)

	if chat.State == string(defaultState) {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chat.ChatID,
			Text:   h.lp.GetText(chat.LanguageCode).Noop,
		})
		if err != nil {
			log.Error("cancel: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
			return domain.ErrInternal
		}
		return nil
	}

	err := h.db.SetState(ctx, chat.BotID, chat.ChatID, string(defaultState))
	if err != nil {
		log.Error("cancel: set state", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chat.ChatID,
		Text:   h.lp.GetText(chat.LanguageCode).Cancel,
	})
	if err != nil {
		log.Error("cancel: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	return nil
}

func (h *Handler) defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat := getContextChat(ctx)

	var err error
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
		ok := h.isDirectoBotCommand(ctx, chat, b, update)
		if ok {
			b.ProcessUpdate(ctx, update)
			return nil
		}
		log.Info("defaultHandler: got unexpected update", log.BotID(chat.BotID), log.ChatID(chat.ChatID), "update", update)
		return nil // do nothing
	}

	if err != nil {
		log.Error("defaultHandler: process state",
			log.Err(err),
			log.BotID(chat.BotID),
			log.ChatID(chat.ChatID),
			log.String("state", chat.State),
		)
		return domain.ErrInternal
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) isDirectoBotCommand(ctx context.Context, chat *domain.Chat, b *bot.Bot, update *models.Update) bool {
	user, err := b.GetMe(ctx)
	if err != nil {
		log.Error("get me", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return false
	}

	regexCommand := fmt.Sprintf(string(regexCommandTmpl), user.Username /* bot username */)
	exp, err := regexp.Compile(regexCommand)
	if err != nil {
		log.Error("regex compile", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return false
	}

	if update.Message == nil || !exp.Match([]byte(update.Message.Text)) {
		return false
	}

	// mutate the message and try to process it again
	update.Message.Text = strings.TrimSuffix(update.Message.Text, fmt.Sprintf("@%s", user.Username))
	update.Message.Entities[0].Length = len(update.Message.Text)
	// update.Message.Entities[0].Type = models.MessageEntityTypeBotCommand
	return true
}

func (h *Handler) resetState(ctx context.Context, chat *domain.Chat) {
	if chat.State == string(defaultState) {
		return
	}
	err := h.db.SetState(ctx, chat.BotID, chat.ChatID, string(defaultState))
	if err != nil {
		log.Error("reset state", log.Err(err))
	}
}
