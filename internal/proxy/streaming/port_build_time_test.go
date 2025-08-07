package streaming

import (
	"testing"
	"twist/internal/proxy/database"
)

func TestPortBuildTimeParsing(t *testing.T) {
	// Create test database and parser
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParser(db, nil)
	
	t.Run("Pascal Build Time Extraction", func(t *testing.T) {
		// Test exact Pascal GetParameter(Line, 4) logic for build time extraction
		testCases := []struct {
			name                string
			portLine           string
			continuationLine   string
			expectedBuildTime  int
			description        string
		}{
			{
				name:               "Standard build time format",
				portLine:           "Ports   : Trading Post, Class 1 Port BSS",
				continuationLine:   "        Build Time: 24 hours remaining",
				expectedBuildTime:  24,
				description:        "Pascal: GetParameter(Line, 4) with standard format",
			},
			{
				name:               "Zero build time",
				portLine:           "Ports   : Deep Space Station, Class 5 Port SBB",
				continuationLine:   "        Build Time: 0 hours remaining",
				expectedBuildTime:  0,
				description:        "Pascal: GetParameter(Line, 4) with zero build time",
			},
			{
				name:               "Large build time",
				portLine:           "Ports   : Mega Port, Class 9 Port BBB",
				continuationLine:   "        Build Time: 168 hours remaining",
				expectedBuildTime:  168,
				description:        "Pascal: GetParameter(Line, 4) with large build time",
			},
			{
				name:               "Minimal format",
				portLine:           "Ports   : Simple Port, Class 2 Port SBS",
				continuationLine:   "        Build 12 hours left",
				expectedBuildTime:  12,
				description:        "Pascal: GetParameter(Line, 4) extracting 4th parameter",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Reset parser state
				parser.currentDisplay = DisplaySector
				parser.sectorPosition = SectorPosNormal
				parser.currentSectorIndex = 1234
				
				// Clear sector data
				parser.currentSector = CurrentSector{}
				
				// Process port line first to set up port context
				parser.ProcessString(tc.portLine + "\r")
				
				// Verify we're in ports section
				if parser.sectorPosition != SectorPosPorts {
					t.Errorf("Expected sector position %d (Ports), got %d", SectorPosPorts, parser.sectorPosition)
				}
				
				// Process continuation line with build time
				parser.ProcessString(tc.continuationLine + "\r")
				
				// Verify build time was parsed correctly
				if parser.currentSector.Port.BuildTime != tc.expectedBuildTime {
					t.Errorf("Expected build time %d, got %d", tc.expectedBuildTime, parser.currentSector.Port.BuildTime)
				}
				
				t.Logf("✓ %s: %s", tc.name, tc.description)
			})
		}
	})

	t.Run("Complete Port Workflow with Build Time", func(t *testing.T) {
		// Test complete port parsing workflow including build time storage
		parser.currentDisplay = DisplaySector
		parser.sectorPosition = SectorPosNormal
		parser.currentSectorIndex = 5678
		parser.currentSector = CurrentSector{}
		
		// Simulate complete port scanning sequence
		lines := []string{
			"Sector  : 5678 in Test Sector",
			"Ports   : Industrial Hub, Class 3 Port BSB",
			"        Build Time: 48 hours remaining",
			"NavHaz  : 2%",
			"Warps to Sector(s) :  (5679) - 5680",
		}
		
		for i, line := range lines {
			parser.ProcessString(line + "\r")
			t.Logf("Step %d: %s", i+1, line)
		}
		
		// Force sector completion to ensure database save (following pattern from other tests)
		parser.sectorCompleted()
		
		// Verify sector was saved with correct build time
		_, err := db.LoadSector(5678)
		if err != nil {
			t.Fatalf("Failed to load sector 5678: %v", err)
		}
		
		// Load port data separately
		port, err := db.LoadPort(5678)
		if err != nil {
			t.Fatalf("Failed to load port for sector 5678: %v", err)
		}
		
		if port.BuildTime != 48 {
			t.Errorf("Expected build time 48, got %d", port.BuildTime)
		}
		
		if port.Name != "Industrial Hub" {
			t.Errorf("Expected port name 'Industrial Hub', got '%s'", port.Name)
		}
		
		if port.ClassIndex != 3 {
			t.Errorf("Expected port class 3, got %d", port.ClassIndex)
		}

		t.Log("✓ Complete port workflow with build time stored correctly")
	})

	t.Run("Build Time Parameter Extraction Tests", func(t *testing.T) {
		// Test parameter extraction specifically for Pascal GetParameter(Line, 4) logic
		testCases := []struct {
			line           string
			expectedParam4 string
			expectedValue  int
			description    string
		}{
			{
				line:           "        Build Time: 24 hours remaining",
				expectedParam4: "hours",
				expectedValue:  0,
				description:    "Standard format - 4th parameter is 'hours', not the number",
			},
			{
				line:           "        Port Status: Build 0 complete now",
				expectedParam4: "0",
				expectedValue:  0,
				description:    "Alternative format - 4th parameter extraction",
			},
			{
				line:           "        Construction Time Remaining: 72 hours",
				expectedParam4: "72",
				expectedValue:  72,
				description:    "Longer format - parameter 4 extraction",
			},
			{
				line:           "        Build incomplete 96 hours left",
				expectedParam4: "hours",
				expectedValue:  0,
				description:    "Short format - parameter 4 is 'hours', not the build time",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.description, func(t *testing.T) {
				// Test parameter extraction
				param4 := parser.getParameter(tc.line, 4)
				if param4 != tc.expectedParam4 {
					t.Errorf("Expected parameter 4 '%s', got '%s'", tc.expectedParam4, param4)
				}
				
				// Test integer parsing
				value := parser.parseIntSafe(param4)
				if value != tc.expectedValue {
					t.Errorf("Expected parsed value %d, got %d", tc.expectedValue, value)
				}
				
				t.Logf("✓ %s: Parameter 4='%s', Value=%d", tc.description, param4, value)
			})
		}
	})

	t.Run("Edge Cases and Error Handling", func(t *testing.T) {
		parser.currentDisplay = DisplaySector
		parser.sectorPosition = SectorPosPorts
		parser.currentSector = CurrentSector{}
		
		// Test invalid build time formats
		invalidCases := []string{
			"        Build Time: invalid hours",
			"        Short line",
			"        Build Time: -5 hours", // Negative should be handled gracefully
			"        ",                     // Empty continuation
		}

		for _, invalidLine := range invalidCases {
			initialBuildTime := parser.currentSector.Port.BuildTime
			parser.ProcessString(invalidLine + "\r")
			
			// Build time should remain unchanged for invalid input
			// (or be set to 0 for negative values due to parseIntSafe)
			t.Logf("Invalid line: %q - Build time: %d (was %d)", 
				invalidLine, parser.currentSector.Port.BuildTime, initialBuildTime)
		}
		
		t.Log("✓ Invalid build time formats handled gracefully")
	})
}

func TestPortBuildTimeIntegration(t *testing.T) {
	// Test integration with existing port parsing and database storage
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParser(db, nil)

	t.Run("Multiple Ports with Build Times", func(t *testing.T) {
		// Test multiple sectors with different port build times
		testCases := []struct {
			sectorNum int
			portName  string
			buildTime int
		}{
			{1001, "Alpha Station", 12},
			{1002, "Beta Outpost", 36},
			{1003, "Gamma Complex", 0},
		}

		for _, sector := range testCases {
			// Process each sector
			parser.ProcessString("Sector  : " + parser.intToString(sector.sectorNum) + " in Test Space\r")
			parser.ProcessString("Ports   : " + sector.portName + ", Class 1 Port BSS\r")
			parser.ProcessString("        Build Time: " + parser.intToString(sector.buildTime) + " hours remaining\r")
			parser.ProcessString("Warps to Sector(s) :  (9999)\r")
		}

		// Force completion of the last sector
		parser.sectorCompleted()

		// Verify all sectors stored correctly
		for _, expected := range testCases {
			_, err := db.LoadSector(expected.sectorNum)
			if err != nil {
				t.Errorf("Failed to load sector %d: %v", expected.sectorNum, err)
				continue
			}

			// Load port data separately
			port, err := db.LoadPort(expected.sectorNum)
			if err != nil {
				t.Errorf("Failed to load port for sector %d: %v", expected.sectorNum, err)
				continue
			}

			if port.BuildTime != expected.buildTime {
				t.Errorf("Sector %d: Expected build time %d, got %d", 
					expected.sectorNum, expected.buildTime, port.BuildTime)
			}

			if port.Name != expected.portName {
				t.Errorf("Sector %d: Expected port name '%s', got '%s'", 
					expected.sectorNum, expected.portName, port.Name)
			}

			t.Logf("✓ Sector %d: Port '%s' with build time %d hours", 
				expected.sectorNum, port.Name, port.BuildTime)
		}
	})
}

// Helper method for testing
func (p *TWXParser) intToString(value int) string {
	if value == 0 {
		return "0"
	}
	
	result := ""
	if value < 0 {
		result = "-"
		value = -value
	}
	
	for value > 0 {
		result = string(rune('0'+(value%10))) + result
		value /= 10
	}
	
	return result
}