// Package botprofile applies Telegram Bot API profile fields (name, descriptions, commands).
// Run at deploy time via cmd/syncprofile, not from the webhook request path.
package botprofile

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func isRateLimited(err error) bool {
	if err == nil {
		return false
	}
	if bot.IsTooManyRequestsError(err) {
		return true
	}
	var rateErr *bot.TooManyRequestsError
	return errors.As(err, &rateErr)
}

func wrapProfileErr(step string, err error) error {
	if err == nil || isRateLimited(err) {
		return nil
	}
	return fmt.Errorf("%s: %w", step, err)
}

func commandsEqual(a, b []models.BotCommand) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Command != b[i].Command || a[i].Description != b[i].Description {
			return false
		}
	}
	return true
}

func syncMyName(ctx context.Context, b *bot.Bot, desired string) error {
	current, err := b.GetMyName(ctx, &bot.GetMyNameParams{})
	if err != nil {
		return wrapProfileErr("getMyName", err)
	}
	if current.Name == desired {
		return nil
	}
	if _, err := b.SetMyName(ctx, &bot.SetMyNameParams{Name: desired}); err != nil {
		return wrapProfileErr("setMyName", err)
	}
	return nil
}

func syncMyShortDescription(ctx context.Context, b *bot.Bot, desired string) error {
	current, err := b.GetMyShortDescription(ctx, &bot.GetMyShortDescriptionParams{})
	if err != nil {
		return wrapProfileErr("getMyShortDescription", err)
	}
	if current.ShortDescription == desired {
		return nil
	}
	if _, err := b.SetMyShortDescription(ctx, &bot.SetMyShortDescriptionParams{ShortDescription: desired}); err != nil {
		return wrapProfileErr("setMyShortDescription", err)
	}
	return nil
}

func syncMyDescription(ctx context.Context, b *bot.Bot, desired string) error {
	current, err := b.GetMyDescription(ctx, &bot.GetMyDescriptionParams{})
	if err != nil {
		return wrapProfileErr("getMyDescription", err)
	}
	if current.Description == desired {
		return nil
	}
	if _, err := b.SetMyDescription(ctx, &bot.SetMyDescriptionParams{Description: desired}); err != nil {
		return wrapProfileErr("setMyDescription", err)
	}
	return nil
}

func syncMyCommands(ctx context.Context, b *bot.Bot, desired []models.BotCommand, scope models.BotCommandScope, label string) error {
	current, err := b.GetMyCommands(ctx, &bot.GetMyCommandsParams{Scope: scope})
	if err != nil {
		return wrapProfileErr("getMyCommands ("+label+")", err)
	}
	if commandsEqual(current, desired) {
		return nil
	}
	if _, err := b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: desired,
		Scope:    scope,
	}); err != nil {
		return wrapProfileErr("setMyCommands ("+label+")", err)
	}
	return nil
}

// PublicAbout is prepended to /help in the dispatcher handler; keep in sync with Telegram descriptions.
const PublicAbout = "🕌 Daily prayer times\n⏰ Prayer notifications\n🌍 Multiple languages supported"

func cityFromUsername(username string) string {
	base := username
	if i := strings.IndexByte(base, '_'); i >= 0 {
		base = base[:i]
	}
	if base == "" {
		return "Prayer"
	}

	runes := []rune(base)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func profilePrefix(username string) string {
	if strings.HasSuffix(username, "test_bot") {
		return "[TEST] "
	}
	return ""
}

// Sync updates Telegram profile fields only when they differ from the desired values.
// Command strings must match serverless/dispatcher/internal/handler/command.go.
func Sync(ctx context.Context, b *bot.Bot, ownerID int64) error {
	me, err := b.GetMe(ctx)
	if err != nil {
		return fmt.Errorf("getMe: %w", err)
	}

	city := cityFromUsername(me.Username)
	prefix := profilePrefix(me.Username)

	if ownerID == 0 {
		return fmt.Errorf("owner id is required")
	}

	desiredName := fmt.Sprintf("%s%s Prayer Times", prefix, city)
	if err := syncMyName(ctx, b, desiredName); err != nil {
		return err
	}

	about := fmt.Sprintf("%s\n📍 %s", PublicAbout, city)
	if err := syncMyShortDescription(ctx, b, about); err != nil {
		return err
	}
	if err := syncMyDescription(ctx, b, about); err != nil {
		return err
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

	if err := syncMyCommands(ctx, b, userCommands, &models.BotCommandScopeDefault{}, "default"); err != nil {
		return err
	}
	if err := syncMyCommands(ctx, b, allCommands, &models.BotCommandScopeChat{ChatID: ownerID}, "owner chat"); err != nil {
		return err
	}

	return nil
}
