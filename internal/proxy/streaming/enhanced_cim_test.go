package streaming

import (
	"testing"
	"twist/internal/proxy/database"
)

func TestEnhancedCIMProcessing(t *testing.T) {
	// Create test database and parser
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParserWithAPI(db, nil)

	t.Run("CIM Prompt Detection", func(t *testing.T) {
		// Test CIM prompt detection (Pascal ": " handling)
		parser.ProcessString(": \r")
		
		if parser.currentDisplay != DisplayCIM {
			t.Errorf("Expected DisplayCIM after prompt, got %d", parser.currentDisplay)
		}
		
		t.Log("✓ CIM prompt correctly sets DisplayCIM state")
	})

	t.Run("Port CIM Line Processing", func(t *testing.T) {
		// Set up CIM state
		parser.currentDisplay = DisplayCIM
		
		// Test port CIM with various buy/sell patterns
		testCases := []struct {
			name           string
			cimLine        string
			expectedSector int
			expectedBuyOre bool
			expectedBuyOrg bool
			expectedBuyEquip bool
			expectedClass  int
			description    string
		}{
			{
				name:           "Selling all commodities",
				cimLine:        "1234 5000 60% 3000 80% 2000 90%",
				expectedSector: 1234,
				expectedBuyOre: false,
				expectedBuyOrg: false,
				expectedBuyEquip: false,
				expectedClass:  7, // SSS pattern
				description:    "Port selling all three commodities",
			},
			{
				name:           "Buying ore and equipment",
				cimLine:        "2345 -5000 60% 3000 80% -2000 90%",
				expectedSector: 2345,
				expectedBuyOre: true,
				expectedBuyOrg: false,
				expectedBuyEquip: true,
				expectedClass:  2, // BSB pattern
				description:    "Port buying ore and equipment, selling organics",
			},
			{
				name:           "Buying ore and organics",
				cimLine:        "3456 -5000 60% -3000 80% 2000 90%",
				expectedSector: 3456,
				expectedBuyOre: true,
				expectedBuyOrg: true,
				expectedBuyEquip: false,
				expectedClass:  1, // BBS pattern
				description:    "Port buying ore and organics, selling equipment",
			},
			{
				name:           "Buying all commodities",
				cimLine:        "4567 -5000 60% -3000 80% -2000 90%",
				expectedSector: 4567,
				expectedBuyOre: true,
				expectedBuyOrg: true,
				expectedBuyEquip: true,
				expectedClass:  8, // BBB pattern
				description:    "Port buying all three commodities",
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Reset parser state
				parser.currentDisplay = DisplayCIM
				
				// Process the CIM line
				parser.processCIMLine(tc.cimLine)
				
				// Verify display state changed to PortCIM
				if parser.currentDisplay != DisplayPortCIM {
					t.Errorf("Expected DisplayPortCIM, got %d", parser.currentDisplay)
				}
				
				// Load sector to verify data was stored
				sector, err := db.LoadSector(tc.expectedSector)
				if err != nil {
					t.Fatalf("Failed to load sector %d: %v", tc.expectedSector, err)
				}
				
				// Verify port class was determined correctly
				if sector.SPort.ClassIndex != tc.expectedClass {
					t.Errorf("Expected port class %d, got %d", tc.expectedClass, sector.SPort.ClassIndex)
				}
				
				// Verify buy/sell status
				if sector.SPort.BuyProduct[0] != tc.expectedBuyOre {
					t.Errorf("Expected buy ore %t, got %t", tc.expectedBuyOre, sector.SPort.BuyProduct[0])
				}
				if sector.SPort.BuyProduct[1] != tc.expectedBuyOrg {
					t.Errorf("Expected buy org %t, got %t", tc.expectedBuyOrg, sector.SPort.BuyProduct[1])
				}
				if sector.SPort.BuyProduct[2] != tc.expectedBuyEquip {
					t.Errorf("Expected buy equip %t, got %t", tc.expectedBuyEquip, sector.SPort.BuyProduct[2])
				}
				
				t.Logf("✓ %s: %s", tc.name, tc.description)
			})
		}
	})

	t.Run("Warp CIM Line Processing", func(t *testing.T) {
		// Test warp CIM processing
		parser.currentDisplay = DisplayCIM
		
		warpCIMLine := "5678 1234 2345 3456 4567 5678 6789"
		parser.processCIMLine(warpCIMLine)
		
		// Verify display state changed to WarpCIM
		if parser.currentDisplay != DisplayWarpCIM {
			t.Errorf("Expected DisplayWarpCIM, got %d", parser.currentDisplay)
		}
		
		// Load sector to verify warp data was stored
		sector, err := db.LoadSector(5678)
		if err != nil {
			t.Fatalf("Failed to load sector 5678: %v", err)
		}
		
		// Verify warp data
		expectedWarps := []int{1234, 2345, 3456, 4567, 5678, 6789}
		for i, expectedWarp := range expectedWarps {
			if sector.Warp[i] != expectedWarp {
				t.Errorf("Warp %d: expected %d, got %d", i, expectedWarp, sector.Warp[i])
			}
		}
		
		t.Log("✓ Warp CIM line processed and stored correctly")
	})

	t.Run("CIM Error Handling", func(t *testing.T) {
		parser.currentDisplay = DisplayCIM
		
		// Test invalid CIM lines
		invalidLines := []string{
			"", // Empty line
			"12", // Too short
			"1234", // Port CIM without enough parameters
			"1234 5000", // Port CIM incomplete
			"invalid 5000 60% 3000 80% 2000 90%", // Invalid sector number
			"1234 5000 150% 3000 80% 2000 90%", // Invalid percentage
		}
		
		for _, invalidLine := range invalidLines {
			parser.currentDisplay = DisplayCIM // Reset state
			parser.processCIMLine(invalidLine)
			
			// Should reset display to None on error
			if parser.currentDisplay != DisplayNone {
				t.Errorf("Expected DisplayNone after invalid line '%s', got %d", invalidLine, parser.currentDisplay)
			}
		}
		
		t.Log("✓ CIM error handling works correctly")
	})
}

func TestCIMValueExtraction(t *testing.T) {
	parser := NewTestTWXParser()
	
	tests := []struct {
		line     string
		paramNum int
		expected int
		description string
	}{
		{
			line:     "1234 5000 60 3000 80 2000 90",
			paramNum: 1,
			expected: 1234,
			description: "Extract sector number (parameter 1)",
		},
		{
			line:     "1234 5000 60 3000 80 2000 90",
			paramNum: 2,
			expected: 5000,
			description: "Extract ore amount (parameter 2)",
		},
		{
			line:     "1234 5000 60 3000 80 2000 90",
			paramNum: 3,
			expected: 60,
			description: "Extract ore percentage (parameter 3)",
		},
		{
			line:     "1234 0 60 3000 80 2000 90",
			paramNum: 2,
			expected: 0,
			description: "Handle zero value",
		},
		{
			line:     "1234 5000 60 3000 80 2000 90",
			paramNum: 10,
			expected: -1,
			description: "Invalid parameter number returns -1",
		},
		{
			line:     "",
			paramNum: 1,
			expected: -1,
			description: "Empty line returns -1",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := parser.getCIMValue(tt.line, tt.paramNum)
			if result != tt.expected {
				t.Errorf("getCIMValue('%s', %d): expected %d, got %d", 
					tt.line, tt.paramNum, tt.expected, result)
			}
			t.Logf("✓ %s", tt.description)
		})
	}
}

func TestCIMBuyStatusDetection(t *testing.T) {
	parser := NewTestTWXParser()
	
	tests := []struct {
		line     string
		paramNum int
		expected bool
		description string
	}{
		{
			line:     "1234 -5000 60% 3000 80% 2000 90%",
			paramNum: 2,
			expected: true,
			description: "Detect buy status for ore (parameter 2 with dash)",
		},
		{
			line:     "1234 5000 60% -3000 80% 2000 90%",
			paramNum: 4,
			expected: true,
			description: "Detect buy status for organics (parameter 4 with dash)",
		},
		{
			line:     "1234 5000 60% 3000 80% -2000 90%",
			paramNum: 6,
			expected: true,
			description: "Detect buy status for equipment (parameter 6 with dash)",
		},
		{
			line:     "1234 5000 60% 3000 80% 2000 90%",
			paramNum: 2,
			expected: false,
			description: "No buy status for ore (no dash)",
		},
		{
			line:     "1234 -5000 60% -3000 80% -2000 90%",
			paramNum: 2,
			expected: true,
			description: "Multiple buy indicators - ore",
		},
		{
			line:     "1234 -5000 60% -3000 80% -2000 90%",
			paramNum: 4,
			expected: true,
			description: "Multiple buy indicators - organics",
		},
		{
			line:     "1234 -5000 60% -3000 80% -2000 90%",
			paramNum: 6,
			expected: true,
			description: "Multiple buy indicators - equipment",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := parser.determineCIMBuyStatus(tt.line, tt.paramNum)
			if result != tt.expected {
				t.Errorf("determineCIMBuyStatus('%s', %d): expected %t, got %t", 
					tt.line, tt.paramNum, tt.expected, result)
			}
			t.Logf("✓ %s", tt.description)
		})
	}
}

func TestCIMIntegrationWithRealData(t *testing.T) {
	// Test with database integration
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParserWithAPI(db, nil)
	
	t.Run("Complete CIM workflow", func(t *testing.T) {
		// Simulate complete CIM download workflow
		testSequence := []string{
			": ",                                          // CIM prompt
			"1234 5000 60% 3000 80% 2000 90%",           // Port CIM data
			"5678 1111 2222 3333 4444 5555 6666",        // Warp CIM data
			"9999 -1000 50% -2000 70% 3000 90%",         // Another port CIM with buying
		}
		
		for _, line := range testSequence {
			parser.ProcessString(line + "\r")
		}
		
		// Verify port CIM data was stored
		sector1, err := db.LoadSector(1234)
		if err != nil {
			t.Fatalf("Failed to load sector 1234: %v", err)
		}
		
		// Check port data
		if sector1.SPort.ProductAmount[0] != 5000 { // Ore
			t.Errorf("Expected ore amount 5000, got %d", sector1.SPort.ProductAmount[0])
		}
		if sector1.SPort.ProductPercent[1] != 80 { // Organics %
			t.Errorf("Expected organics percent 80, got %d", sector1.SPort.ProductPercent[1])
		}
		if sector1.SPort.BuyProduct[0] != false { // Not buying ore
			t.Errorf("Expected not buying ore, got %t", sector1.SPort.BuyProduct[0])
		}
		
		// Verify warp CIM data was stored
		sector2, err := db.LoadSector(5678)
		if err != nil {
			t.Fatalf("Failed to load sector 5678: %v", err)
		}
		
		expectedWarps := []int{1111, 2222, 3333, 4444, 5555, 6666}
		for i, expected := range expectedWarps {
			if sector2.Warp[i] != expected {
				t.Errorf("Warp %d: expected %d, got %d", i, expected, sector2.Warp[i])
			}
		}
		
		// Verify buying port CIM data
		sector3, err := db.LoadSector(9999)
		if err != nil {
			t.Fatalf("Failed to load sector 9999: %v", err)
		}
		
		if !sector3.SPort.BuyProduct[0] { // Should be buying ore
			t.Errorf("Expected buying ore, got %t", sector3.SPort.BuyProduct[0])
		}
		if !sector3.SPort.BuyProduct[1] { // Should be buying organics
			t.Errorf("Expected buying organics, got %t", sector3.SPort.BuyProduct[1])
		}
		if sector3.SPort.BuyProduct[2] { // Should not be buying equipment
			t.Errorf("Expected not buying equipment, got %t", sector3.SPort.BuyProduct[2])
		}
		
		// Check port class determination (BBS = 1)
		if sector3.SPort.ClassIndex != 1 {
			t.Errorf("Expected port class 1 (BBS), got %d", sector3.SPort.ClassIndex)
		}
		
		t.Log("✓ Complete CIM workflow processed and stored correctly")
	})
}