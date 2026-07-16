package store

import (
	"testing"

	"github.com/escalopa/prayer-bot/global/internal/database"
)

func TestQualifySQLUsesIsolatedSchema(t *testing.T) {
	query := qualifySQL("SELECT * FROM global_bot.chats", database.TestingSchema)
	if query != `SELECT * FROM "global_bot_testing".chats` {
		t.Fatalf("unexpected qualified query: %s", query)
	}
}

func TestQualifySQLDoesNotChangeUnrelatedNames(t *testing.T) {
	query := qualifySQL("SELECT * FROM public.chats", database.ProductionSchema)
	if query != "SELECT * FROM public.chats" {
		t.Fatalf("unexpected query rewrite: %s", query)
	}
}
