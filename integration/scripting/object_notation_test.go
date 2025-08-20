package scripting

import (
	"testing"
	"time"
	"twist/internal/proxy/database"
)

// TestGetSectorDotNotation tests dot notation access with getSector
func TestGetSectorDotNotation_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Create test sector data
	err := tester.setupData.DB.SaveSector(createTestSector(), 100)
	if err != nil {
		t.Fatalf("Failed to save test sector: %v", err)
	}

	// Create test port data
	testPort := database.TPort{
		Name:           "Trading Post Alpha",
		Dead:           false,
		BuildTime:      0,
		ClassIndex:     1,
		BuyProduct:     [3]bool{true, false, true},
		ProductPercent: [3]int{100, 0, 100},
		ProductAmount:  [3]int{500, 0, 300},
		UpDate:         time.Now(),
	}
	err = tester.setupData.DB.SavePort(testPort, 100)
	if err != nil {
		t.Fatalf("Failed to save test port: %v", err)
	}

	script := `
		getSector 100 $sector
		echo "Sector properties:"
		echo "  Index: " $sector.index
		echo "  Density: " $sector.density  
		echo "  Explored: " $sector.explored
		echo "  Beacon: " $sector.beacon
		echo "  Constellation: " $sector.constellation
		echo "  Warps: " $sector.warps
		echo "Port properties:"
		echo "  Exists: " $sector.port.exists
		echo "  Class: " $sector.port.class
		echo "  Name: " $sector.port.name
		echo "Warp array access:"
		echo "  Warp[1]: " $sector.warp[1]
		echo "  Warp[2]: " $sector.warp[2]
		echo "  Warp[3]: " $sector.warp[3]
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Sector properties:",
		"  Index: 100",
		"  Density: 45",
		"  Explored: DENSITY",
		"  Beacon: Test Beacon",
		"  Constellation: Test Constellation",
		"  Warps: 3",
		"Port properties:",
		"  Exists: 1",
		"  Class: 1",
		"  Name: Trading Post Alpha",
		"Warp array access:",
		"  Warp[1]: 2",
		"  Warp[2]: 3",
		"  Warp[3]: 4",
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
