package domain

import "encoding/json"

const (
	StageSplitter = "-"
)

type Delay int

const (
	Delay5m  Delay = 5
	Delay10m Delay = 10
	Delay15m Delay = 15
	Delay30m Delay = 30
	Delay60m Delay = 60
)

type Command string

const (
	// handler commands

	StartCommand Command = "/start"
	HelpCommand  Command = "/help"

	TodayCommand       Command = "/today"
	DateCommand        Command = "/date"   // 2 stages
	NotifyCommand      Command = "/notify" // 1 stage
	SubscribeCommand   Command = "/subscribe"
	UnsubscribeCommand Command = "/unsubscribe"
	LangCommand        Command = "/lang" // 1 stage

	FeedbackCommand Command = "/feedback" // 1 stage
	BugCommand      Command = "/bug"      // 1 stage

	// notifier commands

	NotifySoon Command = "/notify_soon"
	NotifyNow  Command = "/notify_now"
)

type (
	BotConfig struct {
		BotID    uint8  `json:"bot_id"`
		Location string `json:"location"`
		Token    string `json:"token"`
		Secret   string `json:"secret"`
	}

	NotifyUser struct {
		ChatID    int64  `json:"chat_id"`
		Lang      string `json:"lang"`
		MessageID int64  `json:"message_id"` // previous notify messageID
	}

	NotifyPayload struct {
		Users []NotifyUser `json:"users"`
		Delay Delay        `json:"delay"`
	}

	Payload struct {
		BotID  uint8 `json:"bot_id"`
		ChatID int64 `json:"chat_id"`

		Command Command `json:"command"`
		Stage   uint8   `json:"stage"`

		Data interface{} `json:"payload"`
	}
)

func (p *Payload) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Payload) Unmarshal(data []byte) error {
	return json.Unmarshal(data, p)
}

func IsValidCommand(cmd string) bool {
	for _, command := range []Command{
		StartCommand,
		HelpCommand,
		TodayCommand,
		DateCommand,
		NotifyCommand,
		SubscribeCommand,
		UnsubscribeCommand,
		LangCommand,
		FeedbackCommand,
		BugCommand,
	} {
		if cmd == string(command) {
			return true
		}
	}
	return false
}
