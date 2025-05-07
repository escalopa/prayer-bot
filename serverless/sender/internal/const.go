package internal

const (
	defaultLanguageCode = "en"

	prayerDayFormat  = "02.01.2006"
	prayerTimeFormat = "15:04"
	prayerText       = `
🗓 %s, %s
🕊 %s — %s  
🌤 %s — %s  
☀️ %s — %s  
🌇 %s — %s  
🌅 %s — %s  
🌙 %s — %s
	`
)

type callback string

const (
	callbackDataSplitter = "|"

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
