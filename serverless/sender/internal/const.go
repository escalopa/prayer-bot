package internal

import "encoding/json"

const (
	defaultLanguageCode = "en"

	prayerDayFormat  = "02.01.2006"
	prayerTimeFormat = "15:04"
	prayerText       = `
🗓 %s,  %s

🕊 %s — %s  
🌤 %s — %s  
☀️ %s — %s  
🌇 %s — %s  
🌅 %s — %s  
🌙 %s — %s
`
)

type replyType string

const (
	replyTypeBug      replyType = "bug"
	replyTypeFeedback replyType = "feedback"
)

type replyInfo struct {
	Type      replyType `json:"type"`
	ChatID    int64     `json:"chat_id"`
	MessageID int       `json:"message_id"`
	Username  string    `json:"username"`
}

func newReplyInfo(replyType replyType, chatID int64, messageID int, username string) *replyInfo {
	return &replyInfo{
		Type:      replyType,
		ChatID:    chatID,
		MessageID: messageID,
		Username:  username,
	}
}

func (r *replyInfo) JSON() string {
	bytes, _ := json.MarshalIndent(r, "", "\t")
	return string(bytes)
}

type callback string

const (
	callbackDataSplitter = "|"

	callbackEmpty callback = "empty|"

	callbackDateMonth callback = "date:month|"
	callbackDateDay   callback = "date:day|"

	callbackNotify callback = "notify|"

	callbackBug      callback = "bug|"
	callbackFeedback callback = "feedback|"
	callbackLanguage callback = "language|"
)

func (c callback) String() string {
	return string(c)
}

type state string

const (
	chatStateDefault state = "default"

	// user state

	chatStateBug      state = "bug"
	chatStateFeedback state = "feedback"

	// admin state

	chatStateReply    state = "reply"
	chatStateAnnounce state = "announce"
)

func (c state) String() string {
	return string(c)
}

type command string

const (
	// user commands

	startCommand       command = "start"
	helpCommand        command = "help"
	todayCommand       command = "today"
	dateCommand        command = "date" // 2 stages
	nextCommand        command = "next"
	notifyCommand      command = "notify"   // 1 stage
	bugCommand         command = "bug"      // 1 stage
	feedbackCommand    command = "feedback" // 1 stage
	languageCommand    command = "language" // 1 stage
	subscribeCommand   command = "subscribe"
	unsubscribeCommand command = "unsubscribe"
	cancelCommand      command = "cancel"

	// admin commands

	adminCommand    command = "admin"
	replyCommand    command = "reply" // 1 stage
	statsCommand    command = "stats"
	announceCommand command = "announce" // 1 stage
)

func (c command) String() string {
	return string(c)
}
