package streaming

import (
	"testing"
	"twist/internal/proxy/database"
)

func TestStardockDetection(t *testing.T) {
	// Create test database and parser
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParser(db, nil)

	t.Run("V Screen Stardock Detection", func(t *testing.T) {
		// Test the exact Pascal pattern: Copy(Line, 14, 8) = 'StarDock' and Copy(Line, 37, 6) = 'sector'
		// Sample line: "             StarDock                   sector 1234."
		stardockLine := "             StarDock                   sector 1234."
		
		// Process the line
		parser.ProcessString(stardockLine + "\r")
		
		// Verify Stardock was detected and stored
		stardockSector := parser.getStardockSector()
		if stardockSector != 1234 {
			t.Errorf("Expected Stardock sector 1234, got %d", stardockSector)
		}
		
		// Verify sector was set up correctly
		sector, err := db.LoadSector(1234)
		if err != nil {
			t.Fatalf("Failed to load Stardock sector: %v", err)
		}
		
		// Check Pascal-compliant setup
		if sector.Constellation != "The Federation" {
			t.Errorf("Expected constellation 'The Federation', got '%s'", sector.Constellation)
		}
		if sector.Beacon != "FedSpace, FedLaw Enforced" {
			t.Errorf("Expected beacon 'FedSpace, FedLaw Enforced', got '%s'", sector.Beacon)
		}
		if sector.SPort.Name != "Stargate Alpha I" {
			t.Errorf("Expected port name 'Stargate Alpha I', got '%s'", sector.SPort.Name)
		}
		if sector.SPort.ClassIndex != 9 {
			t.Errorf("Expected port class 9, got %d", sector.SPort.ClassIndex)
		}
		if sector.SPort.Dead != false {
			t.Errorf("Expected port not dead, got %t", sector.SPort.Dead)
		}
		if sector.SPort.BuildTime != 0 {
			t.Errorf("Expected build time 0, got %d", sector.SPort.BuildTime)
		}
		if sector.Explored != database.EtCalc {
			t.Errorf("Expected explored status EtCalc, got %d", sector.Explored)
		}
		
		t.Logf("✓ Stardock detected and set up correctly in sector %d", stardockSector)
	})

	t.Run("Multiple Stardock Detection Prevention", func(t *testing.T) {
		// Reset for clean test
		parser2 := NewTWXParser(db, nil)
		
		// First detection
		stardockLine1 := "             StarDock                   sector 1234."
		parser2.ProcessString(stardockLine1 + "\r")
		
		// Verify first detection
		if parser2.getStardockSector() != 1234 {
			t.Errorf("Expected first Stardock detection to succeed")
		}
		
		// Second detection attempt (should be ignored)
		stardockLine2 := "             StarDock                   sector 5678."
		parser2.ProcessString(stardockLine2 + "\r")
		
		// Should still be the original sector
		if parser2.getStardockSector() != 1234 {
			t.Errorf("Expected Stardock to remain at sector 1234, got %d", parser2.getStardockSector())
		}
		
		// Verify second sector was NOT set up
		sector, err := db.LoadSector(5678)
		if err == nil && sector.SPort.ClassIndex == 9 {
			t.Errorf("Second Stardock should not have been set up")
		}
		
		t.Log("✓ Multiple Stardock detection correctly prevented")
	})

	t.Run("Pattern Position Requirements", func(t *testing.T) {
		// Test lines that don't match the exact position requirements
		invalidLines := []string{
			"StarDock                   sector 1234.",           // Wrong position for StarDock
			"             NotStarDock                   sector 1234.", // Wrong word
			"             StarDock                   port 1234.",      // Wrong word at position 37
			"             StarDock             sector 1234.",          // Wrong position for sector
			"StarDock sector 1234.",                                  // Too short
			"",                                                       // Empty line
		}
		
		// Create fresh parser for clean test
		db2 := database.NewDatabase()
		if err := db2.CreateDatabase(":memory:"); err != nil {
			t.Fatalf("Failed to create test database: %v", err)
		}
		defer db2.CloseDatabase()
		
		parser3 := NewTWXParser(db2, nil)
		
		for _, line := range invalidLines {
			parser3.ProcessString(line + "\r")
			
			// Should not detect Stardock
			if parser3.getStardockSector() != 0 {
				t.Errorf("Line '%s' should not trigger Stardock detection", line)
			}
		}
		
		t.Log("✓ Position requirements correctly enforced")
	})

	t.Run("Sector Number Extraction", func(t *testing.T) {
		// Test different sector number formats
		testCases := []struct {
			line           string
			expectedSector int
			description    string
		}{
			{
				line:           "             StarDock                   sector 1.",
				expectedSector: 1,
				description:    "Single digit sector",
			},
			{
				line:           "             StarDock                   sector 12345.",
				expectedSector: 12345,
				description:    "Five digit sector",
			},
			{
				line:           "             StarDock                   sector 999.",
				expectedSector: 999,
				description:    "Three digit sector",
			},
		}
		
		for _, tc := range testCases {
			// Create fresh parser for each test
			dbTest := database.NewDatabase()
			if err := dbTest.CreateDatabase(":memory:"); err != nil {
				t.Fatalf("Failed to create test database: %v", err)
			}
			
			parserTest := NewTWXParser(dbTest, nil)
			
			// Process the line
			parserTest.ProcessString(tc.line + "\r")
			
			// Check sector detection
			detectedSector := parserTest.getStardockSector()
			if detectedSector != tc.expectedSector {
				t.Errorf("%s: expected sector %d, got %d", tc.description, tc.expectedSector, detectedSector)
			} else {
				t.Logf("✓ %s: correctly detected sector %d", tc.description, detectedSector)
			}
			
			dbTest.CloseDatabase()
		}
	})

	t.Run("Database Integration", func(t *testing.T) {
		// Test without database (should handle gracefully)
		parserNoDB := &TWXParser{}
		
		// Should not crash
		parserNoDB.handleStardockDetection("             StarDock                   sector 1234.")
		
		// Should return 0 for unknown
		if parserNoDB.getStardockSector() != 0 {
			t.Errorf("Parser without database should return 0 for Stardock sector")
		}
		
		t.Log("✓ Graceful handling without database")
	})
}

func TestStardockConfigurationPersistence(t *testing.T) {
	// Create test database and parser
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParser(db, nil)

	t.Run("Configuration Storage and Retrieval", func(t *testing.T) {
		// Set Stardock sector
		parser.setStardockSector(1234)
		
		// Verify retrieval
		retrieved := parser.getStardockSector()
		if retrieved != 1234 {
			t.Errorf("Expected retrieved Stardock sector 1234, got %d", retrieved)
		}
		
		// Create new parser instance with same database
		parser2 := NewTWXParser(db, nil)
		
		// Should retrieve the same value
		retrieved2 := parser2.getStardockSector()
		if retrieved2 != 1234 {
			t.Errorf("Expected persistent Stardock sector 1234, got %d", retrieved2)
		}
		
		t.Log("✓ Stardock configuration persists across parser instances")
	})

	t.Run("Unknown Stardock Handling", func(t *testing.T) {
		// Create fresh database
		db2 := database.NewDatabase()
		if err := db2.CreateDatabase(":memory:"); err != nil {
			t.Fatalf("Failed to create test database: %v", err)
		}
		defer db2.CloseDatabase()

		parser3 := NewTWXParser(db2, nil)
		
		// Should return 0 for unknown Stardock
		unknown := parser3.getStardockSector()
		if unknown != 0 {
			t.Errorf("Expected unknown Stardock to return 0, got %d", unknown)
		}
		
		t.Log("✓ Unknown Stardock correctly returns 0")
	})
}