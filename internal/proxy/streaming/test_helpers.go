package streaming

import (
	"twist/internal/proxy/database"
)

// test_helpers.go - Helper functions for testing

// NewTestDatabase creates an in-memory database for testing
func NewTestDatabase() database.Database {
	db := database.NewDatabase()
	// Create an in-memory SQLite database for testing
	if err := db.CreateDatabase(":memory:"); err != nil {
		panic("Failed to create test database: " + err.Error())
	}
	return db
}

// NewTestTWXParser creates a parser with a test database for testing
func NewTestTWXParser() *TWXParser {
	return NewTWXParser(NewTestDatabase(), nil)
}
