package streaming

import (
	"testing"
	"twist/internal/proxy/database"
)

func TestCitadelTreasuryDetection(t *testing.T) {
	// Create test database and parser
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParser(db, nil)

	t.Run("Basic Citadel Treasury Detection", func(t *testing.T) {
		// Test basic citadel treasury detection (Pascal: Copy(Line, 1, 25) = 'Citadel treasury contains')
		testCases := []struct {
			name              string
			input             string
			shouldTrigger     bool
			expectedDisplay   DisplayType
			description       string
		}{
			{
				name:              "Standard treasury line",
				input:             "Citadel treasury contains 1,250,000 credits",
				shouldTrigger:     true,
				expectedDisplay:   DisplayNone,
				description:       "Pascal: Exact match for 'Citadel treasury contains' at start",
			},
			{
				name:              "Treasury with different amount",
				input:             "Citadel treasury contains 500,000 credits",
				shouldTrigger:     true,
				expectedDisplay:   DisplayNone,
				description:       "Different treasury amount should still trigger",
			},
			{
				name:              "Treasury with zero credits",
				input:             "Citadel treasury contains 0 credits",
				shouldTrigger:     true,
				expectedDisplay:   DisplayNone,
				description:       "Zero credits should still trigger detection",
			},
			{
				name:              "Treasury with no amount specified",
				input:             "Citadel treasury contains valuable resources",
				shouldTrigger:     true,
				expectedDisplay:   DisplayNone,
				description:       "Non-standard treasury format should still trigger",
			},
			{
				name:              "Case sensitivity test - should not match",
				input:             "citadel treasury contains 1,000,000 credits",
				shouldTrigger:     false,
				expectedDisplay:   DisplaySector, // Should remain unchanged
				description:       "Pascal string comparison is case-sensitive",
			},
			{
				name:              "Partial match - should not trigger",
				input:             "The Citadel treasury contains wealth beyond measure",
				shouldTrigger:     false,
				expectedDisplay:   DisplaySector, // Should remain unchanged
				description:       "Must match exactly from start of line",
			},
			{
				name:              "Similar but different text",
				input:             "Citadel treasures are kept in the vault",
				shouldTrigger:     false,
				expectedDisplay:   DisplaySector, // Should remain unchanged
				description:       "Different text should not trigger detection",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Reset parser state
				parser.currentDisplay = DisplaySector
				parser.sectorSaved = false
				parser.currentSectorIndex = 1234
				
				// Process the line
				parser.ProcessString(tc.input + "\r")
				
				// Verify display state
				if parser.currentDisplay != tc.expectedDisplay {
					t.Errorf("Expected display %d, got %d", tc.expectedDisplay, parser.currentDisplay)
				}
				
				// If should trigger, verify sector was saved
				if tc.shouldTrigger && !parser.sectorSaved {
					t.Errorf("Expected sector to be saved after citadel treasury detection")
				}
				
				t.Logf("✓ %s: %s", tc.name, tc.description)
			})
		}
	})

	t.Run("Sector Completion Integration", func(t *testing.T) {
		// Test that citadel treasury detection properly saves sector data
		parser.currentDisplay = DisplaySector
		parser.sectorSaved = false
		parser.currentSectorIndex = 5678
		parser.currentSector = CurrentSector{}
		
		// Simulate a complete sector being parsed before citadel treasury
		testSequence := []string{
			"Sector  : 5678 in Citadel System",
			"Planets : Alpha Citadel",
			"Warps to Sector(s) :  (5679) - 5680",
			"Citadel treasury contains 2,500,000 credits",
		}
		
		for i, line := range testSequence {
			parser.ProcessString(line + "\r")
			t.Logf("Step %d: %s", i+1, line)
		}
		
		// Verify sector was saved with citadel planet
		sector, err := db.LoadSector(5678)
		if err != nil {
			t.Fatalf("Failed to load sector 5678: %v", err)
		}
		
		// Verify sector has planet data (citadel should be detected as planet)
		if len(sector.Planets) == 0 {
			t.Errorf("Expected sector to have planet data")
		} else {
			// Check if citadel planet was detected
			foundCitadel := false
			for _, planet := range sector.Planets {
				if planet.Citadel {
					foundCitadel = true
					break
				}
			}
			if !foundCitadel {
				t.Errorf("Expected to find citadel planet in sector data")
			}
		}
		
		// Verify display was reset to None
		if parser.currentDisplay != DisplayNone {
			t.Errorf("Expected display to be None after citadel treasury, got %d", parser.currentDisplay)
		}
		
		// Verify sector was marked as saved
		if !parser.sectorSaved {
			t.Errorf("Expected sector to be saved after citadel treasury detection")
		}
		
		t.Log("✓ Sector completion integration working correctly")
	})

	t.Run("Already Saved Sector Handling", func(t *testing.T) {
		// Test that if sector is already saved, we don't save again
		parser.currentDisplay = DisplaySector
		parser.sectorSaved = true  // Already saved
		parser.currentSectorIndex = 9999
		
		// Mock a way to track if sectorCompleted was called
		originalSectorIndex := parser.currentSectorIndex
		
		// Process citadel treasury line
		parser.ProcessString("Citadel treasury contains 750,000 credits\r")
		
		// Verify display was still reset to None
		if parser.currentDisplay != DisplayNone {
			t.Errorf("Expected display to be None, got %d", parser.currentDisplay)
		}
		
		// Verify sector index unchanged (no new sector completion)
		if parser.currentSectorIndex != originalSectorIndex {
			t.Errorf("Sector index should not change when already saved")
		}
		
		t.Log("✓ Already saved sector handled correctly")
	})
}

func TestCitadelTreasuryEdgeCases(t *testing.T) {
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParser(db, nil)

	t.Run("Multiple Treasury Lines", func(t *testing.T) {
		// Test multiple citadel treasury lines in sequence
		parser.currentDisplay = DisplaySector
		parser.sectorSaved = false
		parser.currentSectorIndex = 1234 // Set valid sector number for test
		
		treasuryLines := []string{
			"Citadel treasury contains 1,000,000 credits",
			"Citadel treasury contains additional resources",
			"Citadel treasury contains rare artifacts",
		}
		
		for i, line := range treasuryLines {
			parser.ProcessString(line + "\r")
			
			// First line should trigger, others should be ignored due to DisplayNone
			if i == 0 {
				if parser.currentDisplay != DisplayNone {
					t.Errorf("First treasury line should set display to None")
				}
				if !parser.sectorSaved {
					t.Errorf("First treasury line should save sector")
				}
			}
		}
		
		t.Log("✓ Multiple treasury lines handled correctly")
	})

	t.Run("Mixed Content Lines", func(t *testing.T) {
		// Test lines that contain the pattern but not at the start
		mixedLines := []string{
			"You notice the Citadel treasury contains wealth",
			"Report: Citadel treasury contains 500,000 credits",
			"  Citadel treasury contains hidden gold", // Leading spaces
		}
		
		for _, line := range mixedLines {
			parser.currentDisplay = DisplaySector
			parser.sectorSaved = false
			
			parser.ProcessString(line + "\r")
			
			// None of these should trigger (must be exact start match)
			if parser.currentDisplay == DisplayNone {
				t.Errorf("Line '%s' should not trigger citadel treasury detection", line)
			}
			if parser.sectorSaved {
				t.Errorf("Line '%s' should not save sector", line)
			}
		}
		
		t.Log("✓ Mixed content lines correctly ignored")
	})

	t.Run("Empty and Malformed Lines", func(t *testing.T) {
		// Test edge cases with empty or malformed lines
		edgeCases := []struct {
			line          string
			shouldTrigger bool
			description   string
		}{
			{"Citadel treasury contains", true, "Missing amount - still matches exact start pattern"},
			{"Citadel treasury contains ", true, "Trailing space - still matches exact start pattern"},
			{"Citadel treasury", false, "Truncated - doesn't match full pattern"},
			{"", false, "Empty line"},
		}
		
		for _, tc := range edgeCases {
			parser.currentDisplay = DisplaySector
			parser.sectorSaved = false
			
			parser.ProcessString(tc.line + "\r")
			
			if tc.shouldTrigger {
				if parser.currentDisplay != DisplayNone {
					t.Errorf("Line '%s' should trigger citadel treasury detection", tc.line)
				}
			} else {
				if parser.currentDisplay == DisplayNone {
					t.Errorf("Line '%s' should not trigger citadel treasury detection", tc.line)
				}
			}
			
			t.Logf("✓ %s: %s", tc.line, tc.description)
		}
		
		t.Log("✓ Edge cases handled correctly")
	})
}

func TestCitadelTreasuryIntegration(t *testing.T) {
	// Test integration with full game session workflow
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParser(db, nil)

	t.Run("Complete Citadel Visit Workflow", func(t *testing.T) {
		// Simulate a complete workflow of visiting a citadel planet
		sessionLines := []string{
			"Command [TL=30] (? for help): ",
			"Enter course, or port ('?' for commands): ",
			"",
			"Sector  : 1 in Sol System",
			"Planets : Terra Citadel",
			"Warps to Sector(s) :  (2) - 3 - 4",
			"",
			"Command [TL=29] (? for help): ",
			"Enter course, or port ('?' for commands): ",
			"Landing on planet...",
			"",
			"Welcome to Terra Citadel!",
			"This mighty fortress guards the gateway to Sol.",
			"",
			"Citadel treasury contains 5,000,000 credits and rare artifacts",
			"Your contribution to the defense fund is appreciated.",
			"Command [TL=29] (? for help): ",
		}

		for i, line := range sessionLines {
			parser.ProcessString(line + "\r")
			t.Logf("Step %d: %s", i+1, line)
		}

		// Verify sector 1 was saved with citadel planet
		sector, err := db.LoadSector(1)
		if err != nil {
			t.Fatalf("Failed to load sector 1: %v", err)
		}

		// Check citadel planet was detected
		foundCitadel := false
		for _, planet := range sector.Planets {
			if planet.Citadel {
				foundCitadel = true
				t.Logf("Found citadel planet: %s", planet.Name)
				break
			}
		}
		if !foundCitadel {
			t.Errorf("Expected to find citadel planet in sector 1")
		}

		// Verify display state is None after treasury detection
		if parser.currentDisplay != DisplayNone {
			t.Errorf("Expected display to be None after citadel treasury, got %d", parser.currentDisplay)
		}

		t.Log("✓ Complete citadel visit workflow successful")
	})

	t.Run("Multiple Citadels in Session", func(t *testing.T) {
		// Test visiting multiple citadels in the same session
		citadelSectors := []struct {
			sectorNum int
			name      string
		}{
			{100, "Alpha Citadel"},
			{200, "Beta Citadel Fortress"},
			{300, "Gamma Citadel Stronghold"},
		}

		for _, citadel := range citadelSectors {
			// Reset for new sector
			parser.currentDisplay = DisplaySector
			parser.sectorSaved = false
			
			// Process sector with citadel
			parser.ProcessString("Sector  : " + parser.intToString(citadel.sectorNum) + " in Citadel Space\r")
			parser.ProcessString("Planets : " + citadel.name + "\r")
			parser.ProcessString("Warps to Sector(s) :  (999)\r")
			parser.ProcessString("Citadel treasury contains vast wealth\r")
			
			// Verify sector was saved
			sector, err := db.LoadSector(citadel.sectorNum)
			if err != nil {
				t.Errorf("Failed to load sector %d: %v", citadel.sectorNum, err)
				continue
			}
			
			// Verify citadel planet was detected
			foundCitadel := false
			for _, planet := range sector.Planets {
				if planet.Citadel {
					foundCitadel = true
					break
				}
			}
			if !foundCitadel {
				t.Errorf("Expected to find citadel planet in sector %d", citadel.sectorNum)
			}
			
			t.Logf("✓ Citadel %d (%s) processed correctly", citadel.sectorNum, citadel.name)
		}

		t.Log("✓ Multiple citadels in session handled correctly")
	})
}

