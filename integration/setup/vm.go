package setup

import (
	"os"
	"path/filepath"
	"testing"
	"twist/internal/proxy/database"
	"twist/internal/proxy/scripting"
	"twist/internal/proxy/scripting/types"
	"twist/internal/proxy/scripting/vm"
)

// IntegrationTestSetup provides real components for integration testing
type IntegrationTestSetup struct {
	DB          database.Database
	GameAdapter *scripting.GameAdapter
	VM          *vm.VirtualMachine
	DBPath      string
	t           *testing.T
}

// SetupRealComponents creates real production components for integration testing
func SetupRealComponents(t *testing.T) *IntegrationTestSetup {
	// Create temporary database file
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Create real database using the existing pattern
	db := database.NewDatabase()
	err := db.CreateDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create real game adapter
	gameAdapter := scripting.NewGameAdapter(db)

	// Create real VM
	realVM := vm.NewVirtualMachine(gameAdapter)

	// Initialize CURRENTLINE system constant for integration tests
	// This ensures tests that use cutText on CURRENTLINE have valid data to work with
	if systemConstants := gameAdapter.GetSystemConstants(); systemConstants != nil {
		systemConstants.UpdateCurrentLine("Command [TL=00:00:00]:")
	}

	setup := &IntegrationTestSetup{
		DB:          db,
		GameAdapter: gameAdapter,
		VM:          realVM,
		DBPath:      dbPath,
		t:           t,
	}

	// Register cleanup
	t.Cleanup(func() {
		setup.Cleanup()
	})

	return setup
}

// Cleanup cleans up all test resources
func (s *IntegrationTestSetup) Cleanup() {
	if s.DB != nil {
		s.DB.CloseDatabase()
	}

	// Remove database file (t.TempDir() handles this automatically, but explicit is better)
	if s.DBPath != "" {
		os.Remove(s.DBPath)
	}
}

// VerifyScriptVariable checks that a variable is persisted correctly in the database
func (s *IntegrationTestSetup) VerifyScriptVariable(t *testing.T, name string, expectedValue interface{}) {
	value, err := s.GameAdapter.LoadScriptVariable(name)
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
func (s *IntegrationTestSetup) VerifyScriptVariableExists(t *testing.T, name string) bool {
	_, err := s.GameAdapter.LoadScriptVariable(name)
	return err == nil
}

// CreateTestValue creates a Value for testing
func CreateTestValue(valueType types.ValueType, stringVal string, numberVal float64) *types.Value {
	return &types.Value{
		Type:   valueType,
		String: stringVal,
		Number: numberVal,
	}
}

// SaveVariableToDatabase directly saves a variable to database for test setup
func (s *IntegrationTestSetup) SaveVariableToDatabase(name string, value *types.Value) error {
	return s.GameAdapter.SaveScriptVariable(name, value)
}

// LoadVariableFromDatabase directly loads a variable from database for test verification
func (s *IntegrationTestSetup) LoadVariableFromDatabase(name string) (*types.Value, error) {
	return s.GameAdapter.LoadScriptVariable(name)
}
