package internal

import (
	"context"
	"fmt"

	"github.com/escalopa/prayer-bot/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	stageSplitter = "-"
)

type chatState string

const (
	ChatStateDefault chatState = "default"

	// user state

	ChatStateBug      chatState = "bug"
	ChatStateFeedback chatState = "feedback"

	// admin state

	ChatStateReply    chatState = "reply"
	ChatStateAnnounce chatState = "announce"
)

type Command string

const (
	// user commands

	StartCommand       Command = "/start"
	HelpCommand        Command = "/help"
	TodayCommand       Command = "/today"
	DateCommand        Command = "/date"     // 2 stages
	NotifyCommand      Command = "/notify"   // 1 stage
	BugCommand         Command = "/bug"      // 1 stage
	FeedbackCommand    Command = "/feedback" // 1 stage
	LanguageCommand    Command = "/language" // 1 stage
	SubscribeCommand   Command = "/subscribe"
	UnsubscribeCommand Command = "/unsubscribe"

	// admin commands

	AdminCommand    Command = "/admin"
	ReplyCommand    Command = "/reply"
	StatsCommand    Command = "/stats"
	AnnounceCommand Command = "/announce"

	// other commands

	CancelCommand Command = "/cancel"
)

func IsValidCommand(cmd string) bool {
	for _, command := range []Command{
		StartCommand,
		HelpCommand,
		TodayCommand,
		DateCommand,
		NotifyCommand,
		SubscribeCommand,
		UnsubscribeCommand,
		LanguageCommand,
		FeedbackCommand,
		BugCommand,
		AdminCommand,
		ReplyCommand,
		StatsCommand,
		AnnounceCommand,
		CancelCommand,
	} {
		if cmd == string(command) {
			return true
		}
	}
	return false
}

func (h *Handler) notifyBot(_ context.Context, _ *bot.Bot, info *domain.NotifierPayload) error {
	// TODO: implement
	fmt.Printf("Notify bot ID: %d info: %+v\n", info.BotID, info)
	return nil
}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	botID := getContextBotID(ctx)

	text := "Hello, world!"
	if update.Message != nil {
		text = update.Message.Text
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   text,
	})
	if err != nil {
		fmt.Printf("send message: bot_id: %d %v\n", botID, err)
	}
}
