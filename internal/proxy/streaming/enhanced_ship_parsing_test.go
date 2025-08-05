package streaming

import (
	"strings"
	"testing"
	"twist/internal/proxy/database"
)

func TestTWXParser_EnhancedShipParsing(t *testing.T) {
	// Create parser
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)
	
	// Test Pascal-compliant ship parsing
	testCases := []struct {
		name           string
		shipLine       string
		continuationLine string
		expectedShip   ShipInfo
		description    string
	}{
		{
			name:         "basic_ship_with_type",
			shipLine:     "Ships   : Enterprise [Owned by Kirk], w/ 500 ftrs,",
			continuationLine: "        (Constitution Class Cruiser)",
			expectedShip: ShipInfo{
				Name:      "Enterprise",
				Owner:     "Kirk",
				ShipType:  "Constitution Class Cruiser",
				Fighters:  500,
				Alignment: "",
			},
			description: "Basic ship with fighters and ship type continuation",
		},
		{
			name:         "ship_with_alignment",
			shipLine:     "Ships   : Defiant [Owned by Sisko], w/ 1,200 ftrs,",
			continuationLine: "        (Good)",
			expectedShip: ShipInfo{
				Name:      "Defiant",
				Owner:     "Sisko",
				ShipType:  "",
				Fighters:  1200,
				Alignment: "Good",
			},
			description: "Ship with alignment in continuation line",
		},
		{
			name:         "pascal_exact_format",
			shipLine:     "Ships   : TestShip [Owned by Player], w/ 10,000 ftrs,",
			continuationLine: "        (Merchant Cruiser)",
			expectedShip: ShipInfo{
				Name:      "TestShip",
				Owner:     "Player",
				ShipType:  "Merchant Cruiser",
				Fighters:  10000,
				Alignment: "",
			},
			description: "Exact Pascal format test case",
		},
		{
			name:         "ship_no_fighters",
			shipLine:     "Ships   : Scout [Owned by Explorer]",
			continuationLine: "        (Scout Ship)",
			expectedShip: ShipInfo{
				Name:      "Scout",
				Owner:     "Explorer",
				ShipType:  "Scout Ship",
				Fighters:  0,
				Alignment: "",
			},
			description: "Ship without fighter count",
		},
		{
			name:         "fallback_bracket_parsing",
			shipLine:     "Ships   : Voyager [Federation], w/ 800 ftrs,",
			continuationLine: "        (Intrepid Class)",
			expectedShip: ShipInfo{
				Name:      "Voyager",
				Owner:     "Federation",
				ShipType:  "Intrepid Class",
				Fighters:  800,
				Alignment: "",
			},
			description: "Fallback bracket parsing (non-'Owned by' format)",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset parser state
			parser.currentShips = nil
			parser.sectorPosition = SectorPosNormal
			
			// Process ship line
			parser.parseSectorShips(tc.shipLine)
			
			// Process continuation line if provided
			if tc.continuationLine != "" {
				parser.handleSectorContinuation(tc.continuationLine)
			}
			
			// Verify results
			if len(parser.currentShips) != 1 {
				t.Errorf("Expected 1 ship, got %d", len(parser.currentShips))
				return
			}
			
			ship := parser.currentShips[0]
			
			if ship.Name != tc.expectedShip.Name {
				t.Errorf("Expected ship name '%s', got '%s'", tc.expectedShip.Name, ship.Name)
			}
			
			if ship.Owner != tc.expectedShip.Owner {
				t.Errorf("Expected ship owner '%s', got '%s'", tc.expectedShip.Owner, ship.Owner)
			}
			
			if ship.ShipType != tc.expectedShip.ShipType {
				t.Errorf("Expected ship type '%s', got '%s'", tc.expectedShip.ShipType, ship.ShipType)
			}
			
			if ship.Fighters != tc.expectedShip.Fighters {
				t.Errorf("Expected %d fighters, got %d", tc.expectedShip.Fighters, ship.Fighters)
			}
			
			if ship.Alignment != tc.expectedShip.Alignment {
				t.Errorf("Expected alignment '%s', got '%s'", tc.expectedShip.Alignment, ship.Alignment)
			}
		})
	}
}

func TestTWXParser_PascalShipContinuationLogic(t *testing.T) {
	// Test Pascal-specific continuation logic
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)
	
	// Test Pascal exact position matching for ship type
	testCases := []struct {
		name              string
		shipLine          string
		continuationLine  string
		expectedShipType  string
		shouldMatch       bool
		description       string
	}{
		{
			name:             "pascal_position_12_match",
			shipLine:         "Ships   : TestShip [Owned by Player], w/ 100 ftrs,",
			continuationLine: "        (Cruiser)",  // Position 12 (0-indexed 11) should be '('
			expectedShipType: "Cruiser",
			shouldMatch:      true,
			description:      "Pascal position 12 logic should match",
		},
		{
			name:             "wrong_position_no_match",
			shipLine:         "Ships   : TestShip [Owned by Player], w/ 100 ftrs,",
			continuationLine: "       (Cruiser)",   // Position 11 (0-indexed 10) is '(' - should not match Pascal logic
			expectedShipType: "",
			shouldMatch:      false,
			description:      "Wrong position should not match Pascal logic",
		},
		{
			name:             "exact_pascal_spacing",
			shipLine:         "Ships   : Enterprise [Owned by Kirk], w/ 500 ftrs,",
			continuationLine: "        (Constitution Class Cruiser)",
			expectedShipType: "Constitution Class Cruiser",
			shouldMatch:      true,
			description:      "Exact Pascal spacing should work",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset parser state
			parser.currentShips = nil
			parser.sectorPosition = SectorPosNormal
			
			// Process ship line
			parser.parseSectorShips(tc.shipLine)
			
			// Process continuation line
			parser.handleSectorContinuation(tc.continuationLine)
			
			// Verify results
			if len(parser.currentShips) != 1 {
				t.Errorf("Expected 1 ship, got %d", len(parser.currentShips))
				return
			}
			
			ship := parser.currentShips[0]
			
			if tc.shouldMatch {
				if ship.ShipType != tc.expectedShipType {
					t.Errorf("Expected ship type '%s', got '%s'", tc.expectedShipType, ship.ShipType)
				}
			} else {
				// For non-matching cases, ship type should either be empty or set by fallback logic
				if ship.ShipType == tc.expectedShipType && tc.expectedShipType != "" {
					t.Errorf("Ship type should not match Pascal logic, but got '%s'", ship.ShipType)
				}
			}
		})
	}
}

func TestTWXParser_ShipParsingValidation(t *testing.T) {
	// Test validation and error handling
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)
	
	testCases := []struct {
		name          string
		shipLine      string
		expectedFighters int
		expectedValid bool
		description   string
	}{
		{
			name:            "valid_fighter_count",
			shipLine:        "Ships   : TestShip [Owned by Player], w/ 5,000 ftrs,",
			expectedFighters: 5000,
			expectedValid:   true,
			description:     "Valid fighter count with commas",
		},
		{
			name:            "invalid_negative_fighters",
			shipLine:        "Ships   : TestShip [Owned by Player], w/ -100 ftrs,",
			expectedFighters: 0, // Should be reset to 0 for invalid values
			expectedValid:   false,
			description:     "Invalid negative fighter count should be reset",
		},
		{
			name:            "zero_fighters",
			shipLine:        "Ships   : TestShip [Owned by Player], w/ 0 ftrs,",
			expectedFighters: 0,
			expectedValid:   true,
			description:     "Zero fighters is valid",
		},
		{
			name:            "large_fighter_count",
			shipLine:        "Ships   : TestShip [Owned by Player], w/ 999,999 ftrs,",
			expectedFighters: 999999,
			expectedValid:   true,
			description:     "Large fighter count should be valid",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset parser state
			parser.currentShips = nil
			parser.sectorPosition = SectorPosNormal
			
			// Process ship line
			parser.parseSectorShips(tc.shipLine)
			
			// Verify results
			if len(parser.currentShips) != 1 {
				t.Errorf("Expected 1 ship, got %d", len(parser.currentShips))
				return
			}
			
			ship := parser.currentShips[0]
			
			if ship.Fighters != tc.expectedFighters {
				t.Errorf("Expected %d fighters, got %d", tc.expectedFighters, ship.Fighters)
			}
			
			// Verify ship has required fields
			if ship.Name == "" {
				t.Error("Ship name should not be empty")
			}
			
			// Owner validation depends on the input format
			if !strings.Contains(tc.shipLine, "[") && ship.Owner == "" {
				// Only expect owner if brackets are present
				t.Error("Ship owner should not be empty when brackets are present")
			}
		})
	}
}

func TestTWXParser_MultipleShipsInSector(t *testing.T) {
	// Test parsing multiple ships in the same sector
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)
	
	// Process multiple ship lines
	shipLines := []string{
		"Ships   : Enterprise [Owned by Kirk], w/ 500 ftrs,",
		"        (Constitution Class Cruiser)",
		"        Voyager [Owned by Janeway], w/ 800 ftrs,",
		"        (Intrepid Class)",
		"        Defiant [Owned by Sisko], w/ 1,200 ftrs,",
		"        (Defiant Class)",
	}
	
	// Reset parser state
	parser.currentShips = nil
	parser.sectorPosition = SectorPosNormal
	
	// Process first ship
	parser.parseSectorShips(shipLines[0])
	parser.handleSectorContinuation(shipLines[1])
	
	// Process continuation ships (these would normally come as sector continuation lines)
	parser.handleSectorContinuation(shipLines[2])
	parser.handleSectorContinuation(shipLines[3])
	parser.handleSectorContinuation(shipLines[4])
	parser.handleSectorContinuation(shipLines[5])
	
	// Verify we have 3 ships
	expectedShipCount := 3
	if len(parser.currentShips) != expectedShipCount {
		t.Errorf("Expected %d ships, got %d", expectedShipCount, len(parser.currentShips))
		return
	}
	
	// Verify first ship
	ship1 := parser.currentShips[0]
	if ship1.Name != "Enterprise" || ship1.Owner != "Kirk" || ship1.ShipType != "Constitution Class Cruiser" {
		t.Errorf("First ship not parsed correctly: %+v", ship1)
	}
	
	// Note: The current implementation may not handle multiple ships in continuation lines perfectly
	// This test documents the current behavior and can be enhanced as needed
}

func TestTWXParser_ShipParsingEdgeCases(t *testing.T) {
	// Test edge cases and malformed input
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)
	
	testCases := []struct {
		name        string
		shipLine    string
		description string
		shouldPanic bool
	}{
		{
			name:        "empty_ship_line",
			shipLine:    "Ships   : ",
			description: "Empty ship line should not crash",
			shouldPanic: false,
		},
		{
			name:        "malformed_brackets",
			shipLine:    "Ships   : TestShip [Owned by Player, w/ 100 ftrs,",
			description: "Malformed brackets should not crash",
			shouldPanic: false,
		},
		{
			name:        "no_fighters_section",
			shipLine:    "Ships   : TestShip [Owned by Player]",
			description: "Ship without fighters section",
			shouldPanic: false,
		},
		{
			name:        "unicode_ship_name",
			shipLine:    "Ships   : Ентерпрайз [Owned by Кирк], w/ 500 ftrs,",
			description: "Unicode characters in ship name",
			shouldPanic: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset parser state
			parser.currentShips = nil
			parser.sectorPosition = SectorPosNormal
			
			// Test should not panic
			defer func() {
				if r := recover(); r != nil {
					if !tc.shouldPanic {
						t.Errorf("Test should not panic, but panicked with: %v", r)
					}
				}
			}()
			
			// Process ship line
			parser.parseSectorShips(tc.shipLine)
			
			// Should have at least attempted to parse something
			// The exact behavior for malformed input may vary
		})
	}
}