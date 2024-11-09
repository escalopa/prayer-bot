package handler

const (
	operationCanceled = "Operation canceled."
	unexpectedErrMsg  = "unexpected error 😢\nUse /bug to report the error if it remains"
)

const (
	respondErr     = "Failed to respond to user."
	respondSuccess = "Successfully responded to user."

	respondStart      = "Send your response message, Or /cancel"
	respondNoReplyMsg = "No reply message provided, /respond"
	respondInvalidMsg = "Invalid message, can't parse."
)

const (
	getSubscribersErr = "Failed to get subscribers."
)

const (
	sendAllErr     = "Failed to send message to all subscribers."
	sendAllSuccess = "Successfully sent message to all subscribers."

	sendAllStart   = "Send your message, Or /cancel"
	sendAllConfirm = "Use /confirm to send the message, Or /cancel"
)
