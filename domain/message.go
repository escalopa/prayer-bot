package domain

import "encoding/json"

type delay int

const (
	Delay0m  delay = 0
	Delay5m  delay = 5
	Delay10m delay = 10
	Delay15m delay = 15
	Delay20m delay = 20
	Delay30m delay = 30
	Delay45m delay = 45
	Delay60m delay = 60
	Delay90m delay = 90
)

type payloadType string

const (
	PayloadTypeHandel payloadType = "handle"
	PayloadTypeNotify payloadType = "notify"
)

type (
	BotConfig struct {
		BotID    uint8  `json:"bot_id"`
		Token    string `json:"token"`
		Secret   string `json:"secret"`
		Location string `json:"location"`
	}

	NotifyBot struct {
		BotID    uint8   `json:"bot_id"`
		ChatIDs  []int64 `json:"chat_ids"`
		PrayerID uint8   `json:"prayer_id"`
		Delay    delay   `json:"delay"`
	}

	// NotifyPayload sent by `notifier-fn` to notify users about prayer times
	NotifyPayload struct {
		Data []NotifyBot `json:"data"`
	}

	// HandlePayload sent by `handler-fn` to handle incoming messages from the user
	HandlePayload struct {
		BotID uint8       `json:"bot_id"`
		Data  interface{} `json:"data"` // `models.Update` (type is hidden implicitly to prevent extra import)
	}

	// Payload main struct that is sent to the `queue` for process
	Payload struct {
		Type payloadType `json:"type"`
		Data interface{} `json:"data"` // one of [`HandlePayload`, `NotifyPayload`]
	}
)

func (p *Payload) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Payload) Unmarshal(data []byte) error {
	return json.Unmarshal(data, p)
}
