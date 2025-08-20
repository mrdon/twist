package scripting

import (
	"testing"
)

// TestSystemConstants tests CURRENTLINE and other system constants
func TestSystemConstants_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		echo "Testing system constants"
		echo "TRUE constant: " TRUE
		echo "FALSE constant: " FALSE
		echo "VERSION constant: " VERSION
		echo "GAME constant: " GAME
		echo "CURRENTSECTOR: " CURRENTSECTOR
		echo "CURRENTLINE: " CURRENTLINE
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	// Verify we got all the expected outputs
	if len(result.Output) != 7 {
		t.Errorf("Expected 7 output lines, got %d", len(result.Output))
	}

	// Check that system constants are properly resolved
	expectedOutputs := []string{
		"Testing system constants",
		"TRUE constant: 1",
		"FALSE constant: 0",
		"VERSION constant: 3.09",
		"GAME constant: TradeWars 2002",
		"CURRENTSECTOR: 1",
		"CURRENTLINE: Command [TL=00:00:00]:", // Should contain command prompt
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Output line %d: got %q, want %q", i, result.Output[i], expected)
		}
	}
}

// TestSystemConstantsWithExpressions tests system constants in expressions
func TestSystemConstantsWithExpressions_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test system constants in expressions
		if (TRUE = 1)
			echo "TRUE constant works in expressions"
		end
		
		if (FALSE <> 1) 
			echo "FALSE constant works in expressions"
		end
		
		# Test with CURRENTSECTOR constant
		if (CURRENTSECTOR > 0)
			echo "CURRENTSECTOR is positive: " CURRENTSECTOR
		end
		
		# Test string constants
		setVar $gameCheck GAME
		if ($gameCheck = "TradeWars 2002")
			echo "GAME constant matches expected value"
		end
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"FALSE constant works in expressions",
		"GAME constant matches expected value",
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
