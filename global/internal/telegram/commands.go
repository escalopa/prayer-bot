package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot/models"

	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/hijri"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
	"github.com/escalopa/prayer-bot/global/internal/location"
	"github.com/escalopa/prayer-bot/global/internal/store"
)

func (h *Handler) handleLocation(ctx context.Context, message *models.Message, locale i18n.Locale) error {
	ok, err := h.canConfigure(ctx, message, locale)
	if err != nil || !ok {
		return err
	}
	latitude, longitude := message.Location.Latitude, message.Location.Longitude
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return h.send(ctx, message.Chat.ID, locale.Message("invalid_location"), mainKeyboard(locale))
	}
	resolved, err := h.resolver.Resolve(ctx, latitude, longitude)
	if err != nil {
		return fmt.Errorf("resolve location: %w", err)
	}
	latitude, longitude = domain.RoundedCoordinates(latitude, longitude)
	profile := domain.PrayerProfile{
		ChatID: message.Chat.ID, Latitude: latitude, Longitude: longitude,
		Timezone: resolved.Timezone, PlaceID: resolved.PlaceID,
		Method: location.RecommendedMethod(resolved.CountryCode), Madhab: domain.MadhabShafii,
		HighLatitudeRule: domain.HighLatitudeAngleBased,
	}
	if current, err := h.store.Profile(ctx, message.Chat.ID); err == nil {
		profile.Method = current.Method
		profile.Madhab = current.Madhab
		profile.HighLatitudeRule = current.HighLatitudeRule
		profile.Adjustments = current.Adjustments
		profile.HijriAdjustment = current.HijriAdjustment
		profile.LocationLabel = current.LocationLabel
	} else if !store.IsNotFound(err) {
		return fmt.Errorf("load current profile: %w", err)
	}
	profile, err = h.store.UpsertProfile(ctx, profile)
	if err != nil {
		return fmt.Errorf("save prayer profile: %w", err)
	}
	if err := h.planner.RebuildChat(ctx, message.Chat.ID, h.now()); err != nil {
		return fmt.Errorf("rebuild reminders: %w", err)
	}
	city := resolved.City
	if city == "" {
		city = resolved.Timezone
	}
	return h.send(ctx, message.Chat.ID, fmt.Sprintf(
		locale.Message("location_set"), escape(city), escape(resolved.Timezone), escape(locale.Method(profile.Method)),
	), mainKeyboard(locale))
}

func (h *Handler) requestLocation(ctx context.Context, chat models.Chat, locale i18n.Locale) error {
	if chat.Type != models.ChatTypePrivate {
		return h.send(ctx, chat.ID, locale.Message("location_group"), nil)
	}
	keyboard := &models.ReplyKeyboardMarkup{
		Keyboard: [][]models.KeyboardButton{{{
			Text: locale.Button("share_location"), RequestLocation: true,
		}}},
		ResizeKeyboard: true, OneTimeKeyboard: true,
		InputFieldPlaceholder: locale.Button("share_location"),
	}
	return h.send(ctx, chat.ID, locale.Message("location_prompt"), keyboard)
}

func (h *Handler) sendSchedule(ctx context.Context, chatID int64, date time.Time, heading string, locale i18n.Locale) error {
	profile, ok, err := h.profileOrPrompt(ctx, chatID, locale)
	if err != nil || !ok {
		return err
	}
	schedule, err := h.calculator.Day(ctx, date, profile)
	if err != nil {
		return err
	}
	return h.send(ctx, chatID, formatSchedule(heading, schedule, profile, locale), mainKeyboard(locale))
}

func (h *Handler) sendNext(ctx context.Context, chatID int64, locale i18n.Locale) error {
	profile, ok, err := h.profileOrPrompt(ctx, chatID, locale)
	if err != nil || !ok {
		return err
	}
	now := h.now()
	for day := 0; day < 2; day++ {
		schedule, err := h.calculator.Day(ctx, now.AddDate(0, 0, day), profile)
		if err != nil {
			return err
		}
		for _, prayer := range obligatoryPrayers() {
			at, found := schedule.At(prayer)
			if found && at.After(now) {
				return h.send(ctx, chatID, fmt.Sprintf(
					locale.Message("next_prayer"), escape(locale.Prayer(prayer)), at.Format("15:04"), escape(profile.Timezone),
				), mainKeyboard(locale))
			}
		}
	}
	return fmt.Errorf("could not find the next prayer")
}

func (h *Handler) sendSettings(ctx context.Context, chatID int64, locale i18n.Locale) error {
	profile, ok, err := h.profileOrPrompt(ctx, chatID, locale)
	if err != nil || !ok {
		return err
	}
	return h.send(ctx, chatID, formatSettings(profile, locale), settingsKeyboard(locale))
}

func (h *Handler) setMethod(ctx context.Context, chatID int64, argument string, locale i18n.Locale) error {
	method := domain.Method(strings.ToLower(strings.TrimSpace(argument)))
	if !method.Valid() {
		profile, ok, err := h.profileOrPrompt(ctx, chatID, locale)
		if err != nil || !ok {
			return err
		}
		return h.send(ctx, chatID, locale.Message("choose_method"), methodKeyboard(profile.Method, locale))
	}
	profile, ok, err := h.updateProfile(ctx, chatID, locale, func(profile *domain.PrayerProfile) { profile.Method = method })
	if err != nil || !ok {
		return err
	}
	return h.send(ctx, chatID, fmt.Sprintf(locale.Message("method_saved"), escape(locale.Method(profile.Method))), settingsKeyboard(locale))
}

func (h *Handler) setMadhab(ctx context.Context, chatID int64, argument string, locale i18n.Locale) error {
	madhab := domain.Madhab(strings.ToLower(strings.TrimSpace(argument)))
	if !madhab.Valid() {
		profile, ok, err := h.profileOrPrompt(ctx, chatID, locale)
		if err != nil || !ok {
			return err
		}
		return h.send(ctx, chatID, locale.Message("choose_madhab"), madhabKeyboard(profile.Madhab, locale))
	}
	profile, ok, err := h.updateProfile(ctx, chatID, locale, func(profile *domain.PrayerProfile) { profile.Madhab = madhab })
	if err != nil || !ok {
		return err
	}
	return h.send(ctx, chatID, fmt.Sprintf(locale.Message("madhab_saved"), escape(locale.Madhab(profile.Madhab))), settingsKeyboard(locale))
}

func (h *Handler) setHighLatitude(ctx context.Context, chatID int64, argument string, locale i18n.Locale) error {
	rule := domain.HighLatitudeRule(strings.ToLower(strings.TrimSpace(argument)))
	if !rule.Valid() {
		profile, ok, err := h.profileOrPrompt(ctx, chatID, locale)
		if err != nil || !ok {
			return err
		}
		return h.send(ctx, chatID, locale.Message("choose_highlat"), highLatitudeKeyboard(profile.HighLatitudeRule, locale))
	}
	profile, ok, err := h.updateProfile(ctx, chatID, locale, func(profile *domain.PrayerProfile) { profile.HighLatitudeRule = rule })
	if err != nil || !ok {
		return err
	}
	return h.send(ctx, chatID, fmt.Sprintf(locale.Message("highlat_saved"), escape(locale.HighLatitudeRule(profile.HighLatitudeRule))), settingsKeyboard(locale))
}

func (h *Handler) setAdjustment(ctx context.Context, chatID int64, argument string, locale i18n.Locale) error {
	fields := strings.Fields(argument)
	if len(fields) != 2 {
		profile, ok, err := h.profileOrPrompt(ctx, chatID, locale)
		if err != nil || !ok {
			return err
		}
		return h.send(ctx, chatID, locale.Message("choose_adjustment"), adjustmentKeyboard(profile, locale))
	}
	minutes, err := strconv.Atoi(fields[1])
	prayer := domain.Prayer(strings.ToLower(fields[0]))
	if err != nil || minutes < -30 || minutes > 30 || !prayer.Valid() {
		profile, ok, loadErr := h.profileOrPrompt(ctx, chatID, locale)
		if loadErr != nil || !ok {
			return loadErr
		}
		return h.send(ctx, chatID, locale.Message("choose_adjustment"), adjustmentKeyboard(profile, locale))
	}
	profile, ok, err := h.updateProfile(ctx, chatID, locale, func(profile *domain.PrayerProfile) {
		setAdjustmentValue(&profile.Adjustments, prayer, minutes)
	})
	if err != nil || !ok {
		return err
	}
	return h.send(ctx, chatID, fmt.Sprintf(
		locale.Message("adjust_saved"), escape(locale.Prayer(prayer)), adjustmentValue(profile.Adjustments, prayer),
	), adjustmentKeyboard(profile, locale))
}

func (h *Handler) sendReminders(ctx context.Context, chatID int64, locale i18n.Locale) error {
	if _, ok, err := h.profileOrPrompt(ctx, chatID, locale); err != nil || !ok {
		return err
	}
	state, err := h.loadReminderState(ctx, chatID)
	if err != nil {
		return err
	}
	return h.send(ctx, chatID, formatReminders(state, locale), remindersKeyboard(state, locale))
}

func (h *Handler) setReminders(ctx context.Context, chatID int64, argument string, locale i18n.Locale) error {
	switch strings.ToLower(strings.TrimSpace(argument)) {
	case "on":
		if _, ok, err := h.profileOrPrompt(ctx, chatID, locale); err != nil || !ok {
			return err
		}
		if err := h.store.EnableDefaultRules(ctx, chatID); err != nil {
			return err
		}
		if err := h.planner.RebuildChat(ctx, chatID, h.now()); err != nil {
			return err
		}
		return h.sendReminders(ctx, chatID, locale)
	case "off":
		if err := h.store.DisableRules(ctx, chatID); err != nil {
			return err
		}
		return h.sendReminders(ctx, chatID, locale)
	default:
		return h.sendReminders(ctx, chatID, locale)
	}
}

func (h *Handler) deleteChat(ctx context.Context, chatID int64, locale i18n.Locale) error {
	if err := h.store.DeleteChat(ctx, chatID); err != nil {
		return err
	}
	return h.send(ctx, chatID, locale.Message("deleted"), mainKeyboard(locale))
}

func (h *Handler) profileOrPrompt(ctx context.Context, chatID int64, locale i18n.Locale) (domain.PrayerProfile, bool, error) {
	profile, err := h.store.Profile(ctx, chatID)
	if store.IsNotFound(err) {
		return domain.PrayerProfile{}, false, h.send(ctx, chatID, locale.Message("need_location"), mainKeyboard(locale))
	}
	if err != nil {
		return domain.PrayerProfile{}, false, err
	}
	return profile, true, nil
}

func (h *Handler) updateProfile(ctx context.Context, chatID int64, locale i18n.Locale, update func(*domain.PrayerProfile)) (domain.PrayerProfile, bool, error) {
	profile, ok, err := h.profileOrPrompt(ctx, chatID, locale)
	if err != nil || !ok {
		return domain.PrayerProfile{}, ok, err
	}
	update(&profile)
	profile, err = h.store.UpsertProfile(ctx, profile)
	if err != nil {
		return domain.PrayerProfile{}, false, err
	}
	if err := h.planner.RebuildChat(ctx, chatID, h.now()); err != nil {
		return domain.PrayerProfile{}, false, err
	}
	return profile, true, nil
}

func formatSchedule(heading string, schedule domain.DaySchedule, profile domain.PrayerProfile, locale i18n.Locale) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "<b>%s</b> 🕌\n📅 %s", escape(heading), localizedDate(schedule.Date, locale))
	if hijriDate, err := hijri.FromGregorian(schedule.Date, profile.HijriAdjustment); err == nil {
		fmt.Fprintf(&builder, "\n🌙 %d %s %d %s <i>(%s)</i>", hijriDate.Day,
			escape(locale.HijriMonth(hijriDate.Month)), hijriDate.Year, escape(locale.Message("hijri_era")), escape(locale.Message("hijri_note")))
	}
	builder.WriteString("\n")
	for _, prayer := range allPrayers() {
		if at, ok := schedule.At(prayer); ok {
			fmt.Fprintf(&builder, "\n%s %s  <code>%s</code>", prayerEmoji(prayer), escape(locale.Prayer(prayer)), at.Format("15:04"))
		}
	}
	fmt.Fprintf(&builder, "\n\n🧭 %s · %s", escape(profile.Timezone), escape(locale.Method(profile.Method)))
	return builder.String()
}

func formatSettings(profile domain.PrayerProfile, locale i18n.Locale) string {
	return fmt.Sprintf("%s\n\n🌍 <b>%s:</b> %s\n🧭 <b>%s:</b> %s\n🕌 <b>%s:</b> %s\n🌙 <b>%s:</b> %s\n⏱ <b>%s:</b> %s\n🌙 <b>%s:</b> %s",
		locale.Message("settings_title"), escape(locale.Message("timezone")), escape(profile.Timezone),
		escape(locale.Message("method")), escape(locale.Method(profile.Method)),
		escape(locale.Message("madhab")), escape(locale.Madhab(profile.Madhab)),
		escape(locale.Message("highlat")), escape(locale.HighLatitudeRule(profile.HighLatitudeRule)),
		escape(locale.Message("adjustments")), formatAdjustmentSummary(profile.Adjustments, locale),
		escape(locale.Message("hijri_date")), fmt.Sprintf(locale.Message("hijri_setting"), profile.HijriAdjustment),
	)
}

func formatAdjustmentSummary(adjustments domain.Adjustments, locale i18n.Locale) string {
	parts := make([]string, 0, len(allPrayers()))
	for _, prayer := range allPrayers() {
		parts = append(parts, fmt.Sprintf("%s %+d", escape(locale.Prayer(prayer)), adjustmentValue(adjustments, prayer)))
	}
	return strings.Join(parts, " · ")
}

type reminderState struct {
	Prayer  bool
	Fasting bool
	Kahf    bool
}

func (h *Handler) loadReminderState(ctx context.Context, chatID int64) (reminderState, error) {
	rules, err := h.store.EnabledRules(ctx, chatID)
	if err != nil {
		return reminderState{}, err
	}
	var state reminderState
	for _, rule := range rules {
		switch rule.Kind {
		case domain.ReminderWeeklyFasting:
			state.Fasting = true
		case domain.ReminderWeeklyKahf:
			state.Kahf = true
		default:
			state.Prayer = true
		}
	}
	return state, nil
}

func formatReminders(state reminderState, locale i18n.Locale) string {
	status := func(enabled bool) string {
		if enabled {
			return "✅ " + escape(locale.Message("enabled"))
		}
		return "○ " + escape(locale.Message("disabled"))
	}
	return fmt.Sprintf("%s\n\n🔔 <b>%s</b> · %s\n\n🌙 <b>%s</b> · %s\n   %s\n\n📖 <b>%s</b> · %s\n   %s",
		locale.Message("reminders_title"), escape(locale.Button("prayer_reminders")), status(state.Prayer),
		escape(locale.Button("fasting_reminders")), status(state.Fasting), escape(locale.Message("fasting_schedule")),
		escape(locale.Button("kahf_reminders")), status(state.Kahf), escape(locale.Message("kahf_schedule")))
}

func localizedDate(date time.Time, locale i18n.Locale) string {
	return fmt.Sprintf("%d %s %d", date.Day(), escape(locale.Month(int(date.Month()))), date.Year())
}

func allPrayers() []domain.Prayer {
	return []domain.Prayer{domain.PrayerFajr, domain.PrayerSunrise, domain.PrayerDhuhr, domain.PrayerAsr, domain.PrayerMaghrib, domain.PrayerIsha}
}

func obligatoryPrayers() []domain.Prayer {
	return []domain.Prayer{domain.PrayerFajr, domain.PrayerDhuhr, domain.PrayerAsr, domain.PrayerMaghrib, domain.PrayerIsha}
}

func prayerEmoji(prayer domain.Prayer) string {
	switch prayer {
	case domain.PrayerFajr:
		return "🌙"
	case domain.PrayerSunrise:
		return "🌅"
	case domain.PrayerDhuhr:
		return "☀️"
	case domain.PrayerAsr:
		return "🌤"
	case domain.PrayerMaghrib:
		return "🌇"
	default:
		return "🌌"
	}
}

func adjustmentValue(adjustments domain.Adjustments, prayer domain.Prayer) int {
	switch prayer {
	case domain.PrayerFajr:
		return adjustments.Fajr
	case domain.PrayerSunrise:
		return adjustments.Sunrise
	case domain.PrayerDhuhr:
		return adjustments.Dhuhr
	case domain.PrayerAsr:
		return adjustments.Asr
	case domain.PrayerMaghrib:
		return adjustments.Maghrib
	case domain.PrayerIsha:
		return adjustments.Isha
	default:
		return 0
	}
}

func setAdjustmentValue(adjustments *domain.Adjustments, prayer domain.Prayer, value int) {
	switch prayer {
	case domain.PrayerFajr:
		adjustments.Fajr = value
	case domain.PrayerSunrise:
		adjustments.Sunrise = value
	case domain.PrayerDhuhr:
		adjustments.Dhuhr = value
	case domain.PrayerAsr:
		adjustments.Asr = value
	case domain.PrayerMaghrib:
		adjustments.Maghrib = value
	case domain.PrayerIsha:
		adjustments.Isha = value
	}
}
