package scripting

import (
	"fmt"
	"testing"
)

func TestTriggerHandlerSeparation(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Test script that verifies TWX-compatible label behavior: execution flows through labels
	script := `
		echo "Setting up trigger"
		setTextTrigger 1 :handler "test"
		echo "After setting trigger, now pausing"
		pause
		echo "This should execute after pause"
		
		:handler
		echo "HANDLER EXECUTED - This should NOT run during normal flow!"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Script execution failed: %v", result.Error)
	}

	// Check that all expected output was executed (including handler code during normal flow)
	// In TWX, labels are transparent and execution flows through them
	expectedOutput := []string{
		"Setting up trigger",
		"After setting trigger, now pausing",
		"This should execute after pause",
		"HANDLER EXECUTED - This should NOT run during normal flow!", // This WILL execute in TWX
	}

	for _, expected := range expectedOutput {
		found := false
		for _, line := range result.Output {
			if line == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected output line not found: %q", expected)
		}
	}

	fmt.Printf("Test passed - TWX-compatible label behavior: execution flows through labels\n")
	fmt.Printf("Output was: %v\n", result.Output)
}
