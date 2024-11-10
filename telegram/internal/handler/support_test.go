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
		wantFullName  string
		ok            bool
	}{
		{
			name:          "valid_1",
			message:       "User ID: 12345678\n Full Name: John Doe\n Message ID: 12345678",
			wantChatID:    12345678,
			wantMessageID: 12345678,
			wantFullName:  "John Doe",
			ok:            true,
		},
		{
			name:          "valid_2",
			message:       "User ID: 127928\n Full Name: Johnny Doe\n  Message ID: 123123",
			wantChatID:    127928,
			wantMessageID: 123123,
			wantFullName:  "Johnny Doe",
			ok:            true,
		},
		{
			name:          "valid_3",
			message:       "User ID: 78381324\n Full Name: Xayah Doe\n  Message ID: 1235341",
			wantChatID:    78381324,
			wantMessageID: 1235341,
			wantFullName:  "Xayah Doe",
			ok:            true,
		},
		{
			name:          "no_chatID",
			message:       "Full Name: John Doe\n Message ID: 12345678",
			wantChatID:    0,
			wantMessageID: 12345678,
			wantFullName:  "John Doe",
			ok:            false,
		},
		{
			name:          "no_messageID",
			message:       "User ID: 12345678\n Full Name: John Doe",
			wantChatID:    12345678,
			wantMessageID: 0,
			wantFullName:  "John Doe",
			ok:            false,
		},
		{
			name:          "no_full_name",
			message:       "User ID: 12345678\n Message ID: 123423678",
			wantChatID:    12345678,
			wantMessageID: 123423678,
			wantFullName:  "",
			ok:            false,
		},
		{
			name:          "no_chatID_no_messageID",
			message:       "Full Name: John Doe",
			wantChatID:    0,
			wantMessageID: 0,
			wantFullName:  "John Doe",
			ok:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			chatID, messageID, fullName, ok := parseUserMessage(tt.message)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.wantChatID, chatID)
			require.Equal(t, tt.wantMessageID, messageID)
			require.Equal(t, tt.wantFullName, fullName)
		})
	}
}
