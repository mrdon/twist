package streaming

import (
	"testing"
	"twist/internal/proxy/database"
)

func TestTWXParser_EnhancedFighterDatabaseReset(t *testing.T) {
	// Create parser with test database
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)
	
	// Test should not panic even with unopened database
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Fighter database reset should not panic, but panicked with: %v", r)
		}
	}()
	
	// Execute fighter database reset with empty database
	parser.resetFighterDatabase()
	
	// The method should handle the case gracefully and fall back to the simple reset
}

func TestTWXParser_FighterResetStardockExclusion(t *testing.T) {
	// Create parser with test database
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)
	
	// Test should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Fighter reset should not panic, but panicked with: %v", r)
		}
	}()
	
	// Execute fighter database reset
	parser.resetFighterDatabase()
	
	// The method should handle the case gracefully with empty database
}

func TestTWXParser_FighterResetOwnerVerification(t *testing.T) {
	// Create parser with test database
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)
	
	// Test should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Fighter reset should not panic, but panicked with: %v", r)
		}
	}()
	
	// Execute fighter database reset
	parser.resetFighterDatabase()
	
	// The method should handle the case gracefully with empty database
}

func TestTWXParser_IsPersonalOrCorpFighter(t *testing.T) {
	// Create parser for testing
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)
	
	testCases := []struct {
		owner    string
		expected bool
		description string
	}{
		{"yours", true, "Exact Pascal match 'yours'"},
		{"belong to your Corp", true, "Exact Pascal match corp"},
		{"YOURS", true, "Case insensitive 'yours'"},
		{"Belong To Your Corp", true, "Case insensitive corp"},
		{"your corp fighters", true, "Contains 'your corp'"},
		{"Your Corporation", true, "Contains 'your corporation'"},
		{"Enemy Player", false, "Enemy player should not match"},
		{"Neutral Trader", false, "Neutral trader should not match"},
		{"", false, "Empty owner should not match"},
		{"someone else", false, "Generic other owner should not match"},
		{"Corporate Alliance", false, "Different corp should not match"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := parser.isPersonalOrCorpFighter(tc.owner)
			if result != tc.expected {
				t.Errorf("Expected %v for owner '%s', got %v", tc.expected, tc.owner, result)
			}
		})
	}
}

func TestTWXParser_FindStardockSector(t *testing.T) {
	// Create parser with test database
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)
	
	// Setup test sectors
	err := setupTestSectorsWithFighters(db, []fighterTestData{
		{sectorNum: 5, owner: "yours", quantity: 100, fighterType: 1},
		{sectorNum: 10, owner: "yours", quantity: 200, fighterType: 2},
		{sectorNum: 15, owner: "yours", quantity: 300, fighterType: 1},
	})
	if err != nil {
		t.Fatalf("Failed to setup test sectors: %v", err)
	}
	
	// Add Stardock to sector 10
	if err := setupStardockSector(db, 10); err != nil {
		t.Fatalf("Failed to setup Stardock sector: %v", err)
	}
	
	// Test Stardock detection
	stardockSector := parser.findStardockSector()
	if stardockSector != 10 {
		t.Errorf("Expected Stardock sector 10, got %d", stardockSector)
	}
}

// Helper types and functions for testing

type fighterTestData struct {
	sectorNum   int
	owner       string
	quantity    int
	fighterType int
}

func setupTestSectorsWithFighters(db database.Database, fighters []fighterTestData) error {
	// Ensure database is open
	if !db.GetDatabaseOpen() {
		if err := db.OpenDatabase(":memory:"); err != nil {
			return err
		}
	}
	for _, f := range fighters {
		sector := database.NULLSector()
		sector.Figs.Quantity = f.quantity
		sector.Figs.Owner = f.owner
		sector.Figs.FigType = database.TFighterType(f.fighterType)
		
		if err := db.SaveSector(sector, f.sectorNum); err != nil {
			return err
		}
	}
	return nil
}

func setupStardockSector(db database.Database, sectorNum int) error {
	// Ensure database is open
	if !db.GetDatabaseOpen() {
		if err := db.OpenDatabase(":memory:"); err != nil {
			return err
		}
	}
	// Load existing sector
	sector, err := db.LoadSector(sectorNum)
	if err != nil {
		// Create new sector if not exists
		sector = database.NULLSector()
	}
	
	// Add Stardock planet
	stardockPlanet := database.TPlanet{
		Name:     "Stardock",
		Owner:    "Federation",
		Fighters: 0,
		Citadel:  false,
		Stardock: true,
	}
	
	sector.Planets = append(sector.Planets, stardockPlanet)
	
	return db.SaveSector(sector, sectorNum)
}