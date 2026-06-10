// Package botprofile applies Telegram Bot API profile fields (name, descriptions, commands).
// Run at deploy time via cmd/syncprofile, not from the webhook request path.
package botprofile

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// PublicAbout is prepended to /help in the dispatcher handler; keep in sync with Telegram descriptions.
const PublicAbout = "🕌 Daily prayer times\n⏰ Prayer notifications\n🌍 Multiple languages supported"

const profileName = "Prayer Times"

// Sync calls setMyName, setMyShortDescription, setMyDescription, and setMyCommands.
// Command strings must match serverless/dispatcher/internal/handler/command.go.
func Sync(ctx context.Context, b *bot.Bot) error {
	if _, err := b.SetMyName(ctx, &bot.SetMyNameParams{Name: profileName}); err != nil {
		return fmt.Errorf("setMyName: %w", err)
	}
	if _, err := b.SetMyShortDescription(ctx, &bot.SetMyShortDescriptionParams{ShortDescription: PublicAbout}); err != nil {
		return fmt.Errorf("setMyShortDescription: %w", err)
	}
	if _, err := b.SetMyDescription(ctx, &bot.SetMyDescriptionParams{Description: PublicAbout}); err != nil {
		return fmt.Errorf("setMyDescription: %w", err)
	}

	userCommands := []models.BotCommand{
		{Command: "start", Description: "Open the bot and show the guide"},
		{Command: "help", Description: "Show commands and tips"},
		{Command: "today", Description: "Today's prayer times"},
		{Command: "tomorrow", Description: "Tomorrow's prayer times"},
		{Command: "date", Description: "Pick a date on the calendar"},
		{Command: "next", Description: "Time until the next prayer"},
		{Command: "remind", Description: "Reminder and jamaat settings"},
		{Command: "language", Description: "Choose bot language"},
		{Command: "bug", Description: "Report a bug"},
		{Command: "feedback", Description: "Send feedback"},
		{Command: "cancel", Description: "Cancel the current step"},
	}
	adminCommands := []models.BotCommand{
		{Command: "admin", Description: "Admin overview (bot owner only)"},
		{Command: "info", Description: "Chat info and settings (owner only)"},
		{Command: "reply", Description: "Reply to a user by chat id (owner only)"},
		{Command: "stats", Description: "Usage statistics (owner only)"},
		{Command: "announce", Description: "Broadcast to all chats (owner only)"},
	}
	allCommands := append(append([]models.BotCommand{}, userCommands...), adminCommands...)

	if _, err := b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: allCommands,
		Scope:    &models.BotCommandScopeDefault{},
	}); err != nil {
		return fmt.Errorf("setMyCommands: %w", err)
	}

	if _, err := b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: adminCommands,
		Scope:    &models.BotCommandScopeAllChatAdministrators{},
	}); err != nil {
		return fmt.Errorf("setMyCommands (group admins): %w", err)
	}

	return nil
}
