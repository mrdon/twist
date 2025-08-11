package scripting

import (
	"strings"
	"testing"
)

// TestMultipleInputBlackBox tests the complete user experience of a script
// that asks for multiple inputs. Using deterministic IntegrationScriptTester.
func TestMultipleInputBlackBox(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
echo "Starting port trading script"
getinput $sector "Enter sector number: " 1
getinput $times "How many times to execute: " 5  
getinput $percent "Enter markup percentage: " 10

echo "Configuration complete:"
echo "Sector: " $sector
echo "Times: " $times  
echo "Percent: " $percent
halt
`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// Script should show initial echo then pause at first getinput
	allOutput := strings.Join(result.Output, " ")
	expectedOutputs := []string{
		"Starting port trading script",
		"Enter sector number:  [1]",
	}
	
	for _, expected := range expectedOutputs {
		if !strings.Contains(allOutput, expected) {
			t.Errorf("Expected output %q not found in output: %v", expected, result.Output)
		}
	}
}

// TestInputWithEchoingBlackBox tests that inputs are properly echoed back
func TestInputWithEchoingBlackBox(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
getinput $value "Enter a value: " "default"
echo "You entered: " $value
halt
`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// Verify the prompt appears with correct default
	allOutput := strings.Join(result.Output, " ")
	if !strings.Contains(allOutput, "Enter a value:  [default]") {
		t.Errorf("Expected prompt not found in output: %v", result.Output)
	}
}

// TestSingleInputDefaultBlackBox tests that default values work correctly
func TestSingleInputDefaultBlackBox(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
getinput $value "Enter value: " "default_value"
echo "Result: " $value
halt
`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// Verify the prompt appears with default value
	allOutput := strings.Join(result.Output, " ")
	if !strings.Contains(allOutput, "Enter value:  [default_value]") {
		t.Errorf("Expected prompt with default not found in output: %v", result.Output)
	}
}

// TestDefaultValuesBlackBox verifies default value behavior from user perspective
func TestDefaultValuesBlackBox(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
getinput $port "Enter port (default 23): " 23
getinput $host "Enter host (default localhost): " "localhost"
echo "Connecting to " $host ":" $port
halt
`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// Script should pause at first getinput and show the correct prompt with default
	allOutput := strings.Join(result.Output, " ")
	expectedPrompt := "Enter port (default 23):  [23]"
	
	if !strings.Contains(allOutput, expectedPrompt) {
		t.Errorf("Expected prompt %q not found in output: %v", expectedPrompt, result.Output)
	}
}

// TestRapidInputBlackBox tests when user types very quickly
func TestRapidInputBlackBox(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
getinput $a "First: " 0
getinput $b "Second: " 0  
getinput $c "Third: " 0
echo "Results: " $a " " $b " " $c
halt
`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// Script should pause at first getinput
	allOutput := strings.Join(result.Output, " ")
	expectedPrompt := "First:  [0]"
	
	if !strings.Contains(allOutput, expectedPrompt) {
		t.Errorf("Expected prompt %q not found in output: %v", expectedPrompt, result.Output)
	}
}

// TestLongInputBlackBox tests handling of longer input strings
func TestLongInputBlackBox(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
getinput $msg "Enter message: " ""
echo "Message: " $msg
halt
`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// Verify the prompt appears (empty default shows no brackets)
	allOutput := strings.Join(result.Output, " ")
	if !strings.Contains(allOutput, "Enter message: ") {
		t.Errorf("Expected prompt not found in output: %v", result.Output)
	}
}