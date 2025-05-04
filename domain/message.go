package domain

import (
	"encoding/json"
	"fmt"
)

type PayloadType string

const (
	PayloadTypeHandler  PayloadType = "handler"
	PayloadTypeNotifier PayloadType = "notifier"
)

type (
	BotConfig struct {
		BotID    int32  `json:"bot_id"`
		Token    string `json:"token"`
		Secret   string `json:"secret"`
		Location string `json:"location"`
	}

	// NotifierPayload sent by `notifier-fn` to notify users about prayer times
	NotifierPayload struct {
		BotID        int32        `json:"bot_id"`
		ChatIDs      []int64      `json:"chat_ids"`
		PrayerID     PrayerID     `json:"prayer_id"`
		NotifyOffset NotifyOffset `json:"notify_offset"`
	}

	// HandlerPayload sent by `handler-fn` to handle incoming messages from the user
	HandlerPayload struct {
		BotID int32       `json:"bot_id"`
		Data  interface{} `json:"data"` // `*models.Update` (type is hidden explicitly to prevent extra import)
	}

	// Payload main struct that is sent to the `queue` for process
	Payload struct {
		Type PayloadType `json:"type"` // one of [`Handler`, `Notifier`]
		Data interface{} `json:"data"` // one of [`HandlerPayload`, `NotifierPayload`]
	}
)

func (p *Payload) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Payload) Unmarshal(data []byte) error {
	return json.Unmarshal(data, p)
}

func Unmarshal[T any](data interface{}) (*T, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var t T
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, fmt.Errorf("unmarshal: %T got: %T: %v", t, data, err)
	}

	return &t, nil
}
