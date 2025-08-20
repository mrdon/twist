package scripting

import (
	"testing"
)

// TestLoginScriptReproduction tests the exact login script pattern that's failing
func TestLoginScriptReproduction_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Reproduce the exact pattern from login.ts
		LoginName = "mrdon"
		Password = "bob"  
		Game = "a"
		
		echo "LoginName is: " LoginName
		echo "Password is: " Password  
		echo "Game is: " Game
		echo "GAME system constant is: " GAME
		
		# Test the send command like the login script does
		# This is where the issue occurs - send Game on line 19 of login.ts
		send "Game selection: " Game
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	// Print all outputs for debugging
	t.Logf("All script outputs: %v", result.Output)
	t.Logf("All script commands sent: %v", result.Commands)

	// Check that Game variable is "a", not "TradeWars 2002" or "d"
	expectedOutputs := []string{
		"LoginName is: mrdon",
		"Password is: bob",
		"Game is: a",
		"GAME system constant is: a", // User variable should override system constant
	}

	expectedCommands := []string{
		"Game selection: a", // This should send "a", not "d" or "TradeWars 2002"
	}

	// Check outputs
	for i, expected := range expectedOutputs {
		if i >= len(result.Output) {
			t.Errorf("Missing output line %d: expected %q", i, expected)
			continue
		}
		if result.Output[i] != expected {
			t.Errorf("Output line %d: got %q, want %q", i, result.Output[i], expected)
		}
	}

	// Check commands sent
	for i, expected := range expectedCommands {
		if i >= len(result.Commands) {
			t.Errorf("Missing command %d: expected %q", i, expected)
			continue
		}
		if result.Commands[i] != expected {
			t.Errorf("Command %d: got %q, want %q", i, result.Commands[i], expected)
		}
	}
}
