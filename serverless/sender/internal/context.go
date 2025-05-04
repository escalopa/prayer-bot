package internal

import (
	"context"
)

type contextBotIDKey struct{}

func setContextBotID(ctx context.Context, botID int32) context.Context {
	return context.WithValue(ctx, contextBotIDKey{}, botID)
}

func getContextBotID(ctx context.Context) int32 {
	botID, _ := ctx.Value(contextBotIDKey{}).(int32)
	return botID
}
