package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	botapi "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
)

func (h *Handler) handleCallback(ctx context.Context, query *models.CallbackQuery) error {
	_, _ = h.bot.AnswerCallbackQuery(ctx, &botapi.AnswerCallbackQueryParams{CallbackQueryID: query.ID})
	message := query.Message.Message
	if message == nil || message.Chat.Type == models.ChatTypeChannel {
		return nil
	}
	languageHint := query.From.LanguageCode
	if languageHint == "" {
		languageHint = "en"
	}
	if err := h.store.UpsertChat(ctx, domain.Chat{
		TelegramChatID: message.Chat.ID,
		Type:           string(message.Chat.Type),
		LanguageCode:   i18n.Resolve(languageHint).Code,
	}); err != nil {
		return fmt.Errorf("save callback chat: %w", err)
	}
	locale, err := h.chatLocale(ctx, message.Chat.ID, languageHint)
	if err != nil {
		return err
	}

	if strings.HasPrefix(query.Data, "admin:") {
		if !h.isOwner(message.Chat, &query.From) {
			return nil
		}
		view, ok := parseAdminView(query.Data)
		if !ok {
			return nil
		}
		return h.editAdminDashboard(ctx, message, view)
	}

	if strings.HasPrefix(query.Data, "language:") {
		if ok, err := h.canConfigureActor(ctx, message.Chat, &query.From, locale); err != nil || !ok {
			return err
		}
		code := strings.TrimPrefix(query.Data, "language:")
		selected := i18n.Resolve(code)
		if selected.Code != code {
			return nil
		}
		if err := h.store.SetLanguage(ctx, message.Chat.ID, selected.Code); err != nil {
			return err
		}
		if err := h.edit(ctx, message.Chat.ID, message.ID, selected.Message("choose_language"), languageKeyboard(selected.Code)); err != nil {
			return err
		}
		return h.send(ctx, message.Chat.ID, selected.Message("language_saved"), mainKeyboard(selected))
	}

	switch query.Data {
	case "close":
		return h.edit(ctx, message.Chat.ID, message.ID, escape(message.Text), nil)
	case "settings":
		profile, ok, err := h.profileOrPrompt(ctx, message.Chat.ID, locale)
		if err != nil || !ok {
			return err
		}
		return h.edit(ctx, message.Chat.ID, message.ID, formatSettings(profile, locale), settingsKeyboard(locale))
	case "settings:method", "settings:madhab", "settings:highlat", "settings:adjustments", "settings:hijri":
		profile, ok, err := h.profileOrPrompt(ctx, message.Chat.ID, locale)
		if err != nil || !ok {
			return err
		}
		switch query.Data {
		case "settings:method":
			return h.edit(ctx, message.Chat.ID, message.ID, locale.Message("choose_method"), methodKeyboard(profile.Method, locale))
		case "settings:madhab":
			return h.edit(ctx, message.Chat.ID, message.ID, locale.Message("choose_madhab"), madhabKeyboard(profile.Madhab, locale))
		case "settings:highlat":
			return h.edit(ctx, message.Chat.ID, message.ID, locale.Message("choose_highlat"), highLatitudeKeyboard(profile.HighLatitudeRule, locale))
		case "settings:hijri":
			return h.edit(ctx, message.Chat.ID, message.ID, locale.Message("choose_hijri"), hijriKeyboard(profile.HijriAdjustment, locale))
		default:
			return h.edit(ctx, message.Chat.ID, message.ID, locale.Message("choose_adjustment"), adjustmentKeyboard(profile, locale))
		}
	}

	if ok, err := h.canConfigureActor(ctx, message.Chat, &query.From, locale); err != nil || !ok {
		return err
	}
	switch {
	case strings.HasPrefix(query.Data, "method:"):
		method := domain.Method(strings.TrimPrefix(query.Data, "method:"))
		if !method.Valid() {
			return nil
		}
		profile, ok, err := h.updateProfile(ctx, message.Chat.ID, locale, func(profile *domain.PrayerProfile) { profile.Method = method })
		if err != nil || !ok {
			return err
		}
		return h.edit(ctx, message.Chat.ID, message.ID, formatSettings(profile, locale), settingsKeyboard(locale))
	case strings.HasPrefix(query.Data, "madhab:"):
		madhab := domain.Madhab(strings.TrimPrefix(query.Data, "madhab:"))
		if !madhab.Valid() {
			return nil
		}
		profile, ok, err := h.updateProfile(ctx, message.Chat.ID, locale, func(profile *domain.PrayerProfile) { profile.Madhab = madhab })
		if err != nil || !ok {
			return err
		}
		return h.edit(ctx, message.Chat.ID, message.ID, formatSettings(profile, locale), settingsKeyboard(locale))
	case strings.HasPrefix(query.Data, "highlat:"):
		rule := domain.HighLatitudeRule(strings.TrimPrefix(query.Data, "highlat:"))
		if !rule.Valid() {
			return nil
		}
		profile, ok, err := h.updateProfile(ctx, message.Chat.ID, locale, func(profile *domain.PrayerProfile) { profile.HighLatitudeRule = rule })
		if err != nil || !ok {
			return err
		}
		return h.edit(ctx, message.Chat.ID, message.ID, formatSettings(profile, locale), settingsKeyboard(locale))
	case strings.HasPrefix(query.Data, "adjust:"):
		prayer := domain.Prayer(strings.TrimPrefix(query.Data, "adjust:"))
		if !prayer.Valid() {
			return nil
		}
		profile, ok, err := h.profileOrPrompt(ctx, message.Chat.ID, locale)
		if err != nil || !ok {
			return err
		}
		return h.edit(ctx, message.Chat.ID, message.ID, fmt.Sprintf(
			locale.Message("adjust_prayer"), escape(locale.Prayer(prayer)), adjustmentValue(profile.Adjustments, prayer),
		), adjustmentDetailKeyboard(prayer, locale))
	case strings.HasPrefix(query.Data, "adjust_delta:"), strings.HasPrefix(query.Data, "adjust_set:"):
		return h.handleAdjustmentCallback(ctx, message, query.Data, locale)
	case strings.HasPrefix(query.Data, "hijri:"):
		adjustment, err := strconv.Atoi(strings.TrimPrefix(query.Data, "hijri:"))
		if err != nil || adjustment < -2 || adjustment > 2 {
			return nil
		}
		profile, ok, err := h.updateProfile(ctx, message.Chat.ID, locale, func(profile *domain.PrayerProfile) {
			profile.HijriAdjustment = adjustment
		})
		if err != nil || !ok {
			return err
		}
		return h.edit(ctx, message.Chat.ID, message.ID, formatSettings(profile, locale), settingsKeyboard(locale))
	case strings.HasPrefix(query.Data, "reminders:"):
		return h.handleReminderCallback(ctx, message, query.Data, locale)
	default:
		return nil
	}
}

func (h *Handler) handleReminderCallback(ctx context.Context, message *models.Message, data string, locale i18n.Locale) error {
	parts := strings.Split(data, ":")
	if len(parts) == 2 { // Backward-compatible buttons from the first UX build.
		parts = []string{"reminders", "prayer", parts[1]}
	}
	if len(parts) == 3 && parts[1] == "pre" {
		state, err := h.loadReminderState(ctx, message.Chat.ID)
		if err != nil {
			return err
		}
		switch parts[2] {
		case "choose":
			return h.edit(ctx, message.Chat.ID, message.ID,
				locale.Message("choose_pre_reminder"), preReminderKeyboard(state.PrePrayerMinutes, locale))
		case "back":
			return h.edit(ctx, message.Chat.ID, message.ID,
				formatReminders(state, locale), remindersKeyboard(state, locale))
		}
		minutes, err := strconv.Atoi(parts[2])
		if err != nil || !domain.ValidPreReminderMinutes(minutes) {
			return nil
		}
		if _, ok, err := h.profileOrPrompt(ctx, message.Chat.ID, locale); err != nil || !ok {
			return err
		}
		if err := h.store.ConfigurePrayerRules(ctx, message.Chat.ID, true, minutes); err != nil {
			return err
		}
		if err := h.planner.RebuildChat(ctx, message.Chat.ID, h.now()); err != nil {
			return err
		}
		state, err = h.loadReminderState(ctx, message.Chat.ID)
		if err != nil {
			return err
		}
		return h.edit(ctx, message.Chat.ID, message.ID,
			formatReminders(state, locale), remindersKeyboard(state, locale))
	}
	if len(parts) != 3 || (parts[2] != "on" && parts[2] != "off") {
		return nil
	}
	enabled := parts[2] == "on"
	if enabled {
		if _, ok, err := h.profileOrPrompt(ctx, message.Chat.ID, locale); err != nil || !ok {
			return err
		}
	}
	switch parts[1] {
	case "prayer":
		if enabled {
			if err := h.store.ConfigurePrayerRules(ctx, message.Chat.ID, true, 0); err != nil {
				return err
			}
		} else if err := h.store.DisableRules(ctx, message.Chat.ID); err != nil {
			return err
		}
	case "fasting":
		if err := h.store.SetWeeklyRule(ctx, message.Chat.ID, domain.ReminderWeeklyFasting, enabled); err != nil {
			return err
		}
	case "kahf":
		if err := h.store.SetWeeklyRule(ctx, message.Chat.ID, domain.ReminderWeeklyKahf, enabled); err != nil {
			return err
		}
	case "occasion_major":
		if err := h.store.SetOccasionRule(ctx, message.Chat.ID, domain.ReminderOccasionMajor, enabled); err != nil {
			return err
		}
	case "occasion_fasting":
		if err := h.store.SetOccasionRule(ctx, message.Chat.ID, domain.ReminderOccasionFasting, enabled); err != nil {
			return err
		}
	case "occasion_observed":
		if err := h.store.SetOccasionRule(ctx, message.Chat.ID, domain.ReminderOccasionObserved, enabled); err != nil {
			return err
		}
	default:
		return nil
	}
	if enabled {
		if err := h.planner.RebuildChat(ctx, message.Chat.ID, h.now()); err != nil {
			return err
		}
	}
	state, err := h.loadReminderState(ctx, message.Chat.ID)
	if err != nil {
		return err
	}
	return h.edit(ctx, message.Chat.ID, message.ID, formatReminders(state, locale), remindersKeyboard(state, locale))
}

func (h *Handler) handleAdjustmentCallback(ctx context.Context, message *models.Message, data string, locale i18n.Locale) error {
	parts := strings.Split(data, ":")
	if len(parts) != 3 {
		return nil
	}
	prayer := domain.Prayer(parts[1])
	value, err := strconv.Atoi(parts[2])
	if err != nil || !prayer.Valid() {
		return nil
	}
	profile, ok, err := h.profileOrPrompt(ctx, message.Chat.ID, locale)
	if err != nil || !ok {
		return err
	}
	if parts[0] == "adjust_delta" {
		value += adjustmentValue(profile.Adjustments, prayer)
	}
	if value < -30 {
		value = -30
	}
	if value > 30 {
		value = 30
	}
	profile, ok, err = h.updateProfile(ctx, message.Chat.ID, locale, func(profile *domain.PrayerProfile) {
		setAdjustmentValue(&profile.Adjustments, prayer, value)
	})
	if err != nil || !ok {
		return err
	}
	return h.edit(ctx, message.Chat.ID, message.ID, fmt.Sprintf(
		locale.Message("adjust_prayer"), escape(locale.Prayer(prayer)), adjustmentValue(profile.Adjustments, prayer),
	), adjustmentDetailKeyboard(prayer, locale))
}
