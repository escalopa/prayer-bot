package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupport_parseUserMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		message       string
		wantChatID    int
		wantMessageID int
		ok            bool
	}{
		{
			name:          "valid_1",
			message:       "ChatID: 12345678\nMessageID: 12345678",
			wantChatID:    12345678,
			wantMessageID: 12345678,
			ok:            true,
		},
		{
			name:          "no_chatID",
			message:       "MessageID: 12345678",
			wantChatID:    0,
			wantMessageID: 12345678,
			ok:            false,
		},
		{
			name:          "no_messageID",
			message:       "ChatID: 12345678",
			wantChatID:    12345678,
			wantMessageID: 0,
			ok:            false,
		},
		{
			name:          "no_chatID_no_messageID",
			message:       "Hello, world!",
			wantChatID:    0,
			wantMessageID: 0,
			ok:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			chatID, messageID, ok := parseUserMessage(tt.message)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.wantChatID, chatID)
			require.Equal(t, tt.wantMessageID, messageID)
		})
	}
}
