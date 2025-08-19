package streaming

import (
	"testing"
	"twist/internal/proxy/database"
)

func TestTraderShipDetailsParsing(t *testing.T) {
	// Create a test database
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	parser := NewTWXParser(db, nil)

	testCases := []struct {
		name           string
		sectorData     string
		expectedTraders []TraderInfo
	}{
		{
			name: "Single trader with ship details",
			sectorData: `Sector  : 1234 in Test Space
Warps to Sector(s) :  100 - 200 - 300
Traders : Captain Kirk, w/ 1,000 ftrs
        in Enterprise (Constitution Class Cruiser)`,
			expectedTraders: []TraderInfo{
				{
					Name:     "Captain Kirk",
					ShipName: "Enterprise",
					ShipType: "Constitution Class Cruiser",
					Fighters: 1000,
				},
			},
		},
		{
			name: "Multiple traders with ship details",
			sectorData: `Sector  : 5678 in Another Space
Warps to Sector(s) :  400 - 500
Traders : Jean-Luc Picard, w/ 2,500 ftrs
        in USS Enterprise (Galaxy Class Starship)
        Commander Riker, w/ 500 ftrs
        in USS Titan (Luna Class Explorer)`,
			expectedTraders: []TraderInfo{
				{
					Name:     "Jean-Luc Picard",
					ShipName: "USS Enterprise",
					ShipType: "Galaxy Class Starship",
					Fighters: 2500,
				},
				{
					Name:     "Commander Riker",
					ShipName: "USS Titan",
					ShipType: "Luna Class Explorer",
					Fighters: 500,
				},
			},
		},
		{
			name: "Trader without ship details",
			sectorData: `Sector  : 9999 in Empty Space
Warps to Sector(s) :  600
Traders : Unknown Trader, w/ 100 ftrs
Ports   : Some Port, Class 1 (BBS)`,
			expectedTraders: []TraderInfo{
				{
					Name:     "Unknown Trader",
					ShipName: "",
					ShipType: "",
					Fighters: 100,
				},
			},
		},
		{
			name: "Trader with ship name but no type",
			sectorData: `Sector  : 1111 in Test Space
Warps to Sector(s) :  700
Traders : Test Trader, w/ 50 ftrs
        in Mystery Ship`,
			expectedTraders: []TraderInfo{
				{
					Name:     "Test Trader",
					ShipName: "Mystery Ship",
					ShipType: "",
					Fighters: 50,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset parser state
			parser.Reset()
			
			// Process the sector data line by line
			lines := splitIntoLines(tc.sectorData)
			for _, line := range lines {
				parser.ProcessInBound(line + "\r")
			}
			
			// Force sector completion
			parser.sectorCompleted()
			
			// Verify traders
			if len(parser.currentTraders) != len(tc.expectedTraders) {
				t.Errorf("Expected %d traders, got %d", len(tc.expectedTraders), len(parser.currentTraders))
				for i, trader := range parser.currentTraders {
					t.Logf("Trader %d: %+v", i, trader)
				}
				return
			}
			
			for i, expected := range tc.expectedTraders {
				actual := parser.currentTraders[i]
				
				if actual.Name != expected.Name {
					t.Errorf("Trader %d: expected name %q, got %q", i, expected.Name, actual.Name)
				}
				if actual.ShipName != expected.ShipName {
					t.Errorf("Trader %d: expected ship name %q, got %q", i, expected.ShipName, actual.ShipName)
				}
				if actual.ShipType != expected.ShipType {
					t.Errorf("Trader %d: expected ship type %q, got %q", i, expected.ShipType, actual.ShipType)
				}
				if actual.Fighters != expected.Fighters {
					t.Errorf("Trader %d: expected %d fighters, got %d", i, expected.Fighters, actual.Fighters)
				}
			}
		})
	}
}

func TestTraderContinuationEdgeCases(t *testing.T) {
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	parser := NewTWXParser(db, nil)

	t.Run("Trader with comma in ship name", func(t *testing.T) {
		sectorData := `Sector  : 2222 in Test Space
Warps to Sector(s) :  100
Traders : Test Captain, w/ 100 ftrs
        in Ship, The Best One (Custom Type)`
		
		parser.Reset()
		lines := splitIntoLines(sectorData)
		for _, line := range lines {
			parser.ProcessInBound(line + "\r")
		}
		parser.sectorCompleted()
		
		if len(parser.currentTraders) != 1 {
			t.Fatalf("Expected 1 trader, got %d", len(parser.currentTraders))
		}
		
		trader := parser.currentTraders[0]
		if trader.ShipName != "Ship, The Best One" {
			t.Errorf("Expected ship name 'Ship, The Best One', got %q", trader.ShipName)
		}
		if trader.ShipType != "Custom Type" {
			t.Errorf("Expected ship type 'Custom Type', got %q", trader.ShipType)
		}
	})

	t.Run("Trader with parentheses in ship name", func(t *testing.T) {
		sectorData := `Sector  : 3333 in Test Space
Warps to Sector(s) :  200
Traders : Another Captain, w/ 200 ftrs
        in Ship (Mark II) (Advanced Fighter)`
		
		parser.Reset()
		lines := splitIntoLines(sectorData)
		for _, line := range lines {
			parser.ProcessInBound(line + "\r")
		}
		parser.sectorCompleted()
		
		if len(parser.currentTraders) != 1 {
			t.Fatalf("Expected 1 trader, got %d", len(parser.currentTraders))
		}
		
		trader := parser.currentTraders[0]
		if trader.ShipName != "Ship" {
			t.Errorf("Expected ship name 'Ship', got %q", trader.ShipName)
		}
		if trader.ShipType != "Mark II" {
			t.Errorf("Expected ship type 'Mark II', got %q", trader.ShipType)
		}
	})
}

// Helper function to split text into lines
func splitIntoLines(text string) []string {
	var lines []string
	current := ""
	for _, ch := range text {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}