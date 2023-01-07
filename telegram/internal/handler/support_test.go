package handler

import "testing"

func TestSupport_parseUserMessage(t *testing.T) {
	tests := []struct {
		name          string
		message       string
		wantUserID    int
		wantMessageID int
		wantFullName  string
		ok            bool
	}{
		{
			name:          "valid message 1",
			message:       "User ID: 12345678\n Full Name: John Doe\n Message ID: 12345678",
			wantUserID:    12345678,
			wantMessageID: 12345678,
			wantFullName:  "John Doe",
			ok:            true,
		},
		{
			name:          "valid message 2",
			message:       "User ID: 127928\n Full Name: Johnny Doe\n  Message ID: 123123",
			wantUserID:    127928,
			wantMessageID: 123123,
			wantFullName:  "Johnny Doe",
			ok:            true,
		},
		{
			name:          "valid message 3",
			message:       "User ID: 78381324\n Full Name: Xayah Doe\n  Message ID: 1235341",
			wantUserID:    78381324,
			wantMessageID: 1235341,
			wantFullName:  "Xayah Doe",
			ok:            true,
		},
		{
			name:          "invalid message, no user ID",
			message:       "Full Name: John Doe\n Message ID: 12345678",
			wantUserID:    0,
			wantMessageID: 12345678,
			wantFullName:  "John Doe",
			ok:            false,
		},
		{
			name:          "invalid message, no message ID",
			message:       "User ID: 12345678\n Full Name: John Doe",
			wantUserID:    12345678,
			wantMessageID: 0,
			wantFullName:  "John Doe",
			ok:            false,
		},
		{
			name:          "invalid message, no full name",
			message:       "User ID: 12345678\n Message ID: 123423678",
			wantUserID:    12345678,
			wantMessageID: 123423678,
			wantFullName:  "",
			ok:            false,
		},
		{
			name:          "invalid message, no user ID, no message ID",
			message:       "Full Name: John Doe",
			wantUserID:    0,
			wantMessageID: 0,
			wantFullName:  "John Doe",
			ok:            false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, messageID, fullName, ok := parseUserMessage(tt.message)
			if ok != tt.ok {
				t.Errorf("parseUserMessage() error = %v, ok %v", ok, tt.ok)
				return
			}
			if userID != tt.wantUserID {
				t.Errorf("parseUserMessage() userID = %v, want %v", userID, tt.wantUserID)
			}
			if messageID != tt.wantMessageID {
				t.Errorf("parseUserMessage() messageID = %v, want %v", messageID, tt.wantMessageID)
			}
			if fullName != tt.wantFullName {
				t.Errorf("parseUserMessage() fullName = %v, want %v", fullName, tt.wantFullName)
			}
		})
	}
}
