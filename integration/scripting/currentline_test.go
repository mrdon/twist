package scripting

import (
	"strings"
	"testing"
)

// TestCurrentLineSystemConstant_RealIntegration tests CURRENTLINE system constant behavior like TWX
func TestCurrentLineSystemConstant_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test CURRENTLINE system constant behavior
		echo "Testing CURRENTLINE system constant"
		echo "CURRENTLINE value: [" CURRENTLINE "]"
		
		# Test the cutText operation like 1_Port.ts does
		cutText CURRENTLINE $location 1 12
		echo "First 12 chars: [" $location "]"
		
		# Test the condition check
		if ($location <> "Command [TL=")
			echo "Location check FAILED - expected 'Command [TL=' but got: " $location
		else
			echo "Location check PASSED"
		end
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	// Check that script executed and displayed outputs
	expectedOutputs := []string{
		"Testing CURRENTLINE system constant",
		"CURRENTLINE value:",
		"First 12 chars:",
		"Location check",
	}

	for _, expected := range expectedOutputs {
		found := false
		for _, output := range result.Output {
			if strings.Contains(output, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find %q in outputs: %v", expected, result.Output)
		}
	}

	// Check that CURRENTLINE contains the expected command prompt format
	foundCurrentLine := false
	foundLocationCheck := false
	
	for _, output := range result.Output {
		if strings.Contains(output, "CURRENTLINE value:") {
			foundCurrentLine = true
			// Should contain a command prompt
			if !strings.Contains(output, "Command") {
				t.Errorf("CURRENTLINE should contain 'Command' prompt, got: %s", output)
			}
		}
		if strings.Contains(output, "Location check PASSED") {
			foundLocationCheck = true
		}
		if strings.Contains(output, "Location check FAILED") {
			t.Errorf("Location check failed unexpectedly: %s", output)
		}
	}

	if !foundCurrentLine {
		t.Error("Should have found CURRENTLINE value in output")
	}
	
	// For now, don't require location check to pass since we need to fix CURRENTLINE first
	t.Logf("Location check passed: %t", foundLocationCheck)
}

// TestPortScriptLocationCheck_RealIntegration tests the specific condition from 1_Port.ts
func TestPortScriptLocationCheck_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// This is the exact check from 1_Port.ts that's failing
	script := `
		# Replicate the 1_Port.ts location check
		cutText CURRENTLINE $location 1 12
		
		if ($location <> "Command [TL=")
			clientMessage "This script must be run from the command prompt"
			echo "FAILED: location is [" $location "] instead of [Command [TL=]"
			halt
		end
		
		# If we get here, the check passed
		echo "Location check passed - showing banner"
		echo "**     --===| Port Pair Trading v2.00 |===--**"
		echo "Script would continue normally"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	// Check if the location check passed
	locationCheckPassed := false
	locationCheckFailed := false
	
	for _, output := range result.Output {
		if strings.Contains(output, "Location check passed") {
			locationCheckPassed = true
		}
		if strings.Contains(output, "FAILED: location is") {
			locationCheckFailed = true
			t.Logf("Location check failed with output: %s", output)
		}
		if strings.Contains(output, "Port Pair Trading v2.00") {
			locationCheckPassed = true // Banner indicates success
		}
	}

	// For debugging - log all outputs
	t.Logf("All script outputs: %v", result.Output)

	if locationCheckFailed {
		t.Error("Location check failed - CURRENTLINE does not contain expected command prompt")
	}

	if !locationCheckPassed {
		t.Error("Expected location check to pass and show banner, but it didn't")
	}
}

// TestGetInputWithCurrentLineWorking_RealIntegration tests getinput after fixing CURRENTLINE
func TestGetInputWithCurrentLineWorking_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Test a simplified version of what 1_Port.ts should do
	script := `
		# Location check (should pass)
		cutText CURRENTLINE $location 1 12
		if ($location <> "Command [TL=")
			clientMessage "This script must be run from the command prompt"
			halt
		end
		
		# Show banner
		echo "Port Pair Trading Script"
		echo "Starting configuration..."
		
		# Get input (should pause script execution)
		getinput $sector2 "Enter sector to trade to" 0
		
		# This should not be reached due to getinput pausing
		echo "This line should not appear - script should pause at getinput"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	// Should show banner and getinput prompt
	bannerFound := false
	promptFound := false
	shouldNotAppear := false

	for _, output := range result.Output {
		if strings.Contains(output, "Port Pair Trading Script") {
			bannerFound = true
		}
		if strings.Contains(output, "Enter sector to trade to") {
			promptFound = true
		}
		if strings.Contains(output, "This line should not appear") {
			shouldNotAppear = true
		}
	}

	if !bannerFound {
		t.Error("Expected to see script banner - indicates location check failed")
	}

	if !promptFound {
		t.Error("Expected to see getinput prompt")
	}

	if shouldNotAppear {
		t.Error("Script should have paused at getinput, but continued executing")
	}

	t.Logf("Script outputs: %v", result.Output)
}