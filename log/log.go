package log

import (
	"log/slog"
	"os"
	"strings"
)

var l *slog.Logger

func init() {
	var level slog.Level

	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelWarn
	}

	l = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

func Debug(msg string, args ...any) {
	l.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	l.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	l.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	l.Error(msg, args...)
}

func BotID(botID int64) slog.Attr {
	return slog.Int64("bot_id", botID)
}

func ChatID(chatID int64) slog.Attr {
	return slog.Int64("chat_id", chatID)
}

func Err(err error) slog.Attr {
	return slog.String("err", err.Error())
}

func String(key string, value string) slog.Attr {
	return slog.String(key, value)
}
