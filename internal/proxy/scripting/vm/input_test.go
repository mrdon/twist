package vm

import (
	"testing"
	"twist/internal/proxy/scripting/types"
)

// MockGameInterface for testing
type MockGameInterface struct{}

func (m *MockGameInterface) GetSector(index int) (types.SectorData, error) {
	return types.SectorData{}, nil
}
func (m *MockGameInterface) SetSectorParameter(sector int, name, value string) error { return nil }
func (m *MockGameInterface) GetSectorParameter(sector int, name string) (string, error) {
	return "", nil
}
func (m *MockGameInterface) GetDatabase() interface{}              { return nil }
func (m *MockGameInterface) GetCourse(from, to int) ([]int, error) { return []int{}, nil }
func (m *MockGameInterface) GetDistance(from, to int) (int, error) { return 0, nil }
func (m *MockGameInterface) GetAllCourses(from int) (map[int][]int, error) {
	return make(map[int][]int), nil
}
func (m *MockGameInterface) GetNearestWarps(sector int, count int) ([]int, error) {
	return []int{}, nil
}
func (m *MockGameInterface) GetCurrentSector() int    { return 1 }
func (m *MockGameInterface) GetCurrentPrompt() string { return "" }
func (m *MockGameInterface) GetLastOutput() string    { return "" }
func (m *MockGameInterface) GetSystemConstants() types.SystemConstantsInterface {
	return &MockSystemConstants{}
}
func (m *MockGameInterface) LoadScriptVariable(name string) (*types.Value, error)     { return nil, nil }
func (m *MockGameInterface) SaveScriptVariable(name string, value *types.Value) error { return nil }
func (m *MockGameInterface) SendCommand(command string) error                         { return nil }

// MockSystemConstants for testing
type MockSystemConstants struct{}

func (m *MockSystemConstants) GetConstant(name string) (*types.Value, bool) {
	return &types.Value{Type: types.StringType, String: ""}, false
}

func (m *MockSystemConstants) UpdateCurrentLine(text string) {}

func TestVMInputStateManagement(t *testing.T) {
	gameInterface := &MockGameInterface{}
	vm := NewVirtualMachine(gameInterface)

	// Initially, VM should not be waiting for input
	if vm.IsWaitingForInput() {
		t.Error("VM should not be waiting for input initially")
	}

	// Call GetInput to simulate script asking for input
	prompt := "Enter test value"
	result, err := vm.GetInput(prompt)
	if err != nil {
		t.Fatalf("GetInput failed: %v", err)
	}

	// Should return empty result initially
	if result != "" {
		t.Errorf("Expected empty result, got '%s'", result)
	}

	// VM should now be waiting for input and paused
	if !vm.IsWaitingForInput() {
		t.Error("VM should be waiting for input after GetInput call")
	}

	if !vm.state.IsPaused() {
		t.Error("VM should be paused after GetInput call")
	}

	// Check that prompt is stored correctly
	if vm.GetPendingInputPrompt() != prompt {
		t.Errorf("Expected pending prompt '%s', got '%s'", prompt, vm.GetPendingInputPrompt())
	}
}

func TestVMInputResumption(t *testing.T) {
	gameInterface := &MockGameInterface{}
	vm := NewVirtualMachine(gameInterface)

	// Set up VM to be waiting for input
	prompt := "Enter test value"
	_, err := vm.GetInput(prompt)
	if err != nil {
		t.Fatalf("GetInput failed: %v", err)
	}

	// Verify VM is waiting
	if !vm.IsWaitingForInput() {
		t.Fatal("VM should be waiting for input")
	}

	// Resume with input
	inputValue := "test_input_value"
	err = vm.ResumeWithInput(inputValue)
	if err != nil {
		t.Fatalf("ResumeWithInput failed: %v", err)
	}

	// VM should no longer be waiting for input
	if vm.IsWaitingForInput() {
		t.Error("VM should not be waiting for input after resume")
	}

	// VM should be running again
	if !vm.state.IsRunning() {
		t.Error("VM should be running after resume")
	}

	// Pending input should be cleared
	if vm.GetPendingInputPrompt() != "" {
		t.Error("Pending input prompt should be cleared after resume")
	}
}

func TestVMInputResumeError(t *testing.T) {
	gameInterface := &MockGameInterface{}
	vm := NewVirtualMachine(gameInterface)

	// Try to resume when not waiting for input
	err := vm.ResumeWithInput("test")
	if err == nil {
		t.Error("ResumeWithInput should fail when VM is not waiting for input")
	}

	expectedError := "VM is not waiting for input"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestVMInputStateTransitions(t *testing.T) {
	gameInterface := &MockGameInterface{}
	vm := NewVirtualMachine(gameInterface)

	// Test full input cycle
	tests := []struct {
		name            string
		action          func() error
		expectedWaiting bool
		expectedState   ExecutionState
	}{
		{
			name:            "Initial state",
			action:          func() error { return nil },
			expectedWaiting: false,
			expectedState:   StateHalted, // VM starts halted
		},
		{
			name: "After GetInput",
			action: func() error {
				_, err := vm.GetInput("test prompt")
				return err
			},
			expectedWaiting: true,
			expectedState:   StatePaused,
		},
		{
			name: "After Resume",
			action: func() error {
				return vm.ResumeWithInput("test input")
			},
			expectedWaiting: false,
			expectedState:   StateRunning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.action(); err != nil {
				t.Fatalf("Action failed: %v", err)
			}

			if vm.IsWaitingForInput() != tt.expectedWaiting {
				t.Errorf("Expected waiting=%t, got %t", tt.expectedWaiting, vm.IsWaitingForInput())
			}

			if vm.state.State != tt.expectedState {
				t.Errorf("Expected state=%v, got %v", tt.expectedState, vm.state.State)
			}
		})
	}
}
