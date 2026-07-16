package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	botapi "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/location"
	"github.com/escalopa/prayer-bot/global/internal/prayertime"
	"github.com/escalopa/prayer-bot/global/internal/reminders"
	"github.com/escalopa/prayer-bot/global/internal/store"
)

type Bot interface {
	SendMessage(context.Context, *botapi.SendMessageParams) (*models.Message, error)
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
	message := update.Message
	if message == nil || message.Chat.Type == models.ChatTypeChannel {
		return nil
	}
	language := "en"
	if message.From != nil && message.From.LanguageCode != "" {
		language = message.From.LanguageCode
	}
	if err := h.store.UpsertChat(ctx, domain.Chat{
		TelegramChatID: message.Chat.ID,
		Type:           string(message.Chat.Type),
		LanguageCode:   language,
	}); err != nil {
		return fmt.Errorf("save chat: %w", err)
	}

	if message.Location != nil {
		return h.handleLocation(ctx, message)
	}
	command, argument := parseCommand(message.Text)
	if command == "" {
		return nil
	}

	switch command {
	case "start":
		if ok, err := h.canConfigure(ctx, message); err != nil || !ok {
			return err
		}
		return h.requestLocation(ctx, message.Chat)
	case "location":
		if ok, err := h.canConfigure(ctx, message); err != nil || !ok {
			return err
		}
		return h.requestLocation(ctx, message.Chat)
	case "today":
		return h.sendSchedule(ctx, message.Chat.ID, h.now(), "Today's prayer times")
	case "tomorrow":
		return h.sendSchedule(ctx, message.Chat.ID, h.now().AddDate(0, 0, 1), "Tomorrow's prayer times")
	case "next":
		return h.sendNext(ctx, message.Chat.ID)
	case "settings":
		return h.sendSettings(ctx, message.Chat.ID)
	case "method", "madhab", "highlat", "adjust", "remind", "delete_me":
		ok, err := h.canConfigure(ctx, message)
		if err != nil || !ok {
			return err
		}
		switch command {
		case "method":
			return h.setMethod(ctx, message.Chat.ID, argument)
		case "madhab":
			return h.setMadhab(ctx, message.Chat.ID, argument)
		case "highlat":
			return h.setHighLatitude(ctx, message.Chat.ID, argument)
		case "adjust":
			return h.setAdjustment(ctx, message.Chat.ID, argument)
		case "remind":
			return h.setReminders(ctx, message.Chat.ID, argument)
		default:
			return h.deleteChat(ctx, message.Chat.ID)
		}
	case "privacy":
		return h.send(ctx, message.Chat.ID, privacyText, nil)
	case "help":
		return h.send(ctx, message.Chat.ID, helpText, nil)
	case "status":
		if h.ownerID == 0 || message.From == nil || message.From.ID != h.ownerID {
			return nil
		}
		stats, err := h.store.Stats(ctx)
		if err != nil {
			return err
		}
		return h.send(ctx, message.Chat.ID, fmt.Sprintf(
			"Global bot status\nChats: %d\nProfiles: %d\nEnabled reminder rules: %d\nPending schedules: %d",
			stats.Chats, stats.Profiles, stats.EnabledRules, stats.PendingSchedules), nil)
	default:
		return h.send(ctx, message.Chat.ID, "Unknown command. Use /help to see available commands.", nil)
	}
}

func (h *Handler) handleLocation(ctx context.Context, message *models.Message) error {
	ok, err := h.canConfigure(ctx, message)
	if err != nil || !ok {
		return err
	}
	latitude, longitude := message.Location.Latitude, message.Location.Longitude
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return h.send(ctx, message.Chat.ID, "That location is invalid. Please try again.", nil)
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
	return h.send(ctx, message.Chat.ID,
		fmt.Sprintf("Location set to %s (%s). Calculation method: %s. Use /today to see prayer times.", city, resolved.Timezone, profile.Method),
		&models.ReplyKeyboardRemove{RemoveKeyboard: true})
}

func (h *Handler) requestLocation(ctx context.Context, chat models.Chat) error {
	if chat.Type != models.ChatTypePrivate {
		return h.send(ctx, chat.ID, "Telegram only offers the location-request button in private chats. As a group admin, attach a location to a message here to set the group's prayer location.", nil)
	}
	keyboard := &models.ReplyKeyboardMarkup{
		Keyboard:       [][]models.KeyboardButton{{{Text: "Share my location", RequestLocation: true}}},
		ResizeKeyboard: true, OneTimeKeyboard: true,
	}
	return h.send(ctx, chat.ID, "Share your location so I can resolve your timezone and calculate local prayer times. Only rounded coordinates will be stored.", keyboard)
}

func (h *Handler) canConfigure(ctx context.Context, message *models.Message) (bool, error) {
	if message.Chat.Type == models.ChatTypePrivate {
		return true, nil
	}
	if message.From == nil {
		return false, h.send(ctx, message.Chat.ID, "Only a group administrator can change prayer settings.", nil)
	}
	member, err := h.bot.GetChatMember(ctx, &botapi.GetChatMemberParams{ChatID: message.Chat.ID, UserID: message.From.ID})
	if err != nil {
		return false, fmt.Errorf("Telegram administrator check failed")
	}
	if member.Type != models.ChatMemberTypeOwner && member.Type != models.ChatMemberTypeAdministrator {
		return false, h.send(ctx, message.Chat.ID, "Only a group administrator can change prayer settings.", nil)
	}
	return true, nil
}

func (h *Handler) sendSchedule(ctx context.Context, chatID int64, date time.Time, heading string) error {
	profile, err := h.store.Profile(ctx, chatID)
	if store.IsNotFound(err) {
		return h.send(ctx, chatID, "I need a location first. Use /location.", nil)
	}
	if err != nil {
		return err
	}
	schedule, err := h.calculator.Day(ctx, date, profile)
	if err != nil {
		return err
	}
	return h.send(ctx, chatID, formatSchedule(heading, schedule, profile), nil)
}

func (h *Handler) sendNext(ctx context.Context, chatID int64) error {
	profile, err := h.store.Profile(ctx, chatID)
	if store.IsNotFound(err) {
		return h.send(ctx, chatID, "I need a location first. Use /location.", nil)
	}
	if err != nil {
		return err
	}
	now := h.now()
	for day := 0; day < 2; day++ {
		schedule, err := h.calculator.Day(ctx, now.AddDate(0, 0, day), profile)
		if err != nil {
			return err
		}
		for _, prayer := range []domain.Prayer{domain.PrayerFajr, domain.PrayerDhuhr, domain.PrayerAsr, domain.PrayerMaghrib, domain.PrayerIsha} {
			at, ok := schedule.At(prayer)
			if ok && at.After(now) {
				return h.send(ctx, chatID, fmt.Sprintf("Next prayer: %s at %s (%s).", title(prayer), at.Format("15:04"), profile.Timezone), nil)
			}
		}
	}
	return fmt.Errorf("could not find the next prayer")
}

func (h *Handler) sendSettings(ctx context.Context, chatID int64) error {
	profile, err := h.store.Profile(ctx, chatID)
	if store.IsNotFound(err) {
		return h.send(ctx, chatID, "I need a location first. Use /location.", nil)
	}
	if err != nil {
		return err
	}
	text := fmt.Sprintf("Settings\nTimezone: %s\nMethod: %s\nMadhab: %s\nHigh-latitude rule: %s\nAdjustments (minutes): Fajr %+d, Sunrise %+d, Dhuhr %+d, Asr %+d, Maghrib %+d, Isha %+d",
		profile.Timezone, profile.Method, profile.Madhab, profile.HighLatitudeRule,
		profile.Adjustments.Fajr, profile.Adjustments.Sunrise, profile.Adjustments.Dhuhr,
		profile.Adjustments.Asr, profile.Adjustments.Maghrib, profile.Adjustments.Isha)
	return h.send(ctx, chatID, text, nil)
}

func (h *Handler) setMethod(ctx context.Context, chatID int64, argument string) error {
	method := domain.Method(strings.ToLower(argument))
	if !method.Valid() {
		values := make([]string, 0, len(domain.SupportedMethods()))
		for _, supported := range domain.SupportedMethods() {
			values = append(values, string(supported))
		}
		return h.send(ctx, chatID, "Usage: /method <"+strings.Join(values, "|")+">", nil)
	}
	return h.updateProfile(ctx, chatID, func(profile *domain.PrayerProfile) { profile.Method = method }, "Calculation method updated to "+string(method)+".")
}

func (h *Handler) setMadhab(ctx context.Context, chatID int64, argument string) error {
	madhab := domain.Madhab(strings.ToLower(argument))
	if !madhab.Valid() {
		return h.send(ctx, chatID, "Usage: /madhab <shafii|hanafi>", nil)
	}
	return h.updateProfile(ctx, chatID, func(profile *domain.PrayerProfile) { profile.Madhab = madhab }, "Madhab updated to "+string(madhab)+".")
}

func (h *Handler) setHighLatitude(ctx context.Context, chatID int64, argument string) error {
	rule := domain.HighLatitudeRule(strings.ToLower(argument))
	if !rule.Valid() {
		return h.send(ctx, chatID, "Usage: /highlat <angle_based|middle_of_night|one_seventh>", nil)
	}
	return h.updateProfile(ctx, chatID, func(profile *domain.PrayerProfile) { profile.HighLatitudeRule = rule }, "High-latitude rule updated to "+string(rule)+".")
}

func (h *Handler) setAdjustment(ctx context.Context, chatID int64, argument string) error {
	fields := strings.Fields(argument)
	if len(fields) != 2 {
		return h.send(ctx, chatID, "Usage: /adjust <fajr|sunrise|dhuhr|asr|maghrib|isha> <minutes from -30 to 30>", nil)
	}
	minutes, err := strconv.Atoi(fields[1])
	if err != nil || minutes < -30 || minutes > 30 {
		return h.send(ctx, chatID, "Adjustment must be a whole number from -30 to 30.", nil)
	}
	prayer := domain.Prayer(strings.ToLower(fields[0]))
	if !prayer.Valid() {
		return h.send(ctx, chatID, "Unknown prayer name.", nil)
	}
	return h.updateProfile(ctx, chatID, func(profile *domain.PrayerProfile) {
		switch prayer {
		case domain.PrayerFajr:
			profile.Adjustments.Fajr = minutes
		case domain.PrayerSunrise:
			profile.Adjustments.Sunrise = minutes
		case domain.PrayerDhuhr:
			profile.Adjustments.Dhuhr = minutes
		case domain.PrayerAsr:
			profile.Adjustments.Asr = minutes
		case domain.PrayerMaghrib:
			profile.Adjustments.Maghrib = minutes
		case domain.PrayerIsha:
			profile.Adjustments.Isha = minutes
		}
	}, fmt.Sprintf("%s adjustment updated to %+d minutes.", title(prayer), minutes))
}

func (h *Handler) updateProfile(ctx context.Context, chatID int64, update func(*domain.PrayerProfile), confirmation string) error {
	profile, err := h.store.Profile(ctx, chatID)
	if store.IsNotFound(err) {
		return h.send(ctx, chatID, "I need a location first. Use /location.", nil)
	}
	if err != nil {
		return err
	}
	update(&profile)
	if _, err := h.store.UpsertProfile(ctx, profile); err != nil {
		return err
	}
	if err := h.planner.RebuildChat(ctx, chatID, h.now()); err != nil {
		return err
	}
	return h.send(ctx, chatID, confirmation, nil)
}

func (h *Handler) setReminders(ctx context.Context, chatID int64, argument string) error {
	switch strings.ToLower(argument) {
	case "on":
		if _, err := h.store.Profile(ctx, chatID); store.IsNotFound(err) {
			return h.send(ctx, chatID, "I need a location first. Use /location.", nil)
		} else if err != nil {
			return err
		}
		if err := h.store.EnableDefaultRules(ctx, chatID); err != nil {
			return err
		}
		if err := h.planner.RebuildChat(ctx, chatID, h.now()); err != nil {
			return err
		}
		return h.send(ctx, chatID, "Prayer-time reminders are enabled.", nil)
	case "off":
		if err := h.store.DisableRules(ctx, chatID); err != nil {
			return err
		}
		return h.send(ctx, chatID, "Prayer-time reminders are disabled.", nil)
	default:
		return h.send(ctx, chatID, "Usage: /remind <on|off>", nil)
	}
}

func (h *Handler) deleteChat(ctx context.Context, chatID int64) error {
	if err := h.store.DeleteChat(ctx, chatID); err != nil {
		return err
	}
	return h.send(ctx, chatID, "Your saved location, settings, reminders, and delivery history have been deleted.", nil)
}

func (h *Handler) send(ctx context.Context, chatID int64, text string, markup models.ReplyMarkup) error {
	_, err := h.bot.SendMessage(ctx, &botapi.SendMessageParams{ChatID: chatID, Text: text, ReplyMarkup: markup})
	if err != nil {
		return fmt.Errorf("Telegram send failed")
	}
	return nil
}

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

func formatSchedule(heading string, schedule domain.DaySchedule, profile domain.PrayerProfile) string {
	var builder strings.Builder
	builder.WriteString(heading)
	builder.WriteString(" — ")
	builder.WriteString(schedule.Date.Format("2 January 2006"))
	for _, prayer := range []domain.Prayer{domain.PrayerFajr, domain.PrayerSunrise, domain.PrayerDhuhr, domain.PrayerAsr, domain.PrayerMaghrib, domain.PrayerIsha} {
		if at, ok := schedule.At(prayer); ok {
			fmt.Fprintf(&builder, "\n%s: %s", title(prayer), at.Format("15:04"))
		}
	}
	fmt.Fprintf(&builder, "\n%s · %s", profile.Timezone, profile.Method)
	return builder.String()
}

func title(prayer domain.Prayer) string {
	switch prayer {
	case domain.PrayerFajr:
		return "Fajr"
	case domain.PrayerSunrise:
		return "Sunrise"
	case domain.PrayerDhuhr:
		return "Dhuhr"
	case domain.PrayerAsr:
		return "Asr"
	case domain.PrayerMaghrib:
		return "Maghrib"
	case domain.PrayerIsha:
		return "Isha"
	default:
		return string(prayer)
	}
}

const helpText = `Commands
/location — set or replace the location
/today, /tomorrow, /next — prayer times
/settings — current calculation settings
/method <name> — calculation convention
/madhab <shafii|hanafi> — Asr convention
/highlat <angle_based|middle_of_night|one_seventh>
/adjust <prayer> <minutes> — per-prayer correction
/remind <on|off> — prayer-time reminders
/privacy — stored data and deletion
/delete_me — delete this chat's global-bot data`

const privacyText = `Privacy
The bot uses a shared location only to resolve its timezone and an approximate place. It stores coordinates rounded to three decimal places, the timezone, Google Place ID, and your calculation/reminder settings. It does not store Google's formatted address or the full Telegram update. Use /delete_me to remove this chat's data.`
