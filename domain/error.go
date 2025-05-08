package domain

import (
	"encoding/json"
	"log/slog"
)

type (
	Error struct {
		internal struct {
			BotID   int64  `json:"bot_id,omitempty"`
			ChatID  int64  `json:"chat_id,omitempty"`
			Method  string `json:"method,omitempty"`
			Message string `json:"message,omitempty"`
			Err     error  `json:"err,omitempty"`
		}
	}
)

func NewError(err error) *Error {
	e := &Error{}
	e.internal.Err = err
	return e
}

func (e *Error) BotID(botID int64) *Error {
	e.internal.BotID = botID
	return e
}

func (e *Error) ChatID(chatID int64) *Error {
	e.internal.ChatID = chatID
	return e
}

func (e *Error) Method(method string) *Error {
	e.internal.Method = method
	return e

}

func (e *Error) Message(message string) *Error {
	e.internal.Message = message
	return e
}

func (e *Error) Error() string {
	slog.With()
	b, _ := json.Marshal(e.internal)
	return string(b)
}
