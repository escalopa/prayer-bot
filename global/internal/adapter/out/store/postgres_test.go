package store

import (
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/escalopa/prayer-bot/global/internal/database"
	"github.com/escalopa/prayer-bot/global/internal/domain"
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

func TestMarshalJSONTextProducesJSONBCompatibleTextParameter(t *testing.T) {
	encoded, err := marshalJSONText(domain.Adjustments{Fajr: 2, Isha: -1})
	if err != nil {
		t.Fatal(err)
	}
	if encoded == "" || encoded[0] != '{' {
		t.Fatalf("unexpected JSON text: %q", encoded)
	}
	var decoded domain.Adjustments
	if err := json.Unmarshal([]byte(encoded), &decoded); err != nil {
		t.Fatalf("JSON text cannot be decoded: %v", err)
	}
	if decoded.Fajr != 2 || decoded.Isha != -1 {
		t.Fatalf("unexpected decoded adjustments: %+v", decoded)
	}
}
