package scripting

import (
	"strings"
	"testing"
)

// TestGetInputCommand_BasicUsage tests GETINPUT command with basic usage
func TestGetInputCommand_BasicUsage_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test getinput command basic functionality
		setVar $defaultValue "test_default"
		getinput $userInput "Enter test value" $defaultValue
		echo "Got input: " $userInput
		halt
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	// The script should pause at getinput and display the prompt
	// This is the correct TWX behavior - script waits for user input
	expectedPrompt := "Enter test value [test_default]"
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

// TestGetInputCommand_WithoutDefault tests GETINPUT without default value
func TestGetInputCommand_WithoutDefault_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test getinput without default
		getinput $userInput "Enter your name"
		echo "Hello " $userInput "!"
		halt
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	// Script should pause and show prompt without default value brackets
	expectedPrompt := "Enter your name"
	found := false
	for _, output := range result.Output {
		if strings.Contains(output, expectedPrompt) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected prompt %q, got outputs: %v", expectedPrompt, result.Output)
	}
}

// TestGetInputCommand_MultipleInputs tests multiple GETINPUT commands
func TestGetInputCommand_MultipleInputs_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test multiple getinput calls like the 1_Port.ts script
		echo "Port Pair Trading Script"
		getinput $sector2 "Enter sector to trade to" 0
		getinput $timesLeft "Enter times to execute script" 0  
		getinput $percent "Enter markup/markdown percentage" 5
		echo "Configuration: Sector=" $sector2 " Times=" $timesLeft " Percent=" $percent
		halt
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	// Should display the banner and pause at first getinput
	bannerFound := false
	promptFound := false
	for _, output := range result.Output {
		if strings.Contains(output, "Port Pair Trading Script") {
			bannerFound = true
		}
		if strings.Contains(output, "Enter sector to trade to [0]") {
			promptFound = true
		}
	}

	if !bannerFound {
		t.Error("Expected to see script banner in output")
	}

	if !promptFound {
		t.Errorf("Expected to see first input prompt, got outputs: %v", result.Output)
	}
}

// TestGetInputCommand_PromptFormatting tests prompt formatting with and without defaults
func TestGetInputCommand_PromptFormatting_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test prompt formatting - this should pause at first getinput
		getinput $test "Test prompt without default"
		halt
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	// Should show prompt without brackets since no default provided
	expectedPrompt := "Test prompt without default"
	found := false
	for _, output := range result.Output {
		if strings.Contains(output, expectedPrompt) && !strings.Contains(output, "[") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected prompt without brackets %q, got outputs: %v", expectedPrompt, result.Output)
	}
}
