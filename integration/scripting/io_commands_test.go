

package scripting

import (
	"testing"
)

// TestClientMessage tests clientMessage command
func TestClientMessage_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		clientMessage "This is a test client message"
		echo "Message sent"
		setVar $msg "Dynamic message with variable"
		clientMessage $msg
		echo "Dynamic message sent"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	// Check the outputs (clientMessage currently uses same handler as echo in tests)
	expectedOutputs := []string{
		"This is a test client message",
		"Message sent",
		"Dynamic message with variable",
		"Dynamic message sent",
	}

	if len(result.Output) != len(expectedOutputs) {
		t.Errorf("Expected %d outputs, got %d", len(expectedOutputs), len(result.Output))
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Output line %d: got %q, want %q", i, result.Output[i], expected)
		}
	}
}

// TestGetInput tests getInput command (returns empty for testing)
func TestGetInput_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		getInput $userInput "Enter your name" 0
		echo "Input received: '" $userInput "'"
		getInput $number "Enter a number" 5
		echo "Number received: '" $number "'"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	// In test mode, getInput returns empty strings
	expectedOutputs := []string{
		"Input received: ''",
		"Number received: ''",
	}

	if len(result.Output) != len(expectedOutputs) {
		t.Errorf("Expected %d output lines, got %d", len(expectedOutputs), len(result.Output))
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Output line %d: got %q, want %q", i, result.Output[i], expected)
		}
	}
}