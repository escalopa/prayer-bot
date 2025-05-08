package domain

import (
	"encoding/json"
	"strconv"
	"time"
)

type PayloadType string

const (
	PayloadTypeDispatcher PayloadType = "dispatcher"
	PayloadTypeReminder   PayloadType = "reminder"
)

type (
	location time.Location

	BotConfig struct {
		BotID    int64     `json:"bot_id"`
		OwnerID  int64     `json:"owner_id"`
		Token    string    `json:"token"`
		Secret   string    `json:"secret"`
		Location *location `json:"location"`
	}

	// ReminderPayload sent by `reminder-fn` to remind users about prayer times
	ReminderPayload struct {
		BotID          int64    `json:"bot_id"`
		ChatIDs        []int64  `json:"chat_ids"`
		PrayerID       PrayerID `json:"prayer_id"`
		ReminderOffset int32    `json:"reminder_offset"`
	}

	// DispatcherPayload sent by `dispatcher-fn` to handle incoming messages from the user
	DispatcherPayload struct {
		BotID int64  `json:"bot_id"`
		Data  string `json:"data"` // data is a JSON string of `*models.Update`
	}

	// Payload main struct that is sent to the `queue` for process
	Payload struct {
		Type PayloadType `json:"type"` // one of [`dispatcher`, `reminder`]
		Data interface{} `json:"data"` // one of [`DispatcherPayload`, `ReminderPayload`]
	}
)

func (p *Payload) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Payload) Unmarshal(data []byte) error {
	return json.Unmarshal(data, p)
}

func (l *location) UnmarshalJSON(bytes []byte) error {
	data, err := strconv.Unquote(string(bytes))
	if err != nil {
		return err
	}

	loc, err := time.LoadLocation(data)
	if err != nil {
		return err
	}

	*l = location(*loc)
	return nil
}

func (l *location) V() *time.Location {
	if l == nil {
		return nil
	}
	return (*time.Location)(l)
}
