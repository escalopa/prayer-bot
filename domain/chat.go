package domain

type NotifyOffset int32

const (
	NotifyOffset0m  NotifyOffset = 0
	NotifyOffset5m  NotifyOffset = 5
	NotifyOffset10m NotifyOffset = 10
	NotifyOffset15m NotifyOffset = 15
	NotifyOffset20m NotifyOffset = 20
	NotifyOffset30m NotifyOffset = 30
	NotifyOffset45m NotifyOffset = 45
	NotifyOffset60m NotifyOffset = 60
	NotifyOffset90m NotifyOffset = 90
)

func NotifyOffsets() []int32 {
	return []int32{
		int32(NotifyOffset5m),
		int32(NotifyOffset10m),
		int32(NotifyOffset15m),
		int32(NotifyOffset20m),
		int32(NotifyOffset30m),
		int32(NotifyOffset45m),
		int32(NotifyOffset60m),
		int32(NotifyOffset90m),
	}
}

type (
	Chat struct {
		BotID           int32
		ChatID          int64
		State           string
		LanguageCode    string
		NotifyMessageID int32
	}

	Stats struct {
		Users            int            // count of users using the bot
		Subscribed       int            // count of subscribed users
		Unsubscribed     int            // count of unsubscribed users
		LanguagesGrouped map[string]int // count of users using a language
	}
)
