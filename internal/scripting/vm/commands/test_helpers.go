package commands

import (
	"os"
	"path/filepath"
	"testing"
	"twist/internal/database"
	"twist/internal/scripting/types"
)

// TestDatabaseSetup provides a test database and mock VM for testing
type TestDatabaseSetup struct {
	DB           database.Database
	GameAdapter  GameAdapterInterface
	VM           types.VMInterface
	TempDBPath   string
}

// GameAdapterInterface provides interface for testing
type GameAdapterInterface interface {
	types.GameInterface
	SaveScriptVariable(name string, value *types.Value) error
	LoadScriptVariable(name string) (*types.Value, error)
}

// TestGameAdapter implements GameAdapterInterface for testing
type TestGameAdapter struct {
	db database.Database
}

func NewTestGameAdapter(db database.Database) *TestGameAdapter {
	return &TestGameAdapter{db: db}
}

func (g *TestGameAdapter) GetSector(index int) (types.SectorData, error) {
	return types.SectorData{Number: index}, nil
}

func (g *TestGameAdapter) SetSectorParameter(sector int, name, value string) error {
	return nil
}

func (g *TestGameAdapter) GetSectorParameter(sector int, name string) (string, error) {
	return "", nil
}

func (g *TestGameAdapter) GetDatabase() interface{} {
	return g.db
}

func (g *TestGameAdapter) GetCourse(from, to int) ([]int, error) {
	return []int{from, to}, nil
}

func (g *TestGameAdapter) GetDistance(from, to int) (int, error) {
	return 1, nil
}

func (g *TestGameAdapter) GetAllCourses(from int) (map[int][]int, error) {
	return make(map[int][]int), nil
}

func (g *TestGameAdapter) GetNearestWarps(sector int, count int) ([]int, error) {
	return []int{}, nil
}

func (g *TestGameAdapter) GetCurrentSector() int {
	return 1
}

func (g *TestGameAdapter) GetCurrentPrompt() string {
	return "Command [TL=00:00:00]:"
}

func (g *TestGameAdapter) SendCommand(cmd string) error {
	return nil
}

func (g *TestGameAdapter) GetLastOutput() string {
	return ""
}

func (g *TestGameAdapter) SaveScriptVariable(name string, value *types.Value) error {
	var dbValue interface{}
	switch value.Type {
	case types.StringType:
		dbValue = value.String
	case types.NumberType:
		dbValue = value.Number
	default:
		dbValue = value.String
	}
	return g.db.SaveScriptVariable(name, dbValue)
}

func (g *TestGameAdapter) LoadScriptVariable(name string) (*types.Value, error) {
	value, err := g.db.LoadScriptVariable(name)
	if err != nil {
		return nil, err
	}
	
	switch v := value.(type) {
	case string:
		return &types.Value{Type: types.StringType, String: v}, nil
	case float64:
		return &types.Value{Type: types.NumberType, Number: v}, nil
	default:
		return &types.Value{Type: types.StringType, String: ""}, nil
	}
}

// SetupTestDatabase creates a temporary SQLite database with real VM for testing
func SetupTestDatabase(t *testing.T) *TestDatabaseSetup {
	// Create temporary file for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	// Create database instance
	db := database.NewDatabase()
	err := db.CreateDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	
	// Create game adapter with real database
	gameAdapter := NewTestGameAdapter(db)
	
	// Create mock VM with the game adapter
	testVM := &TestMockVM{
		variables:    make(map[string]*types.Value),
		gameAdapter:  gameAdapter,
	}
	
	return &TestDatabaseSetup{
		DB:          db,
		GameAdapter: gameAdapter,
		VM:          testVM,
		TempDBPath:  dbPath,
	}
}

// Cleanup closes the database and removes temporary files
func (setup *TestDatabaseSetup) Cleanup() {
	if setup.DB != nil {
		setup.DB.CloseDatabase()
	}
	// TempDir() automatically cleans up, but we can also explicitly remove
	if setup.TempDBPath != "" {
		os.Remove(setup.TempDBPath)
	}
}

// VerifyScriptVariable checks if a variable is stored in the database with expected value
func (setup *TestDatabaseSetup) VerifyScriptVariable(t *testing.T, name string, expectedValue interface{}) {
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

// SaveVariableToDatabase directly saves a variable to database for test setup
func (setup *TestDatabaseSetup) SaveVariableToDatabase(name string, value *types.Value) error {
	return setup.GameAdapter.SaveScriptVariable(name, value)
}

// TestMockVM implements VMInterface for testing
type TestMockVM struct {
	variables   map[string]*types.Value
	gameAdapter GameAdapterInterface
}

func (m *TestMockVM) GetVariable(name string) *types.Value {
	if val, exists := m.variables[name]; exists {
		return val
	}
	// Check database for persistent variables
	if m.gameAdapter != nil {
		if dbValue, err := m.gameAdapter.LoadScriptVariable(name); err == nil {
			m.variables[name] = dbValue // Cache it
			return dbValue
		}
	}
	return &types.Value{Type: types.StringType, String: ""}
}

func (m *TestMockVM) SetVariable(name string, value *types.Value) {
	m.variables[name] = value
	// Also persist to database for real integration testing
	if m.gameAdapter != nil {
		m.gameAdapter.SaveScriptVariable(name, value)
	}
}

func (m *TestMockVM) GetGameInterface() types.GameInterface {
	return m.gameAdapter
}

func (m *TestMockVM) Echo(text string) error { return nil }
func (m *TestMockVM) ClientMessage(text string) error { return nil }
func (m *TestMockVM) Error(text string) error { 
	return &types.VMError{Message: text}
}
func (m *TestMockVM) Send(command string) error { return nil }
func (m *TestMockVM) WaitFor(pattern string) error { return nil }
func (m *TestMockVM) Pause() error { return nil }
func (m *TestMockVM) Halt() error { return nil }
func (m *TestMockVM) GetCurrentSector() int { return 1 }
func (m *TestMockVM) GetCurrentLine() string { return "" }
func (m *TestMockVM) Goto(label string) error { return nil }
func (m *TestMockVM) Gosub(label string) error { return nil }
func (m *TestMockVM) Return() error { return nil }
func (m *TestMockVM) GetInput(prompt string) (string, error) { return "", nil }
func (m *TestMockVM) GetCurrentScript() types.ScriptInterface { return nil }
func (m *TestMockVM) LoadAdditionalScript(filename string) (types.ScriptInterface, error) { return nil, nil }
func (m *TestMockVM) StopScript(scriptID string) error { return nil }
func (m *TestMockVM) GetScriptManager() interface{} { return nil }
func (m *TestMockVM) SetTrigger(trigger types.TriggerInterface) error { return nil }
func (m *TestMockVM) KillTrigger(triggerID string) error { return nil }
func (m *TestMockVM) GetActiveTriggersCount() int { return 0 }
func (m *TestMockVM) KillAllTriggers() {}
func (m *TestMockVM) ProcessInput(filter string) error { return nil }
func (m *TestMockVM) ProcessOutput(filter string) error { return nil }

// VarParam methods for Pascal-compatible array support
func (m *TestMockVM) GetVarParam(name string) *types.VarParam {
	// For testing, create simple VarParam if not exists
	return types.NewVarParam(name, types.VarParamVariable)
}

func (m *TestMockVM) SetVarParam(name string, varParam *types.VarParam) {
	// For testing, this is a no-op since we're using the simple mock
}

// Test parameter creation helper functions
func createStringParam(value string) *types.CommandParam {
	return &types.CommandParam{
		Type:  types.ParamValue,
		Value: &types.Value{Type: types.StringType, String: value},
	}
}

func createVarParam(varName string) *types.CommandParam {
	return &types.CommandParam{
		Type:    types.ParamVar,
		VarName: varName,
	}
}

func createNumberParam(value float64) *types.CommandParam {
	return &types.CommandParam{
		Type:  types.ParamValue,
		Value: &types.Value{Type: types.NumberType, Number: value},
	}
}