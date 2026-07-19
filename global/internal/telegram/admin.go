package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-telegram/bot/models"

	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
	"github.com/escalopa/prayer-bot/global/internal/store"
)

type adminView string

const (
	adminViewOverview  adminView = "overview"
	adminViewActivity  adminView = "activity"
	adminViewLanguages adminView = "languages"
	adminViewMethods   adminView = "methods"
	adminViewReminders adminView = "reminders"
	adminViewHealth    adminView = "health"
	adminViewFeedback  adminView = "feedback"
)

func (h *Handler) isOwner(chat models.Chat, user *models.User) bool {
	return h.ownerID != 0 &&
		chat.Type == models.ChatTypePrivate &&
		user != nil &&
		user.ID == h.ownerID
}

func (h *Handler) sendAdminDashboard(ctx context.Context, chatID int64, view adminView) error {
	metrics, err := h.store.AdminMetrics(ctx)
	if err != nil {
		return fmt.Errorf("load owner dashboard: %w", err)
	}
	return h.send(ctx, chatID, formatAdminDashboard(metrics, view, h.now()), adminKeyboard(view))
}

func (h *Handler) editAdminDashboard(ctx context.Context, message *models.Message, view adminView) error {
	metrics, err := h.store.AdminMetrics(ctx)
	if err != nil {
		return fmt.Errorf("load owner dashboard: %w", err)
	}
	return h.edit(ctx, message.Chat.ID, message.ID, formatAdminDashboard(metrics, view, h.now()), adminKeyboard(view))
}

func parseAdminView(data string) (adminView, bool) {
	view := adminView(strings.TrimPrefix(data, "admin:"))
	switch view {
	case adminViewOverview, adminViewActivity, adminViewLanguages, adminViewMethods, adminViewReminders, adminViewHealth, adminViewFeedback:
		return view, true
	default:
		return "", false
	}
}

func adminKeyboard(current adminView) *models.InlineKeyboardMarkup {
	button := func(label string, view adminView) models.InlineKeyboardButton {
		return callbackButton(selectedLabel(label, current == view), "admin:"+string(view))
	}
	return inlineKeyboard(
		[]models.InlineKeyboardButton{
			button("📊 Overview", adminViewOverview),
			button("⚡ Activity", adminViewActivity),
		},
		[]models.InlineKeyboardButton{
			button("🌐 Languages", adminViewLanguages),
			button("🧭 Methods", adminViewMethods),
		},
		[]models.InlineKeyboardButton{
			button("🔔 Reminders", adminViewReminders),
			button("🩺 Delivery health", adminViewHealth),
		},
		[]models.InlineKeyboardButton{
			button("💬 Feedback help", adminViewFeedback),
			{Text: "🔄 Refresh", CallbackData: "admin:" + string(current)},
		},
	)
}

func formatAdminDashboard(metrics store.AdminDashboard, view adminView, now time.Time) string {
	var body string
	switch view {
	case adminViewActivity:
		body = formatAdminActivity(metrics)
	case adminViewLanguages:
		body = formatAdminLanguages(metrics)
	case adminViewMethods:
		body = formatAdminMethods(metrics)
	case adminViewReminders:
		body = formatAdminReminders(metrics)
	case adminViewHealth:
		body = formatAdminHealth(metrics)
	case adminViewFeedback:
		body = formatAdminFeedback()
	default:
		body = formatAdminOverview(metrics)
	}
	return body + fmt.Sprintf(
		"\n\n<i>Aggregate data only · refreshed %s UTC</i>",
		now.UTC().Format("02 Jan 2006 15:04"),
	)
}

func formatAdminOverview(metrics store.AdminDashboard) string {
	return fmt.Sprintf(
		"<b>Owner dashboard</b> 🔐\n\n"+
			"👤 <b>Users</b>: %d\n"+
			"👥 <b>Groups</b>: %d\n"+
			"📍 <b>Configured</b>: %d · %.1f%%\n"+
			"🔔 <b>Using reminders</b>: %d · %.1f%%\n\n"+
			"⚡ <b>Active users</b>\n"+
			"24 hours: %d\n"+
			"7 days: %d\n"+
			"30 days: %d\n\n"+
			"🌱 <b>New users</b>\n"+
			"24 hours: %d\n"+
			"7 days: %d\n"+
			"30 days: %d",
		metrics.Users,
		metrics.Groups,
		metrics.ConfiguredUsers,
		percentage(metrics.ConfiguredUsers, metrics.Users),
		metrics.ReminderUsers,
		percentage(metrics.ReminderUsers, metrics.Users),
		metrics.ActiveUsers24Hours,
		metrics.ActiveUsers7Days,
		metrics.ActiveUsers30Days,
		metrics.NewUsers24Hours,
		metrics.NewUsers7Days,
		metrics.NewUsers30Days,
	)
}

func formatAdminActivity(metrics store.AdminDashboard) string {
	return fmt.Sprintf(
		"<b>User activity</b> ⚡\n\n"+
			"<b>Active private chats</b>\n"+
			"24 hours: %d · %.1f%%\n"+
			"7 days: %d · %.1f%%\n"+
			"30 days: %d · %.1f%%\n\n"+
			"<b>New private chats</b>\n"+
			"24 hours: %d\n"+
			"7 days: %d\n"+
			"30 days: %d\n\n"+
			"<i>Activity means the user interacted with the bot during the period.</i>",
		metrics.ActiveUsers24Hours,
		percentage(metrics.ActiveUsers24Hours, metrics.Users),
		metrics.ActiveUsers7Days,
		percentage(metrics.ActiveUsers7Days, metrics.Users),
		metrics.ActiveUsers30Days,
		percentage(metrics.ActiveUsers30Days, metrics.Users),
		metrics.NewUsers24Hours,
		metrics.NewUsers7Days,
		metrics.NewUsers30Days,
	)
}

func formatAdminLanguages(metrics store.AdminDashboard) string {
	var builder strings.Builder
	builder.WriteString("<b>User languages</b> 🌐\n")
	if len(metrics.Languages) == 0 {
		builder.WriteString("\nNo users yet.")
		return builder.String()
	}
	for _, metric := range metrics.Languages {
		locale := i18n.Resolve(metric.Key)
		fmt.Fprintf(&builder, "\n%s · <b>%d</b> · %.1f%%",
			escape(locale.NativeName), metric.Count, percentage(metric.Count, metrics.Users))
	}
	return builder.String()
}

func formatAdminMethods(metrics store.AdminDashboard) string {
	var builder strings.Builder
	builder.WriteString("<b>Calculation methods</b> 🧭\n")
	if len(metrics.Methods) == 0 {
		builder.WriteString("\nNo configured users yet.")
		return builder.String()
	}
	english := i18n.Resolve("en")
	for _, metric := range metrics.Methods {
		method := domain.Method(metric.Key)
		label := metric.Key
		if method.Valid() {
			label = english.Method(method)
		}
		fmt.Fprintf(&builder, "\n%s · <b>%d</b> · %.1f%%",
			escape(label), metric.Count, percentage(metric.Count, metrics.ConfiguredUsers))
	}
	return builder.String()
}

func formatAdminReminders(metrics store.AdminDashboard) string {
	counts := make(map[string]int64, len(metrics.ReminderKinds))
	for _, metric := range metrics.ReminderKinds {
		counts[metric.Key] = metric.Count
	}
	return fmt.Sprintf(
		"<b>Reminder adoption</b> 🔔\n\n"+
			"🕌 Prayer times: <b>%d</b>\n"+
			"🌙 Monday &amp; Thursday fasting: <b>%d</b>\n"+
			"📖 Friday Al-Kahf: <b>%d</b>\n"+
			"🕋 Major Islamic occasions: <b>%d</b>\n"+
			"🤲 Special fasting days: <b>%d</b>\n"+
			"🌙 Commonly observed dates: <b>%d</b>\n\n"+
			"Users with any reminder: %d · %.1f%%\n"+
			"Enabled rules: %d\n"+
			"Pending schedules: %d",
		counts["prayer"],
		counts["fasting"],
		counts["kahf"],
		counts["occasion_major"],
		counts["occasion_fasting"],
		counts["occasion_observed"],
		metrics.ReminderUsers,
		percentage(metrics.ReminderUsers, metrics.Users),
		metrics.EnabledRules,
		metrics.PendingSchedules,
	)
}

func formatAdminHealth(metrics store.AdminDashboard) string {
	totalDeliveries := metrics.SentDeliveries24Hours +
		metrics.FailedDeliveries24Hours +
		metrics.StaleDeliveries24Hours
	return fmt.Sprintf(
		"<b>Delivery health</b> 🩺\n\n"+
			"<b>Last 24 hours</b>\n"+
			"✅ Sent: %d\n"+
			"❌ Failed: %d\n"+
			"⏭ Stale: %d\n"+
			"Success rate: %.1f%%\n\n"+
			"<b>Current queues</b>\n"+
			"Processing deliveries: %d\n"+
			"Task outbox: %d\n"+
			"Pending schedules: %d\n\n"+
			"⚠️ Failed bot updates (24h): %d",
		metrics.SentDeliveries24Hours,
		metrics.FailedDeliveries24Hours,
		metrics.StaleDeliveries24Hours,
		percentage(metrics.SentDeliveries24Hours, totalDeliveries),
		metrics.ProcessingDeliveries,
		metrics.QueuedTasks,
		metrics.PendingSchedules,
		metrics.FailedUpdates24Hours,
	)
}

func formatAdminFeedback() string {
	return "<b>Feedback workflow</b> 💬\n\n" +
		"1. The bot sends you the user's identity and language.\n" +
		"2. The next message is a copy of their text or screenshot.\n" +
		"3. Tap <b>✉️ Contact user</b> to open their Telegram profile and answer directly.\n\n" +
		"⚠️ Replying to either message inside this bot chat does not forward your response to the user."
}

func percentage(value, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(value) * 100 / float64(total)
}
