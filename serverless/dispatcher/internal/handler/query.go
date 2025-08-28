package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/escalopa/prayer-bot/log"

	"github.com/escalopa/prayer-bot/domain"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type query string

const (
	dataSplitterQuery = "|"

	emptyQuery query = "empty|"

	dayQuery      query = "date:day|"
	monthQuery    query = "date:month|"
	remindQuery   query = "remind|"
	languageQuery query = "language|"
)

func (q query) String() string {
	return string(q)
}

func (h *Handler) emptyQuery(ctx context.Context, b *bot.Bot, update *models.Update) error {
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})
	return nil
}

func (h *Handler) monthQuery(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("monthQuery: get chat", log.Err(err))
		return fmt.Errorf("monthQuery: get chat: %v", err)
	}

	month, _ := strconv.Atoi(strings.TrimPrefix(update.CallbackQuery.Data, monthQuery.String()))
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chat.ChatID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		Text:        h.lp.GetText(chat.LanguageCode).PrayerDate,
		ReplyMarkup: h.daysKeyboard(h.nowUTC(chat.BotID), month),
	})
	if err != nil {
		log.Error("monthQuery: edit message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("monthQuery: edit message: %v", err)
	}

	_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	if err != nil {
		log.Error("monthQuery: answer query query", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("monthQuery: answer query query: %v", err)
	}

	return nil
}

func (h *Handler) dayQuery(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("dayQuery: get chat", log.Err(err))
		return fmt.Errorf("dayQuery: get chat: %v", err)
	}

	parts := strings.Split(update.CallbackQuery.Data, dataSplitterQuery)
	month, _ := strconv.Atoi(parts[1])
	day, _ := strconv.Atoi(parts[2])

	date := h.nowUTC(chat.BotID)
	date = domain.Date(day, time.Month(month), date.Year())

	prayerDay, err := h.db.GetPrayerDay(ctx, chat.BotID, date)
	if err != nil {
		log.Error("dayQuery: get prayer day", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("dayQuery: get prayer day: %v", err)
	}

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chat.ChatID,
		MessageID: update.CallbackQuery.Message.Message.ID,
		Text:      h.formatPrayerDay(chat.BotID, prayerDay, chat.LanguageCode),
	})
	if err != nil {
		log.Error("dayQuery: edit message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("dayQuery: edit message: %v", err)
	}

	_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	if err != nil {
		log.Error("dayQuery: answer query query", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("dayQuery: answer query query: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) remindQuery(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("remindQuery: get chat", log.Err(err))
		return fmt.Errorf("remindQuery: get chat: %v", err)
	}

	reminderOffset, _ := strconv.Atoi(strings.TrimPrefix(update.CallbackQuery.Data, remindQuery.String()))

	err = h.db.SetReminderOffset(ctx, chat.BotID, chat.ChatID, int32(reminderOffset))
	if err != nil {
		log.Error("remindQuery: set remind offset", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("remindQuery: set remind offset: %v", err)
	}

	err = h.db.SetSubscribed(ctx, chat.BotID, chat.ChatID, true)
	if err != nil {
		log.Error("remindQuery: set subscribed", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("remindQuery: set subscribed: %v", err)
	}

	message := fmt.Sprintf(h.lp.GetText(chat.LanguageCode).Remind.Success, reminderOffset)
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chat.ChatID,
		MessageID: update.CallbackQuery.Message.Message.ID,
		Text:      message,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Error("remindQuery: edit message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("remindQuery: edit message: %v", err)
	}

	_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	if err != nil {
		log.Error("remindQuery: answer query query", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("remindQuery: answer query query: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) languageQuery(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat, err := h.getChat(ctx, update)
	if err != nil {
		log.Error("languageQuery: get chat", log.Err(err))
		return fmt.Errorf("languageQuery: get chat: %v", err)
	}

	languageCode := strings.TrimPrefix(update.CallbackQuery.Data, languageQuery.String())
	if !h.lp.IsSupportedCode(languageCode) {
		log.Error("languageQuery: unsupported language code",
			log.BotID(chat.BotID),
			log.ChatID(chat.ChatID),
			log.String("language_code", languageCode),
		)
		return fmt.Errorf("languageQuery: unsupported language code: %s", languageCode)
	}

	err = h.db.SetLanguageCode(ctx, chat.BotID, chat.ChatID, languageCode)
	if err != nil {
		log.Error("languageQuery: set language code", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("languageQuery: set language code: %v", err)
	}

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chat.ChatID,
		MessageID: update.CallbackQuery.Message.Message.ID,
		Text:      fmt.Sprintf(h.lp.GetText(languageCode).Language.Success, languageCode),
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Error("languageQuery: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("languageQuery: send message: %v", err)
	}

	_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	if err != nil {
		log.Error("languageQuery: answer query query", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return fmt.Errorf("languageQuery: answer query query: %v", err)
	}

	h.resetState(ctx, chat)
	return nil
}
