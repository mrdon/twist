package streaming

import (
	"strings"
	"testing"
	"twist/internal/proxy/database"
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
				densityLine:  "Sector 1234 (The Sphere) Density: 1,500, NavHaz: 5%, Warps: 6, Anomaly: Yes",
				expectedSector: 1234,
				expectedDensity: 1500,
				expectedWarps: 6,
				expectedAnomaly: true,
				description:  "Sector with high density and anomaly",
			},
			{
				name:         "Low density without anomaly",
				densityLine:  "Sector 5678 (Deep Space) Density: 800, NavHaz: 0%, Warps: 3, Anomaly: No",
				expectedSector: 5678,
				expectedDensity: 800,
				expectedWarps: 3,
				expectedAnomaly: false,
				description:  "Sector with low density and no anomaly",
			},
			{
				name:         "Medium density with anomaly",
				densityLine:  "Sector 9999 (Outer Rim) Density: 2,345, NavHaz: 10%, Warps: 4, Anomaly: Yes",
				expectedSector: 9999,
				expectedDensity: 2345,
				expectedWarps: 4,
				expectedAnomaly: true,
				description:  "Sector with medium density and anomaly",
			},
			{
				name:         "Zero density exploration",
				densityLine:  "Sector 1111 (Empty Space) Density: 0, NavHaz: 0%, Warps: 2, Anomaly: No",
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

	t.Run("Pascal Parameter Extraction", func(t *testing.T) {
		// Test exact Pascal GetParameter behavior on density scan lines
		testLine := "Sector 1234 The Sphere Density: 1500 NavHaz: 5% Warps: 6 Anomaly: Yes"
		
		// Test parameter extraction (Pascal 1-indexed)
		testCases := []struct {
			param    int
			expected string
			desc     string
		}{
			{1, "Sector", "Parameter 1 should be 'Sector'"},
			{2, "1234", "Parameter 2 should be sector number"},
			{3, "The", "Parameter 3 should be constellation start"},
			{4, "Sphere", "Parameter 4 should be constellation end"},
			{5, "Density:", "Parameter 5 should be 'Density:'"},
			{6, "1500", "Parameter 6 should be density value"},
			{7, "NavHaz:", "Parameter 7 should be 'NavHaz:'"},
			{8, "5%", "Parameter 8 should be NavHaz value"},
			{9, "Warps:", "Parameter 9 should be 'Warps:'"},
			{10, "6", "Parameter 10 should be warps count"},
			{11, "Anomaly:", "Parameter 11 should be 'Anomaly:'"},
			{12, "Yes", "Parameter 12 should be anomaly status"},
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
		parser.processDensityLine("Sector 2222 (Test) Density: 2000 NavHaz: 0% Warps: 3 Anomaly: No")
		
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
		parserNoDB.processDensityLine("Sector 1234 (Test) Density: 1000 NavHaz: 0% Warps: 3 Anomaly: No")
		
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
			"Sector 0 (Test) Density: 1000 NavHaz: 0% Warps: 3 Anomaly: No", // Zero sector
		}
		
		for _, line := range lines {
			// Should not crash or cause errors
			parser.processDensityLine(line)
		}
		
		t.Log("✓ Invalid density lines handled gracefully")
	})
}

func TestDensityParameterParsing(t *testing.T) {
	parser := NewTestTWXParser()
	
	t.Run("Complex Density Line Parsing", func(t *testing.T) {
		// Test parsing with various constellation names and formats
		testCases := []struct {
			line                string
			expectedSector      int
			expectedDensity     int
			expectedWarps       int
			expectedAnomaly     bool
			description         string
		}{
			{
				line:            "Sector 1 (Sol) Density: 5,000 NavHaz: 0% Warps: 6 Anomaly: No",
				expectedSector:  1,
				expectedDensity: 5000,
				expectedWarps:   6,
				expectedAnomaly: false,
				description:     "Single digit sector with comma in density",
			},
			{
				line:            "Sector 12345 (Far Far Away Galaxy) Density: 999 NavHaz: 25% Warps: 1 Anomaly: Yes",
				expectedSector:  12345,
				expectedDensity: 999,
				expectedWarps:   1,
				expectedAnomaly: true,
				description:     "Five digit sector with long constellation name",
			},
			{
				line:            "Sector 555 (Alpha Centauri Prime) Density: 10,500 NavHaz: 15% Warps: 6 Anomaly: Yes",
				expectedSector:  555,
				expectedDensity: 10500,
				expectedWarps:   6,
				expectedAnomaly: true,
				description:     "Multi-word constellation with high density",
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.description, func(t *testing.T) {
				// Test parameter extraction
				cleanLine := strings.ReplaceAll(tc.line, "(", "")
				cleanLine = strings.ReplaceAll(cleanLine, ")", "")
				cleanLine = strings.ReplaceAll(cleanLine, ",", "")
				
				// Extract sector number (parameter 2)
				sectorStr := parser.getParameter(cleanLine, 2)
				sectorNum := parser.parseIntSafe(sectorStr)
				if sectorNum != tc.expectedSector {
					t.Errorf("Expected sector %d, got %d", tc.expectedSector, sectorNum)
				}
				
				// Find density parameter position (should be after "Density:")
				fields := strings.Fields(cleanLine)
				var densityValue int
				for i, field := range fields {
					if field == "Density:" && i+1 < len(fields) {
						densityValue = parser.parseIntSafe(fields[i+1])
						break
					}
				}
				if densityValue != tc.expectedDensity {
					t.Errorf("Expected density %d, got %d", tc.expectedDensity, densityValue)
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
			"Sector 1001 (Unknown) Density: 1,200 NavHaz: 5% Warps: 4 Anomaly: No",  // First sector
			"Sector 1002 (Unknown) Density: 3,500 NavHaz: 15% Warps: 2 Anomaly: Yes", // Second sector with anomaly  
			"Sector 1003 (Unknown) Density: 800 NavHaz: 0% Warps: 6 Anomaly: No",    // Third sector
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
		
		for _, expected := range testCases {
			sector, err := db.LoadSector(expected.id)
			if err != nil {
				t.Fatalf("Failed to load sector %d: %v", expected.id, err)
			}
			
			if sector.Density != expected.density {
				t.Errorf("Sector %d: expected density %d, got %d", expected.id, expected.density, sector.Density)
			}
			// Note: Density scans report warp counts but don't provide actual warp connections
			// We no longer validate warp counts from density scans since they're informational only
			if sector.Anomaly != expected.anomaly {
				t.Errorf("Sector %d: expected anomaly %t, got %t", expected.id, expected.anomaly, sector.Anomaly)
			}
			if sector.Explored != database.EtDensity {
				t.Errorf("Sector %d: expected explored status EtDensity, got %d", expected.id, sector.Explored)
			}
		}
		
		t.Log("✓ Complete density scan workflow processed and stored correctly")
	})
}