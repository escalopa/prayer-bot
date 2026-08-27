package handler

import "github.com/escalopa/prayer-bot/log"

func logLoader(op, detail string, args ...any) {
	log.Error("loader.handler."+op+": "+detail,
		append([]any{log.Op(op)}, args...)...)
}

func infoLoader(op, detail string, args ...any) {
	log.Info("loader.handler."+op+": "+detail,
		append([]any{log.Op(op)}, args...)...)
}
