package store

import (
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/escalopa/prayer-bot/global/internal/database"
)

func TestRuntimePoolConfigDisablesNamedPreparedStatements(t *testing.T) {
	config, err := runtimePoolConfig(
		"postgres://user:password@localhost:5432/database?default_query_exec_mode=cache_statement",
	)
	if err != nil {
		t.Fatal(err)
	}
	if config.ConnConfig.DefaultQueryExecMode != pgx.QueryExecModeExec {
		t.Fatalf(
			"default query execution mode = %s, want %s",
			config.ConnConfig.DefaultQueryExecMode,
			pgx.QueryExecModeExec,
		)
	}
}

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
