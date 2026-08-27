package handler

import "github.com/escalopa/prayer-bot/log"

func logReminder(op, detail string, args ...any) {
	log.Error("reminder.handler."+op+": "+detail,
		append([]any{log.Op(op)}, args...)...)
}

func warnReminder(op, detail string, args ...any) {
	log.Warn("reminder.handler."+op+": "+detail,
		append([]any{log.Op(op)}, args...)...)
}
