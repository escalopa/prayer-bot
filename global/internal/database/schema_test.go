package database

import "testing"

func TestValidateSchema(t *testing.T) {
	for _, schema := range []string{TestingSchema, ProductionSchema} {
		if err := ValidateSchema(schema); err != nil {
			t.Fatalf("expected %q to be valid: %v", schema, err)
		}
	}

	for _, schema := range []string{"", "global_bot", "public", "global_bot_testing,public"} {
		if err := ValidateSchema(schema); err == nil {
			t.Fatalf("expected %q to be invalid", schema)
		}
	}
}
