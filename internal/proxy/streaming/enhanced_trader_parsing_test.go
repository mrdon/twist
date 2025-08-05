package streaming

import (
	"testing"
	"twist/internal/proxy/database"
)

func TestTWXParser_EnhancedTraderParsing(t *testing.T) {
	// Create parser
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)
	
	// Test Pascal-compliant trader parsing
	testCases := []struct {
		name            string
		traderLine      string
		continuationLine string
		expectedTrader  TraderInfo
		description     string
	}{
		{
			name:            "basic_trader_with_ship",
			traderLine:      "Traders : Captain Kirk, w/ 1,000 ftrs",
			continuationLine: "        in Enterprise (Constitution Class)",
			expectedTrader: TraderInfo{
				Name:      "Captain Kirk",
				ShipName:  "Enterprise",
				ShipType:  "Constitution Class",
				Fighters:  1000,
				Alignment: "",
			},
			description: "Basic trader with ship details on continuation line",
		},
		{
			name:            "trader_with_alignment",
			traderLine:      "Traders : Admiral Picard (Good), w/ 800 ftrs",
			continuationLine: "        in Enterprise-D (Galaxy Class)",
			expectedTrader: TraderInfo{
				Name:      "Admiral Picard",
				ShipName:  "Enterprise-D",
				ShipType:  "Galaxy Class",
				Fighters:  800,
				Alignment: "Good",
			},
			description: "Trader with alignment in main line",
		},
		{
			name:            "trader_ship_name_only",
			traderLine:      "Traders : Captain Sisko, w/ 1,200 ftrs",
			continuationLine: "        in Defiant",
			expectedTrader: TraderInfo{
				Name:      "Captain Sisko",
				ShipName:  "Defiant",
				ShipType:  "",
				Fighters:  1200,
				Alignment: "",
			},
			description: "Trader with ship name but no ship type",
		},
		{
			name:            "trader_no_fighters",
			traderLine:      "Traders : Merchant Bob",
			continuationLine: "        in FreighterShip (Cargo Hauler)",
			expectedTrader: TraderInfo{
				Name:      "Merchant Bob",
				ShipName:  "FreighterShip",
				ShipType:  "Cargo Hauler",
				Fighters:  0,
				Alignment: "",
			},
			description: "Trader without fighter count",
		},
		{
			name:            "trader_with_ship_alignment",
			traderLine:      "Traders : Commander Data, w/ 500 ftrs",
			continuationLine: "        in ShipName [Good]",
			expectedTrader: TraderInfo{
				Name:      "Commander Data",
				ShipName:  "ShipName",
				ShipType:  "",
				Fighters:  500,
				Alignment: "Good",
			},
			description: "Trader with alignment in ship continuation line",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset parser state
			parser.currentTraders = nil
			parser.currentTrader = TraderInfo{}
			parser.sectorPosition = SectorPosNormal
			
			// Process trader line
			parser.parseSectorTraders(tc.traderLine)
			
			// Process continuation line if provided
			if tc.continuationLine != "" {
				parser.handleSectorContinuation(tc.continuationLine)
			} else {
				// Finalize trader if no continuation line
				parser.finalizeCurrentTrader()
			}
			
			// Verify results
			if len(parser.currentTraders) != 1 {
				t.Errorf("Expected 1 trader, got %d", len(parser.currentTraders))
				if len(parser.currentTraders) > 0 {
					t.Errorf("Traders found: %+v", parser.currentTraders)
				}
				return
			}
			
			trader := parser.currentTraders[0]
			
			if trader.Name != tc.expectedTrader.Name {
				t.Errorf("Expected trader name '%s', got '%s'", tc.expectedTrader.Name, trader.Name)
			}
			
			if trader.ShipName != tc.expectedTrader.ShipName {
				t.Errorf("Expected ship name '%s', got '%s'", tc.expectedTrader.ShipName, trader.ShipName)
			}
			
			if trader.ShipType != tc.expectedTrader.ShipType {
				t.Errorf("Expected ship type '%s', got '%s'", tc.expectedTrader.ShipType, trader.ShipType)
			}
			
			if trader.Fighters != tc.expectedTrader.Fighters {
				t.Errorf("Expected %d fighters, got %d", tc.expectedTrader.Fighters, trader.Fighters)
			}
			
			if trader.Alignment != tc.expectedTrader.Alignment {
				t.Errorf("Expected alignment '%s', got '%s'", tc.expectedTrader.Alignment, trader.Alignment)
			}
		})
	}
}

func TestTWXParser_MultipleTradersParsing(t *testing.T) {
	// Test parsing multiple traders in continuation lines (Pascal behavior)
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)
	
	// Process multiple trader lines
	traderLines := []string{
		"Traders : Captain Kirk, w/ 1,000 ftrs",
		"        in Enterprise (Constitution Class)",
		"        Admiral Picard, w/ 800 ftrs",
		"        in Enterprise-D (Galaxy Class)",
		"        Commander Sisko, w/ 1,200 ftrs",
		"        in Defiant (Defiant Class)",
	}
	
	// Reset parser state
	parser.currentTraders = nil
	parser.currentTrader = TraderInfo{}
	parser.sectorPosition = SectorPosNormal
	
	// Process first trader
	parser.parseSectorTraders(traderLines[0])
	
	// Process continuation lines
	for i := 1; i < len(traderLines); i++ {
		parser.handleSectorContinuation(traderLines[i])
	}
	
	// Finalize any pending trader
	parser.finalizeCurrentTrader()
	
	// Verify we have 3 traders
	expectedTraderCount := 3
	if len(parser.currentTraders) != expectedTraderCount {
		t.Errorf("Expected %d traders, got %d", expectedTraderCount, len(parser.currentTraders))
		for i, trader := range parser.currentTraders {
			t.Errorf("Trader %d: %+v", i, trader)
		}
		return
	}
	
	// Verify first trader
	trader1 := parser.currentTraders[0]
	if trader1.Name != "Captain Kirk" || trader1.ShipName != "Enterprise" || trader1.ShipType != "Constitution Class" {
		t.Errorf("First trader not parsed correctly: %+v", trader1)
	}
	
	// Verify second trader
	trader2 := parser.currentTraders[1]
	if trader2.Name != "Admiral Picard" || trader2.ShipName != "Enterprise-D" || trader2.ShipType != "Galaxy Class" {
		t.Errorf("Second trader not parsed correctly: %+v", trader2)
	}
	
	// Verify third trader
	trader3 := parser.currentTraders[2]
	if trader3.Name != "Commander Sisko" || trader3.ShipName != "Defiant" || trader3.ShipType != "Defiant Class" {
		t.Errorf("Third trader not parsed correctly: %+v", trader3)
	}
}

func TestTWXParser_TraderParsingValidation(t *testing.T) {
	// Test validation and error handling
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)
	
	testCases := []struct {
		name             string
		traderLine       string
		expectedFighters int
		expectedValid    bool
		description      string
	}{
		{
			name:             "valid_fighter_count",
			traderLine:       "Traders : Captain Kirk, w/ 5,000 ftrs",
			expectedFighters: 5000,
			expectedValid:    true,
			description:      "Valid fighter count with commas",
		},
		{
			name:             "invalid_negative_fighters",
			traderLine:       "Traders : Captain Kirk, w/ -100 ftrs",
			expectedFighters: 0, // Should be reset to 0 for invalid values
			expectedValid:    false,
			description:      "Invalid negative fighter count should be reset",
		},
		{
			name:             "zero_fighters",
			traderLine:       "Traders : Captain Kirk, w/ 0 ftrs",
			expectedFighters: 0,
			expectedValid:    true,
			description:      "Zero fighters is valid",
		},
		{
			name:             "large_fighter_count",
			traderLine:       "Traders : Captain Kirk, w/ 999,999 ftrs",
			expectedFighters: 999999,
			expectedValid:    true,
			description:      "Large fighter count should be valid",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset parser state
			parser.currentTraders = nil
			parser.currentTrader = TraderInfo{}
			parser.sectorPosition = SectorPosNormal
			
			// Process trader line
			parser.parseSectorTraders(tc.traderLine)
			
			// Finalize trader
			parser.finalizeCurrentTrader()
			
			// Verify results
			if len(parser.currentTraders) != 1 {
				t.Errorf("Expected 1 trader, got %d", len(parser.currentTraders))
				return
			}
			
			trader := parser.currentTraders[0]
			
			if trader.Fighters != tc.expectedFighters {
				t.Errorf("Expected %d fighters, got %d", tc.expectedFighters, trader.Fighters)
			}
			
			// Verify trader has required fields
			if trader.Name == "" {
				t.Error("Trader name should not be empty")
			}
		})
	}
}

func TestTWXParser_TraderParsingEdgeCases(t *testing.T) {
	// Test edge cases and malformed input
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)
	
	testCases := []struct {
		name            string
		traderLine      string
		continuationLine string
		description     string
		shouldPanic     bool
	}{
		{
			name:        "empty_trader_line",
			traderLine:  "Traders : ",
			description: "Empty trader line should not crash",
			shouldPanic: false,
		},
		{
			name:        "no_fighter_section",
			traderLine:  "Traders : Merchant Bob",
			description: "Trader without fighters section",
			shouldPanic: false,
		},
		{
			name:            "malformed_continuation",
			traderLine:      "Traders : Captain Kirk, w/ 100 ftrs",
			continuationLine: "        in",
			description:     "Malformed continuation line should not crash",
			shouldPanic:     false,
		},
		{
			name:            "ship_continuation_without_trader",
			traderLine:      "",
			continuationLine: "        in Enterprise (Constitution)",
			description:     "Ship continuation without pending trader",
			shouldPanic:     false,
		},
		{
			name:        "unicode_trader_name",
			traderLine:  "Traders : Капитан Кирк, w/ 500 ftrs",
			description: "Unicode characters in trader name",
			shouldPanic: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset parser state
			parser.currentTraders = nil
			parser.currentTrader = TraderInfo{}
			parser.sectorPosition = SectorPosNormal
			
			// Test should not panic
			defer func() {
				if r := recover(); r != nil {
					if !tc.shouldPanic {
						t.Errorf("Test should not panic, but panicked with: %v", r)
					}
				}
			}()
			
			// Process trader line if provided
			if tc.traderLine != "" {
				parser.parseSectorTraders(tc.traderLine)
			}
			
			// Process continuation line if provided
			if tc.continuationLine != "" {
				parser.handleSectorContinuation(tc.continuationLine)
			}
			
			// Finalize any pending trader
			parser.finalizeCurrentTrader()
			
			// Should have at least attempted to parse something
			// The exact behavior for malformed input may vary
		})
	}
}

func TestTWXParser_TraderDataConsistency(t *testing.T) {
	// Test data consistency validation
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)
	
	// Test trader with ship name but no ship type
	parser.currentTraders = nil
	parser.currentTrader = TraderInfo{}
	parser.sectorPosition = SectorPosNormal
	
	parser.parseSectorTraders("Traders : Captain Kirk, w/ 100 ftrs")
	parser.handleSectorContinuation("        in Enterprise")
	
	if len(parser.currentTraders) != 1 {
		t.Errorf("Expected 1 trader, got %d", len(parser.currentTraders))
		return
	}
	
	trader := parser.currentTraders[0]
	if trader.Name != "Captain Kirk" {
		t.Errorf("Expected trader name 'Captain Kirk', got '%s'", trader.Name)
	}
	
	if trader.ShipName != "Enterprise" {
		t.Errorf("Expected ship name 'Enterprise', got '%s'", trader.ShipName)
	}
	
	if trader.ShipType != "" {
		t.Errorf("Expected empty ship type, got '%s'", trader.ShipType)
	}
	
	if trader.Fighters != 100 {
		t.Errorf("Expected 100 fighters, got %d", trader.Fighters)
	}
}

func TestTWXParser_TraderStateManagement(t *testing.T) {
	// Test proper state management for multi-line trader data
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)
	
	// Reset parser state
	parser.currentTraders = nil
	parser.currentTrader = TraderInfo{}
	parser.sectorPosition = SectorPosNormal
	
	// Start trader parsing
	parser.parseSectorTraders("Traders : Captain Kirk, w/ 100 ftrs")
	
	// Verify currentTrader state
	if parser.currentTrader.Name != "Captain Kirk" {
		t.Errorf("Expected pending trader name 'Captain Kirk', got '%s'", parser.currentTrader.Name)
	}
	
	if parser.currentTrader.Fighters != 100 {
		t.Errorf("Expected pending trader fighters 100, got %d", parser.currentTrader.Fighters)
	}
	
	// Should have no completed traders yet
	if len(parser.currentTraders) != 0 {
		t.Errorf("Expected 0 completed traders, got %d", len(parser.currentTraders))
	}
	
	// Add ship details
	parser.handleSectorContinuation("        in Enterprise (Constitution)")
	
	// Should now have completed trader and cleared pending state
	if len(parser.currentTraders) != 1 {
		t.Errorf("Expected 1 completed trader, got %d", len(parser.currentTraders))
		return
	}
	
	if parser.currentTrader.Name != "" {
		t.Errorf("Expected cleared pending trader, but currentTrader still has name: %s", parser.currentTrader.Name)
	}
	
	trader := parser.currentTraders[0]
	if trader.Name != "Captain Kirk" || trader.ShipName != "Enterprise" || trader.ShipType != "Constitution" {
		t.Errorf("Completed trader not correct: %+v", trader)
	}
}