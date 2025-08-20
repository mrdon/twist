package scripting

import (
	"testing"
)

// TestEchoStarToNewlineConversion verifies that echo converts * to newlines like TWX
func TestEchoStarToNewlineConversion_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		echo "**     --===| Port Pair Trading v2.00 |===--**"
		echo "For your own safety, please read the warnings*written at the top of the script*before using it!"
		echo "No registration is required*it is open source*can be opened in notepad"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
		return
	}

	// Expected outputs should have * converted to CRLF
	expectedOutputs := []string{
		"\r\n\r\n     --===| Port Pair Trading v2.00 |===--\r\n\r\n",
		"For your own safety, please read the warnings\r\nwritten at the top of the script\r\nbefore using it!",
		"No registration is required\r\nit is open source\r\ncan be opened in notepad",
	}

	if len(result.Output) != len(expectedOutputs) {
		t.Errorf("Expected %d outputs, got %d. Outputs: %v", len(expectedOutputs), len(result.Output), result.Output)
		return
	}

	for i, expected := range expectedOutputs {
		if i >= len(result.Output) {
			t.Errorf("Missing output %d: expected %q", i, expected)
			continue
		}
		if result.Output[i] != expected {
			t.Errorf("Output %d mismatch.\nExpected: %q\nGot: %q", i, expected, result.Output[i])
		}
	}

	// Verify no commands were sent to server
	if len(result.Commands) > 0 {
		t.Errorf("Echo should not send commands to server, but sent: %v", result.Commands)
	}
}

// TestEchoStarConversionWithVariables verifies * to newline conversion works with variables
func TestEchoStarConversionWithVariables_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $banner "**BANNER**"
		setVar $multiline "Line1*Line2*Line3"
		echo $banner
		echo "Text: " $multiline " end"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
		return
	}

	expectedOutputs := []string{
		"\r\n\r\nBANNER\r\n\r\n",
		"Text: Line1\r\nLine2\r\nLine3 end",
	}

	if len(result.Output) != len(expectedOutputs) {
		t.Errorf("Expected %d outputs, got %d. Outputs: %v", len(expectedOutputs), len(result.Output), result.Output)
		return
	}

	for i, expected := range expectedOutputs {
		if i >= len(result.Output) {
			t.Errorf("Missing output %d: expected %q", i, expected)
			continue
		}
		if result.Output[i] != expected {
			t.Errorf("Output %d mismatch.\nExpected: %q\nGot: %q", i, expected, result.Output[i])
		}
	}
}

// TestEchoStarConversionMixedWithActualNewlines tests * conversion alongside real newlines
func TestEchoStarConversionMixedWithActualNewlines_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		echo "Real newline:\nStar newline:*Mixed"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
		return
	}

	expected := "Real newline:\nStar newline:\r\nMixed"

	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output, got %d. Outputs: %v", len(result.Output), result.Output)
		return
	}

	if result.Output[0] != expected {
		t.Errorf("Output mismatch.\nExpected: %q\nGot: %q", expected, result.Output[0])
		// Show character-by-character comparison for debugging
		expectedRunes := []rune(expected)
		actualRunes := []rune(result.Output[0])
		minLen := len(expectedRunes)
		if len(actualRunes) < minLen {
			minLen = len(actualRunes)
		}
		t.Logf("Character-by-character comparison (first %d chars):", minLen)
		for i := 0; i < minLen; i++ {
			t.Logf("  [%d] expected: %q (%d), got: %q (%d)", i, string(expectedRunes[i]), int(expectedRunes[i]), string(actualRunes[i]), int(actualRunes[i]))
		}
		if len(expectedRunes) != len(actualRunes) {
			t.Logf("Length difference: expected %d chars, got %d chars", len(expectedRunes), len(actualRunes))
		}
	}
}

// TestEchoNoStarCharacters verifies normal echo behavior without * characters
func TestEchoNoStarCharacters_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		echo "Normal text without stars"
		echo "Multiple parameters " "joined together"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
		return
	}

	expectedOutputs := []string{
		"Normal text without stars",
		"Multiple parameters joined together",
	}

	if len(result.Output) != len(expectedOutputs) {
		t.Errorf("Expected %d outputs, got %d. Outputs: %v", len(expectedOutputs), len(result.Output), result.Output)
		return
	}

	for i, expected := range expectedOutputs {
		if i >= len(result.Output) {
			t.Errorf("Missing output %d: expected %q", i, expected)
			continue
		}
		if result.Output[i] != expected {
			t.Errorf("Output %d mismatch.\nExpected: %q\nGot: %q", i, expected, result.Output[i])
		}
	}
}

// TestEchoPortPairTradingBanner_RealIntegration tests the specific example from the user's issue
func TestEchoPortPairTradingBanner_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Simulate the actual script output that was problematic
	script := `
		echo "**     --===| Port Pair Trading v2.00 |===--**"
		echo "**For your own safety, please read the warnings*written at the top of the scri"
		echo "before*using it!*"
		echo "No registration is required to use this script,*it is completely open source a"
		echo "can be opened*in notepad."
		echo ""
		echo "Enter sector to trade to [0] "
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
		return
	}

	expectedOutputs := []string{
		"\r\n\r\n     --===| Port Pair Trading v2.00 |===--\r\n\r\n",
		"\r\n\r\nFor your own safety, please read the warnings\r\nwritten at the top of the scri",
		"before\r\nusing it!\r\n",
		"No registration is required to use this script,\r\nit is completely open source a",
		"can be opened\r\nin notepad.",
		"",
		"Enter sector to trade to [0] ",
	}

	if len(result.Output) != len(expectedOutputs) {
		t.Errorf("Expected %d outputs, got %d. Outputs: %v", len(expectedOutputs), len(result.Output), result.Output)
		return
	}

	for i, expected := range expectedOutputs {
		if i >= len(result.Output) {
			t.Errorf("Missing output %d: expected %q", i, expected)
			continue
		}
		if result.Output[i] != expected {
			t.Errorf("Output %d mismatch.\nExpected: %q\nGot: %q", i, expected, result.Output[i])
			// Show detailed character-by-character comparison for debugging
			expectedRunes := []rune(expected)
			actualRunes := []rune(result.Output[i])
			t.Logf("Expected length: %d, Got length: %d", len(expectedRunes), len(actualRunes))
			minLen := len(expectedRunes)
			if len(actualRunes) < minLen {
				minLen = len(actualRunes)
			}
			for j := 0; j < minLen && j < 50; j++ { // Show first 50 chars max
				if expectedRunes[j] != actualRunes[j] {
					t.Logf("  Diff at [%d]: expected %q (%d), got %q (%d)", j, string(expectedRunes[j]), int(expectedRunes[j]), string(actualRunes[j]), int(actualRunes[j]))
				}
			}
		}
	}
}
