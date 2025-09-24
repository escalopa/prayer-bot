package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/escalopa/prayer-bot/log"
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
	chat := getContextChat(ctx)

	month, _ := strconv.Atoi(strings.TrimPrefix(update.CallbackQuery.Data, monthQuery.String()))
	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chat.ChatID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		Text:        h.lp.GetText(chat.LanguageCode).PrayerDate,
		ReplyMarkup: h.daysKeyboard(h.nowUTC(chat.BotID), month),
	})
	if err != nil {
		log.Error("monthQuery: edit message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	if err != nil {
		log.Error("monthQuery: answer query query", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	return nil
}

func (h *Handler) dayQuery(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat := getContextChat(ctx)

	parts := strings.Split(update.CallbackQuery.Data, dataSplitterQuery)
	month, _ := strconv.Atoi(parts[1])
	day, _ := strconv.Atoi(parts[2])
	date := domain.DateUTC(day, time.Month(month), h.nowUTC(chat.BotID).Year())

	prayerDay, err := h.db.GetPrayerDay(ctx, chat.BotID, date)
	if err != nil {
		log.Error("dayQuery: get prayer day", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chat.ChatID,
		MessageID: update.CallbackQuery.Message.Message.ID,
		Text:      h.formatPrayerDay(chat.BotID, prayerDay, chat.LanguageCode),
	})
	if err != nil {
		log.Error("dayQuery: edit message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	if err != nil {
		log.Error("dayQuery: answer query query", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) remindQuery(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat := getContextChat(ctx)

	reminderOffset, _ := strconv.Atoi(strings.TrimPrefix(update.CallbackQuery.Data, remindQuery.String()))

	err := h.db.SetReminderOffset(ctx, chat.BotID, chat.ChatID, int32(reminderOffset))
	if err != nil {
		log.Error("remindQuery: set remind offset", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	err = h.db.SetSubscribed(ctx, chat.BotID, chat.ChatID, true)
	if err != nil {
		log.Error("remindQuery: set subscribed", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
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
		return domain.ErrInternal
	}

	_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	if err != nil {
		log.Error("remindQuery: answer query query", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	h.resetState(ctx, chat)
	return nil
}

func (h *Handler) languageQuery(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat := getContextChat(ctx)

	languageCode := strings.TrimPrefix(update.CallbackQuery.Data, languageQuery.String())
	if !h.lp.IsSupportedCode(languageCode) {
		log.Error("languageQuery: unsupported language code",
			log.BotID(chat.BotID),
			log.ChatID(chat.ChatID),
			log.String("language_code", languageCode),
		)
		return fmt.Errorf("languageQuery: unsupported language code: %s", languageCode)
	}

	err := h.db.SetLanguageCode(ctx, chat.BotID, chat.ChatID, languageCode)
	if err != nil {
		log.Error("languageQuery: set language code", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chat.ChatID,
		MessageID: update.CallbackQuery.Message.Message.ID,
		Text:      fmt.Sprintf(h.lp.GetText(languageCode).Language.Success, languageCode),
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Error("languageQuery: send message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	if err != nil {
		log.Error("languageQuery: answer query query", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	h.resetState(ctx, chat)
	return nil
}
