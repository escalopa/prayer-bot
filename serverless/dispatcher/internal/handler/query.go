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
	languageQuery query = "language|"

	remindMenuQuery         query = "remind:menu|"
	remindToggleQuery       query = "remind:toggle|"
	remindEditQuery         query = "remind:edit:"
	remindAdjustQuery       query = "remind:adjust:"
	remindJamaatMenuQuery   query = "remind:jamaat:menu|"
	remindJamaatToggleQuery query = "remind:jamaat:toggle|"
	remindJamaatEditQuery   query = "remind:jamaat:edit:"
	remindJamaatAdjustQuery query = "remind:jamaat:adjust:"
	remindBackQuery         query = "remind:back:"
	remindCloseQuery        query = "remind:close|"
)

const (
	TomorrowMinOffset = 0 * time.Minute
	TomorrowMaxOffset = 6 * time.Hour

	SoonMinOffset = 5 * time.Minute
	SoonMaxOffset = 1 * time.Hour

	JamaatMinDelay = 5 * time.Minute
	JamaatMaxDelay = 6 * time.Hour
)

func (q query) String() string {
	return string(q)
}

func clampOffset(value, min, max time.Duration) time.Duration {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
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
		ReplyMarkup: h.daysKeyboard(h.nowDateUTC(chat.BotID), month),
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
	date := domain.DateUTC(day, time.Month(month), h.nowDateUTC(chat.BotID).Year())

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

func (h *Handler) remindMenuQuery(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat := getContextChat(ctx)
	text := h.lp.GetText(chat.LanguageCode)

	var messageText string
	if chat.Subscribed {
		messageText = text.RemindMenu.TitleEnabled
	} else {
		messageText = text.RemindMenu.TitleDisabled
	}

	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chat.ChatID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		Text:        messageText,
		ReplyMarkup: h.remindMenuKeyboard(chat),
	})
	if err != nil {
		log.Error("remindMenuQuery: edit message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	if err != nil {
		log.Error("remindMenuQuery: answer callback", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	return nil
}

func (h *Handler) remindToggleQuery(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat := getContextChat(ctx)

	newSubscribed := !chat.Subscribed
	err := h.db.SetSubscribed(ctx, chat.BotID, chat.ChatID, newSubscribed)
	if err != nil {
		log.Error("remindToggleQuery: set subscribed", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	chat.Subscribed = newSubscribed
	ctx = setContextChat(ctx, chat)

	h.remindMenuQuery(ctx, b, update)
	return nil
}

func (h *Handler) remindEditQuery(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat := getContextChat(ctx)
	text := h.lp.GetText(chat.LanguageCode)

	parts := strings.Split(update.CallbackQuery.Data, ":")
	if len(parts) < 3 {
		log.Error("remindEditQuery: invalid callback data", log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	reminderType := domain.ReminderType(strings.TrimSuffix(parts[2], "|"))

	var messageText string
	var offset time.Duration
	switch reminderType {
	case domain.ReminderTypeTomorrow:
		offset = chat.Reminder.Tomorrow.Offset.Duration()
		messageText = fmt.Sprintf("%s - %s", text.RemindEdit.TitleTomorrow, domain.FormatDuration(offset))
	case domain.ReminderTypeSoon:
		offset = chat.Reminder.Soon.Offset.Duration()
		messageText = fmt.Sprintf("%s - %s", text.RemindEdit.TitleSoon, domain.FormatDuration(offset))
	default:
		log.Error("remindEditQuery: unknown reminder type",
			log.BotID(chat.BotID),
			log.ChatID(chat.ChatID),
			log.String("type", reminderType.String()),
		)
		return domain.ErrInternal
	}

	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chat.ChatID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		Text:        messageText,
		ReplyMarkup: h.remindEditKeyboard(reminderType, chat.LanguageCode),
	})
	if err != nil {
		log.Error("remindEditQuery: edit message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	if err != nil {
		log.Error("remindEditQuery: answer callback", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	return nil
}

func (h *Handler) remindAdjustQuery(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat := getContextChat(ctx)

	parts := strings.Split(update.CallbackQuery.Data, ":")
	if len(parts) < 4 {
		log.Error("remindAdjustQuery: invalid callback data", log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	reminderType := domain.ReminderType(parts[2])
	adjustment := parseAdjustment(strings.TrimSuffix(parts[3], "|"))

	var currentOffset time.Duration
	var minOffset, maxOffset time.Duration

	switch reminderType {
	case domain.ReminderTypeTomorrow:
		currentOffset = chat.Reminder.Tomorrow.Offset.Duration()
		minOffset = TomorrowMinOffset
		maxOffset = TomorrowMaxOffset
	case domain.ReminderTypeSoon:
		currentOffset = chat.Reminder.Soon.Offset.Duration()
		minOffset = SoonMinOffset
		maxOffset = SoonMaxOffset
	default:
		log.Error("remindAdjustQuery: unknown reminder type", log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	newOffset := clampOffset(currentOffset+adjustment, minOffset, maxOffset)

	// If value didn't change (hit limit), just answer callback and return
	if newOffset == currentOffset {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
		return nil
	}

	err := h.db.SetReminderOffset(ctx, chat.BotID, chat.ChatID, reminderType, newOffset)
	if err != nil {
		log.Error("remindAdjustQuery: set reminder offset", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	switch reminderType {
	case domain.ReminderTypeTomorrow:
		chat.Reminder.Tomorrow.Offset = domain.Duration(newOffset)
	case domain.ReminderTypeSoon:
		chat.Reminder.Soon.Offset = domain.Duration(newOffset)
	}

	ctx = setContextChat(ctx, chat)
	return h.remindEditQuery(ctx, b, update)
}

func (h *Handler) remindJamaatMenuQuery(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat := getContextChat(ctx)

	if !isChatGroup(chat.ChatID) {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
		return nil
	}

	text := h.lp.GetText(chat.LanguageCode)

	var messageText string
	if chat.Reminder.Jamaat.Enabled {
		messageText = text.JamaatMenu.TitleEnabled
	} else {
		messageText = text.JamaatMenu.TitleDisabled
	}

	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chat.ChatID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		Text:        messageText,
		ReplyMarkup: h.jammatMenuKeyboard(chat),
	})
	if err != nil {
		log.Error("remindJamaatMenuQuery: edit message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	if err != nil {
		log.Error("remindJamaatMenuQuery: answer callback", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	return nil
}

func (h *Handler) remindJamaatToggleQuery(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat := getContextChat(ctx)

	if !isChatGroup(chat.ChatID) {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
		return nil
	}

	newEnabled := !chat.Reminder.Jamaat.Enabled
	err := h.db.SetJamaatEnabled(ctx, chat.BotID, chat.ChatID, newEnabled)
	if err != nil {
		log.Error("remindJamaatToggleQuery: set jamaat enabled", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	chat.Reminder.Jamaat.Enabled = newEnabled
	ctx = setContextChat(ctx, chat)
	return h.remindJamaatMenuQuery(ctx, b, update)
}

func (h *Handler) remindJamaatEditQuery(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat := getContextChat(ctx)

	if !isChatGroup(chat.ChatID) {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
		return nil
	}

	text := h.lp.GetText(chat.LanguageCode)

	parts := strings.Split(update.CallbackQuery.Data, ":")
	if len(parts) < 4 {
		log.Error("remindJamaatEditQuery: invalid callback data", log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	prayerName := strings.TrimSuffix(parts[3], "|")
	prayerID := domain.ParsePrayerID(prayerName)

	delay := chat.Reminder.Jamaat.Delay.GetDelayByPrayerID(prayerID)
	messageText := fmt.Sprintf(text.JamaatEdit.Title, text.Prayer[int(prayerID)]) + " - " + domain.FormatDuration(delay)

	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chat.ChatID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		Text:        messageText,
		ReplyMarkup: h.jammatEditKeyboard(prayerID, chat.LanguageCode),
	})
	if err != nil {
		log.Error("remindJamaatEditQuery: edit message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	if err != nil {
		log.Error("remindJamaatEditQuery: answer callback", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	return nil
}

func (h *Handler) remindJamaatAdjustQuery(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat := getContextChat(ctx)

	if !isChatGroup(chat.ChatID) {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
		return nil
	}

	parts := strings.Split(update.CallbackQuery.Data, ":")
	if len(parts) < 5 {
		log.Error("remindJamaatAdjustQuery: invalid callback data", log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	prayerName := parts[3]
	prayerID := domain.ParsePrayerID(prayerName)
	adjustment := parseAdjustment(strings.TrimSuffix(parts[4], "|"))

	currentDelay := chat.Reminder.Jamaat.Delay.GetDelayByPrayerID(prayerID)
	newDelay := clampOffset(currentDelay+adjustment, JamaatMinDelay, JamaatMaxDelay)

	// If value didn't change (hit limit), just answer callback and return
	if newDelay == currentDelay {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
		return nil
	}

	err := h.db.SetJamaatDelay(ctx, chat.BotID, chat.ChatID, prayerID, newDelay)
	if err != nil {
		log.Error("remindJamaatAdjustQuery: set jamaat delay", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	chat.Reminder.Jamaat.Delay.SetDelayByPrayerID(prayerID, newDelay)
	ctx = setContextChat(ctx, chat)
	return h.remindJamaatEditQuery(ctx, b, update)
}

func (h *Handler) remindBackQuery(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat := getContextChat(ctx)
	text := h.lp.GetText(chat.LanguageCode)

	parts := strings.Split(update.CallbackQuery.Data, ":")
	if len(parts) < 3 {
		log.Error("remindBackQuery: invalid callback data", log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	destination := strings.TrimSuffix(parts[2], "|")

	var messageText string
	var keyboard *models.InlineKeyboardMarkup

	switch destination {
	case "menu":
		if chat.Subscribed {
			messageText = text.RemindMenu.TitleEnabled
		} else {
			messageText = text.RemindMenu.TitleDisabled
		}
		keyboard = h.remindMenuKeyboard(chat)
	case "jamaat":
		if chat.Reminder.Jamaat.Enabled {
			messageText = text.JamaatMenu.TitleEnabled
		} else {
			messageText = text.JamaatMenu.TitleDisabled
		}
		keyboard = h.jammatMenuKeyboard(chat)
	default:
		log.Error("remindBackQuery: unknown destination", log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chat.ChatID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		Text:        messageText,
		ReplyMarkup: keyboard,
	})
	if err != nil {
		log.Error("remindBackQuery: edit message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	if err != nil {
		log.Error("remindBackQuery: answer callback", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	return nil
}

func (h *Handler) remindCloseQuery(ctx context.Context, b *bot.Bot, update *models.Update) error {
	chat := getContextChat(ctx)

	_, err := b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chat.ChatID,
		MessageID: update.CallbackQuery.Message.Message.ID,
	})
	if err != nil {
		log.Error("remindCloseQuery: delete message", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	if err != nil {
		log.Error("remindCloseQuery: answer callback", log.Err(err), log.BotID(chat.BotID), log.ChatID(chat.ChatID))
		return domain.ErrInternal
	}

	return nil
}
