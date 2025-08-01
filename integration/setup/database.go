//go:build integration

package setup

import (
	"path/filepath"
	"testing"
	"twist/internal/proxy/database"
	"twist/internal/proxy/scripting"
	"twist/internal/proxy/scripting/types"
)

// DatabaseTestSetup provides database-specific test utilities
type DatabaseTestSetup struct {
	DB           database.Database
	GameAdapter  *scripting.GameAdapter
	TempDBPath   string
}

// SetupTestDatabase creates a temporary SQLite database for testing
func SetupTestDatabase(t *testing.T) *DatabaseTestSetup {
	// Create temporary file for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	// Create database instance
	db := database.NewDatabase()
	err := db.CreateDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	
	err = db.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	
	// Create game adapter with real database
	gameAdapter := scripting.NewGameAdapter(db)
	
	setup := &DatabaseTestSetup{
		DB:          db,
		GameAdapter: gameAdapter,
		TempDBPath:  dbPath,
	}
	
	// Register cleanup
	t.Cleanup(func() {
		setup.Cleanup()
	})
	
	return setup
}

// Cleanup closes the database and removes temporary files
func (setup *DatabaseTestSetup) Cleanup() {
	if setup.DB != nil {
		setup.DB.CloseDatabase()
	}
}

// VerifyScriptVariable checks if a variable is stored in the database with expected value
func (setup *DatabaseTestSetup) VerifyScriptVariable(t *testing.T, name string, expectedValue interface{}) {
	value, err := setup.GameAdapter.LoadScriptVariable(name)
	if err != nil {
		t.Errorf("Failed to load script variable %s: %v", name, err)
		return
	}
	
	switch expectedValue := expectedValue.(type) {
	case string:
		if value.Type != types.StringType || value.String != expectedValue {
			t.Errorf("Variable %s: expected string %q, got %v %q", 
				name, expectedValue, value.Type, value.String)
		}
	case float64:
		if value.Type != types.NumberType || value.Number != expectedValue {
			t.Errorf("Variable %s: expected number %f, got %v %f", 
				name, expectedValue, value.Type, value.Number)
		}
	case int:
		expectedFloat := float64(expectedValue)
		if value.Type != types.NumberType || value.Number != expectedFloat {
			t.Errorf("Variable %s: expected number %f, got %v %f", 
				name, expectedFloat, value.Type, value.Number)
		}
	default:
		t.Errorf("Unsupported expected value type: %T", expectedValue)
	}
}

// VerifyScriptVariableExists checks if a variable exists in the database
func (setup *DatabaseTestSetup) VerifyScriptVariableExists(t *testing.T, name string) bool {
	_, err := setup.GameAdapter.LoadScriptVariable(name)
	return err == nil
}

// SaveVariableToDatabase directly saves a variable to database for test setup
func (setup *DatabaseTestSetup) SaveVariableToDatabase(name string, value *types.Value) error {
	return setup.GameAdapter.SaveScriptVariable(name, value)
}

// LoadVariableFromDatabase directly loads a variable from database for test verification
func (setup *DatabaseTestSetup) LoadVariableFromDatabase(name string) (*types.Value, error) {
	return setup.GameAdapter.LoadScriptVariable(name)
}

// CreateSharedDatabaseSetup creates a second setup that shares the same database with the first
func (setup *DatabaseTestSetup) CreateSharedDatabaseSetup(t *testing.T) *DatabaseTestSetup {
	// Create new game adapter using the same database
	gameAdapter := scripting.NewGameAdapter(setup.DB)
	
	sharedSetup := &DatabaseTestSetup{
		DB:          setup.DB, // Share the same database
		GameAdapter: gameAdapter,
		TempDBPath:  setup.TempDBPath,
	}
	
	// Note: No cleanup registration here since the original setup handles database cleanup
	
	return sharedSetup
}