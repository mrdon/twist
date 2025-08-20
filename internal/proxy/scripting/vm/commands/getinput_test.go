package commands

import (
	"testing"
	"twist/internal/proxy/scripting/types"
)

// MockVMInterface for testing getinput command
type MockVMInterface struct {
	variables       map[string]*types.Value
	lastEcho        string
	lastInputPrompt string
	inputResult     string
	paused          bool
}

func NewMockVM() *MockVMInterface {
	return &MockVMInterface{
		variables: make(map[string]*types.Value),
	}
}

func (m *MockVMInterface) GetVariable(name string) *types.Value {
	if val, exists := m.variables[name]; exists {
		return val
	}
	return &types.Value{Type: types.StringType, String: ""}
}

func (m *MockVMInterface) SetVariable(name string, value *types.Value) {
	m.variables[name] = value
}

func (m *MockVMInterface) Echo(message string) error {
	m.lastEcho = message
	return nil
}

func (m *MockVMInterface) GetInput(prompt string) (string, error) {
	m.lastInputPrompt = prompt
	m.paused = true
	return m.inputResult, nil
}

func (m *MockVMInterface) Pause() error {
	m.paused = true
	return nil
}

// Stub implementations for other required methods
func (m *MockVMInterface) GetVarParam(name string) *types.VarParam           { return nil }
func (m *MockVMInterface) SetVarParam(name string, varParam *types.VarParam) {}
func (m *MockVMInterface) Goto(label string) error                           { return nil }
func (m *MockVMInterface) Gosub(label string) error                          { return nil }
func (m *MockVMInterface) Return() error                                     { return nil }
func (m *MockVMInterface) Halt() error                                       { return nil }
func (m *MockVMInterface) ClientMessage(message string) error                { return nil }
func (m *MockVMInterface) WaitFor(text string) error                         { return nil }
func (m *MockVMInterface) Send(data string) error                            { return nil }
func (m *MockVMInterface) GetGameInterface() types.GameInterface             { return nil }
func (m *MockVMInterface) GetCurrentScript() types.ScriptInterface           { return nil }
func (m *MockVMInterface) LoadAdditionalScript(filename string) (types.ScriptInterface, error) {
	return nil, nil
}
func (m *MockVMInterface) StopScript(scriptID string) error                { return nil }
func (m *MockVMInterface) GetScriptManager() interface{}                   { return nil }
func (m *MockVMInterface) SetTrigger(trigger types.TriggerInterface) error { return nil }
func (m *MockVMInterface) KillTrigger(triggerID string) error              { return nil }
func (m *MockVMInterface) GetActiveTriggersCount() int                     { return 0 }
func (m *MockVMInterface) KillAllTriggers()                                {}
func (m *MockVMInterface) Error(message string) error                      { return nil }
func (m *MockVMInterface) ProcessInput(filter string) error                { return nil }
func (m *MockVMInterface) ProcessOutput(filter string) error               { return nil }
func (m *MockVMInterface) EvaluateExpression(expression string) (*types.Value, error) {
	return nil, nil
}
func (m *MockVMInterface) GetCurrentLine() int { return 0 }

// New methods for input handling
func (m *MockVMInterface) IsWaitingForInput() bool       { return m.paused }
func (m *MockVMInterface) GetPendingInputPrompt() string { return m.lastInputPrompt }
func (m *MockVMInterface) GetPendingInputResult() string { return m.inputResult }
func (m *MockVMInterface) JustResumedFromInput() bool    { return false } // Mock doesn't use this flag
func (m *MockVMInterface) ClearPendingInput() {
	m.lastInputPrompt = ""
	m.inputResult = ""
	m.paused = false
}

func TestGetInputBasic(t *testing.T) {
	vm := NewMockVM()

	// Test basic getinput with just variable and prompt
	params := []*types.CommandParam{
		{Type: types.ParamVar, VarName: "testvar"},
		{Type: types.ParamValue, Value: &types.Value{Type: types.StringType, String: "Enter value"}},
	}

	// First call - should initiate input and pause
	err := cmdGetInput(vm, params)
	if err == nil {
		t.Fatalf("Expected script to pause on first call")
	}

	// Check that prompt was displayed
	expectedPrompt := "Enter value"
	if vm.lastInputPrompt != expectedPrompt {
		t.Errorf("Expected prompt '%s', got '%s'", expectedPrompt, vm.lastInputPrompt)
	}

	// Simulate user providing input
	vm.inputResult = "test_input"

	// Second call - should process the input and complete
	err = cmdGetInput(vm, params)
	if err != nil {
		t.Fatalf("cmdGetInput failed on resume: %v", err)
	}

	// Check that variable was set
	result := vm.GetVariable("testvar")
	if result.String != "test_input" {
		t.Errorf("Expected variable value 'test_input', got '%s'", result.String)
	}
}

func TestGetInputWithDefault(t *testing.T) {
	vm := NewMockVM()

	// Test getinput with default value
	params := []*types.CommandParam{
		{Type: types.ParamVar, VarName: "testvar"},
		{Type: types.ParamValue, Value: &types.Value{Type: types.StringType, String: "Enter value"}},
		{Type: types.ParamValue, Value: &types.Value{Type: types.StringType, String: "default_val"}},
	}

	// First call - should initiate input and pause
	err := cmdGetInput(vm, params)
	if err == nil {
		t.Fatalf("Expected script to pause on first call")
	}

	// Check that prompt includes default value
	expectedPrompt := "Enter value [default_val]"
	if vm.lastInputPrompt != expectedPrompt {
		t.Errorf("Expected prompt '%s', got '%s'", expectedPrompt, vm.lastInputPrompt)
	}

	// Simulate empty input (user presses enter)
	vm.inputResult = ""

	// Second call - should process empty input and use default
	err = cmdGetInput(vm, params)
	if err != nil {
		t.Fatalf("cmdGetInput failed on resume: %v", err)
	}

	// Check that default value was used
	result := vm.GetVariable("testvar")
	if result.String != "default_val" {
		t.Errorf("Expected variable value 'default_val', got '%s'", result.String)
	}
}

func TestGetInputUserOverridesDefault(t *testing.T) {
	vm := NewMockVM()

	// Test that user input overrides default
	params := []*types.CommandParam{
		{Type: types.ParamVar, VarName: "testvar"},
		{Type: types.ParamValue, Value: &types.Value{Type: types.StringType, String: "Enter value"}},
		{Type: types.ParamValue, Value: &types.Value{Type: types.StringType, String: "default_val"}},
	}

	// First call - should initiate input and pause
	err := cmdGetInput(vm, params)
	if err == nil {
		t.Fatalf("Expected script to pause on first call")
	}

	// User provides input
	vm.inputResult = "user_input"

	// Second call - should process user input and override default
	err = cmdGetInput(vm, params)
	if err != nil {
		t.Fatalf("cmdGetInput failed on resume: %v", err)
	}

	// Check that user input was used instead of default
	result := vm.GetVariable("testvar")
	if result.String != "user_input" {
		t.Errorf("Expected variable value 'user_input', got '%s'", result.String)
	}
}

func TestGetInputPromptFormatting(t *testing.T) {
	tests := []struct {
		name       string
		prompt     string
		defaultVal string
		expected   string
	}{
		{
			name:       "No default",
			prompt:     "Enter name",
			defaultVal: "",
			expected:   "Enter name",
		},
		{
			name:       "With default",
			prompt:     "Enter sector",
			defaultVal: "1",
			expected:   "Enter sector [1]",
		},
		{
			name:       "Empty prompt with default",
			prompt:     "",
			defaultVal: "test",
			expected:   " [test]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := NewMockVM()

			params := []*types.CommandParam{
				{Type: types.ParamVar, VarName: "testvar"},
				{Type: types.ParamValue, Value: &types.Value{Type: types.StringType, String: tt.prompt}},
			}

			if tt.defaultVal != "" {
				params = append(params, &types.CommandParam{
					Type:  types.ParamValue,
					Value: &types.Value{Type: types.StringType, String: tt.defaultVal},
				})
			}

			// First call - should initiate input and pause
			err := cmdGetInput(vm, params)
			if err == nil {
				t.Fatalf("Expected script to pause on first call")
			}

			if vm.lastInputPrompt != tt.expected {
				t.Errorf("Expected prompt '%s', got '%s'", tt.expected, vm.lastInputPrompt)
			}

			// Set input result and make second call to complete the test
			vm.inputResult = "test"

			// Second call - should process input and complete
			err = cmdGetInput(vm, params)
			if err != nil {
				t.Fatalf("cmdGetInput failed on resume: %v", err)
			}
		})
	}
}
