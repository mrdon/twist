//go:build integration

package scripting

import (
	"testing"
)

// TestComplexSSTPattern tests a pattern like 1_SST.ts would use
func TestComplexSSTPattern_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Create test sectors for the pattern
	err := tester.setupData.DB.SaveSector(createTestSectorWithData(50, true, 2), 10)
	if err != nil {
		t.Fatalf("Failed to save test sector: %v", err)
	}

	script := `
		# Pattern similar to 1_SST.ts script
		echo "Starting SST-like pattern test"
		
		# Check current location (using system constant)
		setVar $location "Command"
		if ($location <> "Command")
			clientMessage "This script must be run from the game command menu"
			halt
		end
		echo "Location check passed"
		
		# Get user input (simulated)
		getInput $shipNumber1 "Enter this ship's ID" 0
		echo "Ship number: " $shipNumber1
		
		# Get sector information  
		getSector 10 $sector1
		echo "Sector analysis:"
		echo "  Sector: " $sector1.index
		echo "  Density: " $sector1.density
		echo "  Port exists: " $sector1.port.exists
		echo "  Port class: " $sector1.port.class
		
		# Make decision based on sector data
		if ($sector1.port.exists = 1) and ($sector1.density < 100)
			echo "Good trading sector found"
		else
			echo "Poor trading sector"
		end
		
		echo "SST pattern test completed"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Starting SST-like pattern test",
		"Location check passed",
		"Ship number: ",  // Empty from getInput in test mode
		"Sector analysis:",
		"  Sector: 10",
		"  Density: 50",
		"  Port exists: 1",
		"  Port class: 2",
		"Good trading sector found",
		"SST pattern test completed",
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