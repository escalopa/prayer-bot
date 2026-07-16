package database

import "fmt"

const (
	TestingSchema    = "global_bot_testing"
	ProductionSchema = "global_bot_production"
)

func ValidateSchema(schema string) error {
	switch schema {
	case TestingSchema, ProductionSchema:
		return nil
	default:
		return fmt.Errorf("database schema must be %q or %q", TestingSchema, ProductionSchema)
	}
}
