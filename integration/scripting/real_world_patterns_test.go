

package scripting

import (
	"strings"
	"testing"
	"time"
	"twist/internal/proxy/database"
)

// TestComplexSSTPattern tests a pattern like 1_SST.ts would use
func TestComplexSSTPattern_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Create test sectors for the pattern
	err := tester.setupData.DB.SaveSector(createTestSectorWithData(50, true, 2), 10)
	if err != nil {
		t.Fatalf("Failed to save test sector: %v", err)
	}

	// Create test port data in database
	testPort := database.TPort{
		Name:           "Trading Post Alpha",
		Dead:           false,
		BuildTime:      0,
		ClassIndex:     2, // Port class 2 as specified
		BuyProduct:     [3]bool{true, false, true},
		ProductPercent: [3]int{100, 0, 100},
		ProductAmount:  [3]int{500, 0, 300},
		UpDate:         time.Now(),
	}
	err = tester.setupData.DB.SavePort(testPort, 10)
	if err != nil {
		t.Fatalf("Failed to save test port: %v", err)
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

	// Script should execute up to getInput command and then pause
	// This is correct TWX behavior - script waits for user input
	expectedOutputs := []string{
		"Starting SST-like pattern test",
		"Location check passed",
	}

	// Check that basic outputs are present
	outputCount := 0
	for _, expected := range expectedOutputs {
		found := false
		for _, output := range result.Output {
			if strings.Contains(output, expected) {
				found = true
				break
			}
		}
		if found {
			outputCount++
		}
	}

	if outputCount != len(expectedOutputs) {
		t.Errorf("Expected to find %d basic outputs, found %d in: %v", len(expectedOutputs), outputCount, result.Output)
	}

	// Check that getInput prompt is displayed
	expectedPrompt := "Enter this ship's ID [0]"
	found := false
	for _, output := range result.Output {
		if strings.Contains(output, expectedPrompt) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected prompt containing %q, got outputs: %v", expectedPrompt, result.Output)
	}
}