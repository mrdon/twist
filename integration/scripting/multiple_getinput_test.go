package scripting

import (
	"testing"
	"time"
)

// TestMultipleConsecutiveGetInput verifies that multiple getinput commands work correctly
// This replicates the Port Pair Trading script issue where script restarts after each input
// Now using the dual expect system for black-box testing
func TestMultipleConsecutiveGetInput_RealIntegration(t *testing.T) {
	bridge := NewExpectTelnetBridge(t).
		SetupDatabase().
		SetupTelnetServer()

	// Minimal server script - just provides basic game server simulation
	serverScript := `
log "Minimal server starting"
timeout "1s"

# Basic game server greeting
send "Trade Wars 2002\r\nConnected.\r\n"
send "Welcome to the game!\r\n"

log "Minimal server completed"
`

	// The TWX script that has the multiple getinput commands
	script := `
		getinput $sector2 "Enter sector to trade to" 0
		getinput $timesLeft "Enter times to execute script" 0
		getinput $percent "Enter markup/markdown percentage" 5
		
		# Echo the collected values to verify they were stored
		echo "Sector: " $sector2
		echo "Times: " $timesLeft  
		echo "Percentage: " $percent
	`

	bridge.SetServerScript(serverScript).
		SetupProxy().
		SetupExpectEngine().
		LoadScript(script)

	time.Sleep(300 * time.Millisecond)

	// Client-side expect script that tests the real user experience
	// Now that the bug is fixed, script properly progresses through all getinput commands
	clientScript := `
log "Multiple getinput client starting"
timeout "1s"

# Wait for first prompt and respond
expect "Enter sector to trade to"
send "2157*"

# Wait for second prompt and respond
expect "Enter times to execute script"
send "5*"

# Wait for third prompt and use default (just press Enter)
expect "Enter markup/markdown percentage"
send "*"

# Verify the script echoed the results correctly
expect "Sector: 2157"
expect "Times: 5"
expect "Percentage: 5"

log "Multiple getinput test completed successfully!"
`

	err := bridge.RunExpectScript(clientScript)
	if err != nil {
		t.Fatalf("Multiple getinput client script failed: %v", err)
	}

	err = bridge.WaitForServerScript(1 * time.Second)
	if err != nil {
		t.Fatalf("Multiple getinput server script failed: %v", err)
	}

	t.Log("Multiple consecutive getinput test passed - all three inputs processed correctly!")
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