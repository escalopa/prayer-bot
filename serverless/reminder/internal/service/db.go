package service

import (
	"context"

	"github.com/escalopa/prayer-bot/internal/db"
)

type DB = db.Store

func NewDB(ctx context.Context) (*DB, error) {
	return db.Open(ctx)
}
