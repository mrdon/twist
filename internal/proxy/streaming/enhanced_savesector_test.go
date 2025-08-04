package streaming

import (
	"testing"
	"twist/internal/proxy/database"
)

func TestSaveSectorWithCollections(t *testing.T) {
	// Create test database
	db := database.NewDatabase()
	err := db.CreateDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	// Create test sector
	sector := database.NULLSector()
	sector.Constellation = "Test Space"
	sector.Beacon = "Test Beacon"

	// Create test collections
	ships := []database.TShip{
		{Name: "Enterprise", Owner: "Kirk", ShipType: "Heavy Cruiser", Figs: 100},
		{Name: "Reliant", Owner: "Khan", ShipType: "Frigate", Figs: 50},
	}

	traders := []database.TTrader{
		{Name: "Spock", ShipType: "Scout", ShipName: "Science Vessel", Figs: 25},
	}

	planets := []database.TPlanet{
		{Name: "Vulcan", Owner: "Federation", Fighters: 200, Citadel: true, Stardock: false},
		{Name: "Stardock", Owner: "Federation", Fighters: 0, Citadel: false, Stardock: true},
	}

	// Test SaveSectorWithCollections
	err = db.SaveSectorWithCollections(sector, 1, ships, traders, planets)
	if err != nil {
		t.Fatalf("SaveSectorWithCollections failed: %v", err)
	}

	// Verify sector was saved
	loadedSector, err := db.LoadSector(1)
	if err != nil {
		t.Fatalf("Failed to load sector: %v", err)
	}

	// Verify basic sector data
	if loadedSector.Constellation != "Test Space" {
		t.Errorf("Expected constellation 'Test Space', got '%s'", loadedSector.Constellation)
	}

	if loadedSector.Beacon != "Test Beacon" {
		t.Errorf("Expected beacon 'Test Beacon', got '%s'", loadedSector.Beacon)
	}

	// Verify collections were saved
	if len(loadedSector.Ships) != 2 {
		t.Errorf("Expected 2 ships, got %d", len(loadedSector.Ships))
	} else {
		ship1 := loadedSector.Ships[0]
		if ship1.Name != "Enterprise" || ship1.Owner != "Kirk" {
			t.Errorf("Ship 1 data incorrect: %+v", ship1)
		}
	}

	if len(loadedSector.Traders) != 1 {
		t.Errorf("Expected 1 trader, got %d", len(loadedSector.Traders))
	} else {
		trader := loadedSector.Traders[0]
		if trader.Name != "Spock" || trader.ShipType != "Scout" {
			t.Errorf("Trader data incorrect: %+v", trader)
		}
	}

	if len(loadedSector.Planets) != 2 {
		t.Errorf("Expected 2 planets, got %d", len(loadedSector.Planets))
	} else {
		planet1 := loadedSector.Planets[0]
		if planet1.Name != "Vulcan" || !planet1.Citadel {
			t.Errorf("Planet 1 data incorrect: %+v", planet1)
		}
		planet2 := loadedSector.Planets[1]
		if planet2.Name != "Stardock" || !planet2.Stardock {
			t.Errorf("Planet 2 data incorrect: %+v", planet2)
		}
	}

	t.Log("✓ SaveSectorWithCollections works correctly with Pascal-compliant signature")
}

func TestSaveSectorWithCollectionsVsRegularSaveSector(t *testing.T) {
	// Create test database
	db := database.NewDatabase()
	err := db.CreateDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	// Test that both methods produce the same result
	sector := database.NULLSector()
	sector.Constellation = "Comparison Test"
	sector.Ships = []database.TShip{
		{Name: "TestShip", Owner: "TestOwner", ShipType: "TestType", Figs: 75},
	}
	sector.Traders = []database.TTrader{
		{Name: "TestTrader", ShipType: "TestTraderType", ShipName: "TestTraderShip", Figs: 30},
	}
	sector.Planets = []database.TPlanet{
		{Name: "TestPlanet", Owner: "TestPlanetOwner", Fighters: 150, Citadel: false, Stardock: false},
	}

	// Save using regular SaveSector (sector 2)
	err = db.SaveSector(sector, 2)
	if err != nil {
		t.Fatalf("Regular SaveSector failed: %v", err)
	}

	// Save using SaveSectorWithCollections (sector 3)
	err = db.SaveSectorWithCollections(sector, 3, sector.Ships, sector.Traders, sector.Planets)
	if err != nil {
		t.Fatalf("SaveSectorWithCollections failed: %v", err)
	}

	// Load both sectors
	sector2, err := db.LoadSector(2)
	if err != nil {
		t.Fatalf("Failed to load sector 2: %v", err)
	}

	sector3, err := db.LoadSector(3)
	if err != nil {
		t.Fatalf("Failed to load sector 3: %v", err)
	}

	// Compare results
	if sector2.Constellation != sector3.Constellation {
		t.Errorf("Constellation mismatch: regular='%s', with collections='%s'", sector2.Constellation, sector3.Constellation)
	}

	if len(sector2.Ships) != len(sector3.Ships) {
		t.Errorf("Ships count mismatch: regular=%d, with collections=%d", len(sector2.Ships), len(sector3.Ships))
	}

	if len(sector2.Traders) != len(sector3.Traders) {
		t.Errorf("Traders count mismatch: regular=%d, with collections=%d", len(sector2.Traders), len(sector3.Traders))
	}

	if len(sector2.Planets) != len(sector3.Planets) {
		t.Errorf("Planets count mismatch: regular=%d, with collections=%d", len(sector2.Planets), len(sector3.Planets))
	}

	t.Log("✓ Both SaveSector methods produce equivalent results")
}