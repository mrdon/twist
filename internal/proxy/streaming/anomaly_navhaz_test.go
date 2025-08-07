package streaming

import (
	"testing"
	"twist/internal/proxy/database"
)

func TestNavHazParsingInAllModes(t *testing.T) {
	// Create test database and parser
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParser(db, nil)

	t.Run("NavHaz Parsing in Sector Display", func(t *testing.T) {
		// Test NavHaz parsing in normal sector display mode
		parser.currentDisplay = DisplaySector
		parser.sectorPosition = SectorPosNormal
		parser.currentSectorIndex = 1234
		parser.currentSector = CurrentSector{}
		
		// Process sector with NavHaz
		testSequence := []string{
			"Sector  : 1234 in Test System",
			"NavHaz  : 15%",
			"Warps to Sector(s) :  (2) - 3",
		}
		
		for i, line := range testSequence {
			parser.ProcessString(line + "\r")
			t.Logf("Step %d: %s", i+1, line)
		}
		
		// Force sector completion to ensure database save
		if !parser.sectorSaved {
			parser.sectorCompleted()
		}
		
		// Verify NavHaz was stored
		if parser.currentSector.NavHaz != 15 {
			t.Errorf("Expected NavHaz 15%%, got %d%%", parser.currentSector.NavHaz)
		}
		
		// Verify sector saved to database with NavHaz
		sector, err := db.LoadSector(1234)
		if err != nil {
			t.Fatalf("Failed to load sector 1234: %v", err)
		}
		
		if sector.NavHaz != 15 {
			t.Errorf("Expected saved NavHaz 15%%, got %d%%", sector.NavHaz)
		}
		
		t.Log("✓ NavHaz parsing in sector display working correctly")
	})

	t.Run("NavHaz Format Variations", func(t *testing.T) {
		// Test various NavHaz format variations
		testCases := []struct {
			name           string
			navHazLine     string
			expectedNavHaz int
			description    string
		}{
			{
				name:           "Standard format",
				navHazLine:     "NavHaz  : 5%",
				expectedNavHaz: 5,
				description:    "Standard NavHaz format",
			},
			{
				name:           "With count in parentheses",
				navHazLine:     "NavHaz  : 10% (25)",
				expectedNavHaz: 10,
				description:    "NavHaz with count in parentheses",
			},
			{
				name:           "Zero NavHaz",
				navHazLine:     "NavHaz  : 0%",
				expectedNavHaz: 0,
				description:    "Zero NavHaz",
			},
			{
				name:           "High NavHaz",
				navHazLine:     "NavHaz  : 100%",
				expectedNavHaz: 100,
				description:    "Maximum NavHaz",
			},
			{
				name:           "With extra spaces",
				navHazLine:     "NavHaz  :   25%   ",
				expectedNavHaz: 25,
				description:    "NavHaz with extra spaces",
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Reset parser state completely
				parser.currentDisplay = DisplaySector
				parser.sectorPosition = SectorPosNormal
				parser.currentSectorIndex = 5000 + len(tc.name) // Unique sector
				parser.currentSector = CurrentSector{}
				parser.sectorSaved = false // Reset the saved flag
				
				// Process sector with NavHaz
				parser.ProcessString("Sector  : " + parser.intToString(parser.currentSectorIndex) + " in NavHaz Test\r")
				parser.ProcessString(tc.navHazLine + "\r")
				parser.ProcessString("Warps to Sector(s) :  (999)\r")
				
				// Force sector completion to ensure database save
				if !parser.sectorSaved {
					parser.sectorCompleted()
				}
				
				// Verify NavHaz was parsed correctly
				if parser.currentSector.NavHaz != tc.expectedNavHaz {
					t.Errorf("Expected NavHaz %d%%, got %d%%", tc.expectedNavHaz, parser.currentSector.NavHaz)
				}
				
				// Verify persistence to database
				sector, err := db.LoadSector(parser.currentSectorIndex)
				if err != nil {
					t.Fatalf("Failed to load sector %d: %v", parser.currentSectorIndex, err)
				}
				
				if sector.NavHaz != tc.expectedNavHaz {
					t.Errorf("Expected saved NavHaz %d%%, got %d%%", tc.expectedNavHaz, sector.NavHaz)
				}
				
				t.Logf("✓ %s: %s", tc.name, tc.description)
			})
		}
	})

	t.Run("NavHaz in Different Display Modes", func(t *testing.T) {
		// Test that NavHaz parsing works regardless of display mode
		displayModes := []struct {
			mode        DisplayType
			description string
		}{
			{DisplaySector, "Normal sector display"},
			{DisplayDensity, "Density scanner mode"},
			{DisplayCIM, "CIM mode"},
		}
		
		for _, dm := range displayModes {
			t.Run(dm.description, func(t *testing.T) {
				parser.currentDisplay = dm.mode
				parser.sectorPosition = SectorPosNormal
				parser.currentSectorIndex = 6000 + int(dm.mode)
				parser.currentSector = CurrentSector{}
				parser.sectorSaved = false // Reset the saved flag
				
				// Process NavHaz line
				parser.ProcessString("Sector  : " + parser.intToString(parser.currentSectorIndex) + " in Mode Test\r")
				parser.ProcessString("NavHaz  : 20%\r")
				parser.ProcessString("Warps to Sector(s) :  (999)\r")
				
				// Verify NavHaz was stored
				if parser.currentSector.NavHaz != 20 {
					t.Errorf("Expected NavHaz 20%% in %s, got %d%%", dm.description, parser.currentSector.NavHaz)
				}
				
				t.Logf("✓ NavHaz parsing works in %s", dm.description)
			})
		}
	})
}

func TestAnomalyDetectionValidation(t *testing.T) {
	// Test that anomaly detection only works through density scanner as per Pascal
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParser(db, nil)

	t.Run("Anomaly Only From Density Scanner", func(t *testing.T) {
		// Test that anomalies are only detected through density scanner mode
		// This validates that our implementation matches Pascal behavior
		
		// Process density scan with anomaly
		parser.ProcessString("Relative Density Scan\r")
		parser.ProcessString("Sector 7777 (Anomaly Test) Density: 1,500, NavHaz: 5%, Warps: 6, Anomaly: Yes\r")
		
		// Verify anomaly was detected from density scan
		sector, err := db.LoadSector(7777)
		if err != nil {
			t.Fatalf("Failed to load sector 7777: %v", err)
		}
		
		if !sector.Anomaly {
			t.Error("Expected anomaly to be detected from density scan")
		}
		
		t.Log("✓ Anomaly correctly detected from density scanner")
	})

	t.Run("No Anomaly From Regular Sector Display", func(t *testing.T) {
		// Test that regular sector display doesn't set anomaly flags
		// This validates Pascal behavior where anomalies are only from density scans
		
		parser.currentDisplay = DisplaySector
		parser.sectorPosition = SectorPosNormal
		parser.currentSectorIndex = 8888
		parser.currentSector = CurrentSector{}
		
		// Process regular sector - no anomaly detection should occur
		testSequence := []string{
			"Sector  : 8888 in Regular Test",
			"NavHaz  : 10%",
			"Warps to Sector(s) :  (999)",
		}
		
		for _, line := range testSequence {
			parser.ProcessString(line + "\r")
		}
		
		// Force sector completion to ensure database save
		if !parser.sectorSaved {
			parser.sectorCompleted()
		}
		
		// Verify no anomaly was set (should remain false by default)
		sector, err := db.LoadSector(8888)
		if err != nil {
			t.Fatalf("Failed to load sector 8888: %v", err)
		}
		
		if sector.Anomaly {
			t.Error("Anomaly should not be set from regular sector display")
		}
		
		// But NavHaz should still be set
		if sector.NavHaz != 10 {
			t.Errorf("Expected NavHaz 10%%, got %d%%", sector.NavHaz)
		}
		
		t.Log("✓ No anomaly detection in regular sector display (Pascal compliant)")
	})
}

func TestNavHazErrorRecovery(t *testing.T) {
	// Test error recovery and graceful handling of malformed NavHaz data
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParser(db, nil)

	t.Run("Malformed NavHaz Data", func(t *testing.T) {
		// Test graceful handling of invalid NavHaz formats
		malformedCases := []struct {
			name        string
			navHazLine  string
			expectedVal int
			description string
		}{
			{
				name:        "Missing percentage sign",
				navHazLine:  "NavHaz  : 15",
				expectedVal: 0, // Should fail to parse and remain 0
				description: "NavHaz without % sign should not parse",
			},
			{
				name:        "Non-numeric value",
				navHazLine:  "NavHaz  : high%",
				expectedVal: 0, // Should fail to parse
				description: "Non-numeric NavHaz should not parse",
			},
			{
				name:        "Empty value",
				navHazLine:  "NavHaz  : %",
				expectedVal: 0,
				description: "Empty NavHaz value should default to 0",
			},
			{
				name:        "Negative value",
				navHazLine:  "NavHaz  : -5%",
				expectedVal: 0, // parseIntSafe should return 0 for negative
				description: "Negative NavHaz should be handled gracefully",
			},
		}
		
		for _, tc := range malformedCases {
			t.Run(tc.name, func(t *testing.T) {
				parser.currentDisplay = DisplaySector
				parser.sectorPosition = SectorPosNormal
				parser.currentSectorIndex = 9000 + len(tc.name)
				parser.currentSector = CurrentSector{}
				
				// Process sector with malformed NavHaz
				parser.ProcessString("Sector  : " + parser.intToString(parser.currentSectorIndex) + " in Error Test\r")
				parser.ProcessString(tc.navHazLine + "\r")
				parser.ProcessString("Warps to Sector(s) :  (999)\r")
				
				// Verify graceful handling
				if parser.currentSector.NavHaz != tc.expectedVal {
					t.Errorf("Expected NavHaz %d for malformed input, got %d", tc.expectedVal, parser.currentSector.NavHaz)
				}
				
				// Verify database persistence
				sector, err := db.LoadSector(parser.currentSectorIndex)
				if err != nil {
					t.Fatalf("Failed to load sector %d: %v", parser.currentSectorIndex, err)
				}
				
				if sector.NavHaz != tc.expectedVal {
					t.Errorf("Expected saved NavHaz %d, got %d", tc.expectedVal, sector.NavHaz)
				}
				
				t.Logf("✓ %s: %s", tc.name, tc.description)
			})
		}
	})

	t.Run("Missing NavHaz Data", func(t *testing.T) {
		// Test sectors without NavHaz data
		parser.currentDisplay = DisplaySector
		parser.sectorPosition = SectorPosNormal
		parser.currentSectorIndex = 9999
		parser.currentSector = CurrentSector{}
		
		// Process sector without NavHaz line
		testSequence := []string{
			"Sector  : 9999 in No NavHaz Test",
			"Ports   : Test Port, Class 1 Port BSS",
			"Warps to Sector(s) :  (1) - 2",
		}
		
		for _, line := range testSequence {
			parser.ProcessString(line + "\r")
		}
		
		// Verify NavHaz defaults to 0
		if parser.currentSector.NavHaz != 0 {
			t.Errorf("Expected default NavHaz 0, got %d", parser.currentSector.NavHaz)
		}
		
		// Verify database persistence
		sector, err := db.LoadSector(9999)
		if err != nil {
			t.Fatalf("Failed to load sector 9999: %v", err)
		}
		
		if sector.NavHaz != 0 {
			t.Errorf("Expected saved NavHaz 0, got %d", sector.NavHaz)
		}
		
		t.Log("✓ Missing NavHaz data handled gracefully")
	})
}

