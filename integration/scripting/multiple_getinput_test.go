package scripting

import (
	"testing"
)

// TestMultipleConsecutiveGetInput verifies that multiple getinput commands work correctly
// This replicates the Port Pair Trading script issue where script restarts after each input
func TestMultipleConsecutiveGetInput_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	// Simulate the Port Pair Trading script pattern:
	// getinput $sector2 "Enter sector to trade to" 0
	// getinput $timesLeft "Enter times to execute script" 0  
	// getinput $percent "Enter markup/markdown percentage" 5
	script := `
		getinput $sector2 "Enter sector to trade to" 0
		getinput $timesLeft "Enter times to execute script" 0
		getinput $percent "Enter markup/markdown percentage" 5
		
		# Echo the collected values to verify they were stored
		echo "Sector: " $sector2
		echo "Times: " $timesLeft  
		echo "Percentage: " $percent
	`
	
	// Start script execution
	result := tester.ExecuteScript(script)
	
	// Script should pause at first getinput, not complete or error
	if result.Error != nil {
		t.Fatalf("Script execution failed unexpectedly: %v", result.Error)
	}
	
	// First prompt should be displayed
	expectedFirstOutput := "\r\nEnter sector to trade to [0]\r\n"
	if len(result.Output) < 1 || result.Output[0] != expectedFirstOutput {
		t.Errorf("Expected first prompt %q, got outputs: %v", expectedFirstOutput, result.Output)
	}
	
	// Script should be paused waiting for first input
	if !tester.IsScriptWaitingForInput() {
		t.Fatal("Script should be paused waiting for first input")
	}
	
	// Now test both the ideal behavior (direct VM input) and the real-world behavior
	// by attempting to simulate script restart conditions
	
	// Store the original script VM state
	originalVM := tester.currentScript.VM
	originalPosition := originalVM.GetCurrentPosition()
	t.Logf("Script position before first input: %d", originalPosition)
	
	// Provide first input using direct VM method (current working approach)
	err := tester.ProvideInput("2157")
	if err != nil {
		t.Fatalf("Failed to provide first input: %v", err)
	}
	
	// Continue execution normally
	result = tester.ContinueExecution()
	if result.Error != nil {
		t.Fatalf("Script failed after first input: %v", result.Error)
	}
	
	// Log the current position to see if script advanced
	newPosition := tester.currentScript.VM.GetCurrentPosition()
	t.Logf("Script position after first input: %d", newPosition)
	
	// Check if we're progressing or restarting
	if newPosition <= originalPosition {
		t.Errorf("SCRIPT RESTART DETECTED: Position went from %d to %d after first input", originalPosition, newPosition)
	}
	
	// Second prompt should be displayed
	expectedSecondOutput := "\r\nEnter times to execute script [0]\r\n"
	foundSecond := false
	for _, output := range result.Output {
		if output == expectedSecondOutput {
			foundSecond = true
			break
		}
	}
	if !foundSecond {
		t.Errorf("Expected second prompt %q not found in outputs: %v", expectedSecondOutput, result.Output)
		// This is likely the bug - script restarted instead of continuing to second getinput
		return
	}
	
	// Script should be paused waiting for second input
	if !tester.IsScriptWaitingForInput() {
		t.Fatal("Script should be paused waiting for second input")
	}
	
	// Store position before second input
	secondPosition := tester.currentScript.VM.GetCurrentPosition()
	t.Logf("Script position before second input: %d", secondPosition)
	
	// Provide second input
	err = tester.ProvideInput("5")
	if err != nil {
		t.Fatalf("Failed to provide second input: %v", err)
	}
	
	// Script should continue and pause at third getinput
	result = tester.ContinueExecution()
	if result.Error != nil {
		t.Fatalf("Script failed after second input: %v", result.Error)
	}
	
	// Check position progression again
	thirdPosition := tester.currentScript.VM.GetCurrentPosition()
	t.Logf("Script position after second input: %d", thirdPosition)
	
	if thirdPosition <= secondPosition {
		t.Errorf("SCRIPT RESTART DETECTED: Position went from %d to %d after second input", secondPosition, thirdPosition)
	}
	
	// Third prompt should be displayed
	expectedThirdOutput := "\r\nEnter markup/markdown percentage [5]\r\n"
	foundThird := false
	for _, output := range result.Output {
		if output == expectedThirdOutput {
			foundThird = true
			break
		}
	}
	if !foundThird {
		t.Errorf("Expected third prompt %q not found in outputs: %v", expectedThirdOutput, result.Output)
		return
	}
	
	// Script should be paused waiting for third input
	if !tester.IsScriptWaitingForInput() {
		t.Fatal("Script should be paused waiting for third input")
	}
	
	// Provide third input (use default by providing empty string)
	err = tester.ProvideInput("")
	if err != nil {
		t.Fatalf("Failed to provide third input: %v", err)
	}
	
	// Script should now complete and echo the collected values
	result = tester.ContinueExecution()
	if result.Error != nil {
		t.Fatalf("Script failed after third input: %v", result.Error)
	}
	
	// Verify all variables were stored correctly
	expectedFinalOutputs := []string{
		"Sector: 2157",
		"Times: 5", 
		"Percentage: 5", // Should use default value since we provided empty input
	}
	
	for _, expected := range expectedFinalOutputs {
		found := false
		for _, output := range result.Output {
			if output == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected final output %q not found in outputs: %v", expected, result.Output)
		}
	}
	
	// Script should no longer be waiting for input
	if tester.IsScriptWaitingForInput() {
		t.Error("Script should not be waiting for input after completion")
	}
}

// TestGetInputWithDefaults verifies default value handling in getinput commands
func TestGetInputWithDefaults_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		getinput $value1 "Enter first value" 42
		getinput $value2 "Enter second value" "default_string"
		
		echo "Value1: " $value1
		echo "Value2: " $value2
	`
	
	// Start execution
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Script execution failed: %v", result.Error)
	}
	
	// Provide empty input to use defaults
	if tester.IsScriptWaitingForInput() {
		err := tester.ProvideInput("")
		if err != nil {
			t.Fatalf("Failed to provide first input: %v", err)
		}
		
		result = tester.ContinueExecution()
		if result.Error != nil {
			t.Fatalf("Script failed after first input: %v", result.Error)
		}
	}
	
	if tester.IsScriptWaitingForInput() {
		err := tester.ProvideInput("")
		if err != nil {
			t.Fatalf("Failed to provide second input: %v", err)
		}
		
		result = tester.ContinueExecution()
		if result.Error != nil {
			t.Fatalf("Script failed after second input: %v", result.Error)
		}
	}
	
	// Verify defaults were used
	expectedOutputs := []string{
		"Value1: 42",
		"Value2: default_string",
	}
	
	for _, expected := range expectedOutputs {
		found := false
		for _, output := range result.Output {
			if output == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected output %q not found in outputs: %v", expected, result.Output)
		}
	}
}

// TestGetInputUserOverridesDefault verifies user input overrides defaults
func TestGetInputUserOverridesDefault_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		getinput $override "Enter value" "default"
		echo "Result: " $override
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Script execution failed: %v", result.Error)
	}
	
	// Provide user input instead of using default
	if tester.IsScriptWaitingForInput() {
		err := tester.ProvideInput("user_provided")
		if err != nil {
			t.Fatalf("Failed to provide input: %v", err)
		}
		
		result = tester.ContinueExecution()
		if result.Error != nil {
			t.Fatalf("Script failed after input: %v", result.Error)
		}
	}
	
	// Verify user input was used instead of default
	expected := "Result: user_provided"
	found := false
	for _, output := range result.Output {
		if output == expected {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected output %q not found in outputs: %v", expected, result.Output)
	}
}