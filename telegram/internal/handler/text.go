package handler

const (
	// User commands

	cmdStart       = "/start"
	cmdHelp        = "/help"
	cmdSubscribe   = "/subscribe"
	cmdUnsubscribe = "/unsubscribe"
	cmdToday       = "/today"
	cmdDate        = "/date"
	cmdLang        = "/lang"
	cmdFeedback    = "/feedback"
	cmdBug         = "/bug"

	// Admin commands

	cmdRespond      = "/respond"
	cmdGetSubscribe = "/subs"
	cmdSendAll      = "/sall"

	// Other commands
	cmdCancel  = "/cancel"
	cmdConfirm = "/confirm"
)

const (
	unexpectedErrMsg = "Unexpected error 😢\nUse /bug to report the error if it remains"

	operationCanceled = "Operation canceled"
)

// text messages for /feedback command
const (
	feedbackSendMsg = `
💬 Feedback Message 💬

<b>User ID:</b> %d
<b>Username:</b> @%s
<b>Full Name:</b> %s %s
<b>Message ID:</b> %d
<b>Feedback:</b> %s
	`
)

// text messages for /bug command
const (
	bugSendMsg = `
🐞 Bug Report 🐞

<b>User ID:</b> %d
<b>Username:</b> @%s
<b>Full Name:</b> %s %s
<b>Message ID:</b> %d
<b>Bug Report:</b> %s
	`
)

// text messages for /respond command
const (
	respondErr     = "Failed to respond to user"
	respondSuccess = "Successfully responded to user"

	respondStart      = "Send your response message, Or /cancel"
	respondNoReplyMsg = "No reply message provided, /respond"
	respondInvalidMsg = "Invalid reply message: not parsable"
)

// text messages for /sendall command
const (
	getSubscribersErr = "Failed to get subscribers"
)

// text messages for /sendall command
const (
	sendAllErr     = "Failed to send message to all subscribers"
	sendAllSuccess = "Successfully sent message to all subscribers"

	sendAllStart   = "Send your message, Or /cancel"
	sendAllConfirm = "Use /confirm to send the message, Or /cancel"
)

// text messages for /today & /date command
const (
	prayerText = "```\n%s %d %s 🕌\n\n%s```\n/help"
)
