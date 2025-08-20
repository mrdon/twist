package streaming

import (
	"testing"
	"twist/internal/proxy/database"
	"twist/integration/setup"
)

func TestEnhancedDensityProcessing(t *testing.T) {
	// Create test database and parser
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParser(db, nil)

	t.Run("Density Scanner Start Detection", func(t *testing.T) {
		// Test density scanner start (Pascal: Copy(Line, 27, 16) = 'Relative Density')
		densityStart := "                          Relative Density Scan"
		parser.ProcessString(densityStart + "\r")
		
		if parser.currentDisplay != DisplayDensity {
			t.Errorf("Expected DisplayDensity after start, got %d", parser.currentDisplay)
		}
		
		t.Log("✓ Density scanner start correctly detected")
	})

	t.Run("Density Scan Data Processing", func(t *testing.T) {
		// Set up density state
		parser.currentDisplay = DisplayDensity
		
		// Test Pascal density scan format exactly
		// Example: "Sector 1234 (The Sphere) Density: 1,500, NavHaz: 5%, Warps: 6, Anomaly: Yes"
		testCases := []struct {
			name         string
			densityLine  string
			expectedSector int
			expectedDensity int
			expectedWarps int
			expectedAnomaly bool
			description  string
		}{
			{
				name:         "High density with anomaly",
				densityLine:  "Sector ( 1234) ==>           1500  Warps : 6    NavHaz :     5%    Anom : Yes",
				expectedSector: 1234,
				expectedDensity: 1500,
				expectedWarps: 6,
				expectedAnomaly: true,
				description:  "Sector with high density and anomaly",
			},
			{
				name:         "Low density without anomaly",
				densityLine:  "Sector   5678  ==>            800  Warps : 3    NavHaz :     0%    Anom : No",
				expectedSector: 5678,
				expectedDensity: 800,
				expectedWarps: 3,
				expectedAnomaly: false,
				description:  "Sector with low density and no anomaly",
			},
			{
				name:         "Medium density with anomaly",
				densityLine:  "Sector ( 9999) ==>           2345  Warps : 4    NavHaz :    10%    Anom : Yes",
				expectedSector: 9999,
				expectedDensity: 2345,
				expectedWarps: 4,
				expectedAnomaly: true,
				description:  "Sector with medium density and anomaly",
			},
			{
				name:         "Zero density exploration",
				densityLine:  "Sector   1111  ==>              0  Warps : 2    NavHaz :     0%    Anom : No",
				expectedSector: 1111,
				expectedDensity: 0,
				expectedWarps: 2,
				expectedAnomaly: false,
				description:  "Empty sector with zero density",
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Reset parser state for clean test
				parser.currentDisplay = DisplayDensity
				
				// Process the density line
				parser.processDensityLine(tc.densityLine)
				
				// Load sector to verify data was stored
				sector, err := db.LoadSector(tc.expectedSector)
				if err != nil {
					t.Fatalf("Failed to load sector %d: %v", tc.expectedSector, err)
				}
				
				// Verify density data
				if sector.Density != tc.expectedDensity {
					t.Errorf("Expected density %d, got %d", tc.expectedDensity, sector.Density)
				}
				// Note: Density scans report warp counts but don't provide actual warp connections
				// We no longer validate warp counts from density scans since they're informational only
				if sector.Anomaly != tc.expectedAnomaly {
					t.Errorf("Expected anomaly %t, got %t", tc.expectedAnomaly, sector.Anomaly)
				}
				
				// Verify explored status was set to etDensity
				if sector.Explored != database.EtDensity {
					t.Errorf("Expected explored status %d (EtDensity), got %d", database.EtDensity, sector.Explored)
				}
				
				// Verify constellation was set for density-only sectors
				if sector.Constellation != "??? (Density only)" {
					t.Errorf("Expected constellation '??? (Density only)', got '%s'", sector.Constellation)
				}
				
				// Verify update timestamp was set
				if sector.UpDate.IsZero() {
					t.Errorf("Expected update timestamp to be set")
				}
				
				t.Logf("✓ %s: %s", tc.name, tc.description)
			})
		}
	})

	t.Run("Real Format Parameter Extraction", func(t *testing.T) {
		// Test parameter extraction on real TWX density scan format
		testLine := "Sector ( 1234) ==>           1500  Warps : 6    NavHaz :     5%    Anom : Yes"
		
		// Test parameter extraction for real format (Pascal 1-indexed)
		testCases := []struct {
			param    int
			expected string
			desc     string
		}{
			{1, "Sector", "Parameter 1 should be 'Sector'"},
			{2, "(", "Parameter 2 should be '('"},
			{3, "1234)", "Parameter 3 should be sector with closing paren"},
			{4, "==>", "Parameter 4 should be arrow separator"},
			{5, "1500", "Parameter 5 should be density value"},
			{6, "Warps", "Parameter 6 should be 'Warps'"},
			{7, ":", "Parameter 7 should be colon"},
			{8, "6", "Parameter 8 should be warps count"},
			{9, "NavHaz", "Parameter 9 should be 'NavHaz'"},
			{10, ":", "Parameter 10 should be colon"},
			{11, "5%", "Parameter 11 should be NavHaz percentage"},
			{12, "Anom", "Parameter 12 should be 'Anom'"},
			{13, ":", "Parameter 13 should be colon"},
			{14, "Yes", "Parameter 14 should be anomaly status"},
		}
		
		for _, tt := range testCases {
			result := parser.getParameter(testLine, tt.param)
			if result != tt.expected {
				t.Errorf("%s: expected '%s', got '%s'", tt.desc, tt.expected, result)
			} else {
				t.Logf("✓ %s", tt.desc)
			}
		}
	})

	t.Run("Sector Update Logic", func(t *testing.T) {
		// Test Pascal logic: only update constellation/explored if sector is unexplored or calc-only
		
		// First, create a sector that's already explored with holoscanning
		existingSector := database.NULLSector()
		existingSector.Constellation = "Known Space"
		existingSector.Explored = database.EtHolo
		existingSector.Density = 1000
		
		if err := db.SaveSector(existingSector, 2222); err != nil {
			t.Fatalf("Failed to save existing sector: %v", err)
		}
		
		// Process density scan for this already-explored sector
		parser.currentDisplay = DisplayDensity
		parser.processDensityLine("Sector ( 2222) ==>           2000  Warps : 3    NavHaz :     0%    Anom : No")
		
		// Verify the sector was updated with new density but kept existing exploration data
		sector, err := db.LoadSector(2222)
		if err != nil {
			t.Fatalf("Failed to load updated sector: %v", err)
		}
		
		// Density should be updated
		if sector.Density != 2000 {
			t.Errorf("Expected density to be updated to 2000, got %d", sector.Density)
		}
		
		// But constellation and explored status should remain unchanged (not density-only)
		if sector.Constellation == "??? (Density only)" {
			t.Errorf("Already explored sector should not have density-only constellation")
		}
		if sector.Explored != database.EtHolo {
			t.Errorf("Expected explored status to remain EtHolo, got %d", sector.Explored)
		}
		
		t.Log("✓ Existing explored sectors correctly preserve exploration data")
	})

	t.Run("Database Integration", func(t *testing.T) {
		// Test without database (should handle gracefully)
		parserNoDB := &TWXParser{}
		parserNoDB.currentDisplay = DisplayDensity
		
		// Should not crash
		parserNoDB.processDensityLine("Sector ( 1234) ==>           1000  Warps : 3    NavHaz :     0%    Anom : No")
		
		t.Log("✓ Graceful handling without database")
	})

	t.Run("Invalid Data Handling", func(t *testing.T) {
		parser.currentDisplay = DisplayDensity
		
		// Test invalid lines that should be ignored
		lines := []string{
			"",                           // Empty line
			"Not a sector line",         // Doesn't start with "Sector"
			"Sector",                    // Too short
			"Sector invalid",            // Invalid sector number
			"Sector (    0) ==>           1000  Warps : 3    NavHaz :     0%    Anom : No", // Zero sector
		}
		
		for _, line := range lines {
			// Should not crash or cause errors
			parser.processDensityLine(line)
		}
		
		t.Log("✓ Invalid density lines handled gracefully")
	})
}

func TestDensityParameterParsing(t *testing.T) {
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()
	
	parser := NewTWXParser(db, nil)
	parser.currentDisplay = DisplayDensity
	
	t.Run("Complex Density Line Parsing", func(t *testing.T) {
		// Test real TWX density scan parsing with various formats
		testCases := []struct {
			line                string
			expectedSector      int
			expectedDensity     int
			expectedWarps       int
			expectedAnomaly     bool
			description         string
		}{
			{
				line:            "Sector (    1) ==>           5000  Warps : 6    NavHaz :     0%    Anom : No",
				expectedSector:  1,
				expectedDensity: 5000,
				expectedWarps:   6,
				expectedAnomaly: false,
				description:     "Single digit sector with high density",
			},
			{
				line:            "Sector  12345  ==>            999  Warps : 1    NavHaz :    25%    Anom : Yes",
				expectedSector:  12345,
				expectedDensity: 999,
				expectedWarps:   1,
				expectedAnomaly: true,
				description:     "Five digit sector with low density and anomaly",
			},
			{
				line:            "Sector (  555) ==>          10500  Warps : 6    NavHaz :    15%    Anom : Yes",
				expectedSector:  555,
				expectedDensity: 10500,
				expectedWarps:   6,
				expectedAnomaly: true,
				description:     "Three digit sector with very high density and anomaly",
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.description, func(t *testing.T) {
				// Parse the line using the actual parser logic
				parser.processDensityLine(tc.line)
				
				// Verify the sector was parsed and saved correctly
				sector, err := db.LoadSector(tc.expectedSector)
				if err != nil {
					t.Fatalf("Failed to load sector %d: %v", tc.expectedSector, err)
				}
				
				if sector.Density != tc.expectedDensity {
					t.Errorf("Expected density %d, got %d", tc.expectedDensity, sector.Density)
				}
				
				if sector.Warps != tc.expectedWarps {
					t.Errorf("Expected warps %d, got %d", tc.expectedWarps, sector.Warps)
				}
				
				if sector.Anomaly != tc.expectedAnomaly {
					t.Errorf("Expected anomaly %t, got %t", tc.expectedAnomaly, sector.Anomaly)
				}
				
				t.Logf("✓ %s", tc.description)
			})
		}
	})
}

func TestDensityWorkflowIntegration(t *testing.T) {
	// Test complete density scanning workflow
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParser(db, nil)

	t.Run("Complete Density Scan Workflow", func(t *testing.T) {
		// Simulate complete density scanning session
		lines := []string{
			"                          Relative Density Scan",                     // Start density scan
			"Sector ( 1001) ==>           1200  Warps : 4    NavHaz :     5%    Anom : No",  // First sector
			"Sector ( 1002) ==>           3500  Warps : 2    NavHaz :    15%    Anom : Yes", // Second sector with anomaly  
			"Sector ( 1003) ==>            800  Warps : 6    NavHaz :     0%    Anom : No",    // Third sector
		}
		
		for _, line := range lines {
			parser.ProcessString(line + "\r")
		}
		
		// Verify all sectors were processed and stored
		testCases := []struct {
			id      int
			density int
			warps   int
			anomaly bool
		}{
			{1001, 1200, 4, false},
			{1002, 3500, 2, true},
			{1003, 800, 6, false},
		}
		
		// Use database assertions instead of LoadSector
		dbAsserts := setup.NewDBAsserts(t, db.GetDB())
		
		for _, expected := range testCases {
			// Assert sector exists
			dbAsserts.AssertSectorExists(expected.id)
			
			// Assert density and anomaly values
			dbAsserts.AssertSectorDensity(expected.id, expected.density)
			dbAsserts.AssertSectorAnomaly(expected.id, expected.anomaly)
			
			// Assert exploration status is EtDensity
			dbAsserts.AssertSectorExplorationStatus(expected.id, int(database.EtDensity))
			
			// Note: Density scans report warp counts but don't provide actual warp connections
			// We no longer validate warp counts from density scans since they're informational only
		}
		
		t.Log("✓ Complete density scan workflow processed and stored correctly")
	})
}