package scripting

import (
	"strings"
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

// TestGetInput tests getInput command (pauses execution like TWX)
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

	// Script should pause at first getInput command and display prompt
	// This is correct TWX behavior - script waits for user input
	expectedPrompt := "Enter your name [0]"
	found := false
	for _, output := range result.Output {
		if strings.Contains(output, expectedPrompt) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected prompt containing %q, got outputs: %v", expectedPrompt, result.Output)
	}
}
