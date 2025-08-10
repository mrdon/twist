package scripting

import (
	"testing"
)

// TestGameVariableOverride tests that user variables can override system constants
func TestGameVariableOverride_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		echo "Initial GAME constant: " GAME
		Game = "a"
		echo "After Game = a, GAME is: " GAME
		echo "After Game = a, Game is: " Game
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	// Print all outputs for debugging
	t.Logf("Script outputs: %v", result.Output)

	// Expect that user variable Game="a" overrides system constant GAME="TradeWars 2002"
	expectedOutputs := []string{
		"Initial GAME constant: TradeWars 2002",
		"After Game = a, GAME is: a",       // User variable should override
		"After Game = a, Game is: a",       // Same variable, different case
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