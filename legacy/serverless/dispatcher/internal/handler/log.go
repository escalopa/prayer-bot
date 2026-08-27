package handler

import "github.com/escalopa/prayer-bot/log"

func logDispatcher(op, detail string, args ...any) {
	log.Error("dispatcher.handler."+op+": "+detail,
		append([]any{log.Op(op)}, args...)...)
}

func infoDispatcher(op, detail string, args ...any) {
	log.Info("dispatcher.handler."+op+": "+detail,
		append([]any{log.Op(op)}, args...)...)
}

func logCommand(op string, args ...any) {
	log.Error("dispatcher.command."+op,
		append([]any{log.Op(op)}, args...)...)
}

func logQuery(op string, args ...any) {
	log.Error("dispatcher.query."+op,
		append([]any{log.Op(op)}, args...)...)
}

func logState(op string, args ...any) {
	log.Error("dispatcher.state."+op,
		append([]any{log.Op(op)}, args...)...)
}
