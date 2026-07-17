package telegram

import (
	"context"
	"strings"
	"testing"

	botapi "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/escalopa/prayer-bot/global/internal/i18n"
)

func TestFeedbackReplyRequiresPrivateReplyToBotPrompt(t *testing.T) {
	prompt := i18n.Resolve("ar").Message("feedback_prompt")
	message := &models.Message{
		Chat:           models.Chat{ID: 42, Type: models.ChatTypePrivate},
		ReplyToMessage: &models.Message{Text: prompt, From: &models.User{IsBot: true}},
	}
	if !isFeedbackReply(message) {
		t.Fatal("expected a reply to a localized bot prompt to be recognized")
	}
	message.ReplyToMessage.From.IsBot = false
	if isFeedbackReply(message) {
		t.Fatal("a user-authored lookalike prompt must not be accepted")
	}
}

func TestDeliverFeedbackSendsPrivateContextAndCopiesOriginal(t *testing.T) {
	sender := &fakeFeedbackSender{}
	message := &models.Message{
		ID: 77, Chat: models.Chat{ID: 42, Type: models.ChatTypePrivate}, Text: "The date is wrong",
		From: &models.User{ID: 99, FirstName: "Amina <Admin>", Username: "amina"},
	}
	if err := deliverFeedback(context.Background(), sender, 1234, message, "ar"); err != nil {
		t.Fatal(err)
	}
	if sender.sent == nil || sender.sent.ChatID != int64(1234) || !strings.Contains(sender.sent.Text, "Amina &lt;Admin&gt;") || !strings.Contains(sender.sent.Text, "<code>99</code>") {
		t.Fatalf("unexpected owner notification: %+v", sender.sent)
	}
	if sender.copied == nil || sender.copied.ChatID != int64(1234) || sender.copied.FromChatID != int64(42) || sender.copied.MessageID != 77 {
		t.Fatalf("unexpected copied feedback: %+v", sender.copied)
	}
	if sender.copied.ReplyParameters == nil || sender.copied.ReplyParameters.MessageID != 500 {
		t.Fatalf("feedback copy is not attached to its context: %+v", sender.copied.ReplyParameters)
	}
}

type fakeFeedbackSender struct {
	sent   *botapi.SendMessageParams
	copied *botapi.CopyMessageParams
}

func (f *fakeFeedbackSender) SendMessage(_ context.Context, params *botapi.SendMessageParams) (*models.Message, error) {
	f.sent = params
	return &models.Message{ID: 500}, nil
}

func (f *fakeFeedbackSender) CopyMessage(_ context.Context, params *botapi.CopyMessageParams) (*models.MessageID, error) {
	f.copied = params
	return &models.MessageID{ID: 501}, nil
}

var _ feedbackSender = (*fakeFeedbackSender)(nil)
