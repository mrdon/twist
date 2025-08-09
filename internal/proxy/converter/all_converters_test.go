package converter

import (
	"path/filepath"
	"testing"
	"time"
	"twist/internal/api"
	"twist/internal/proxy/database"
)

func TestPortConverter_WithDatabase(t *testing.T) {

	// Create temporary database for testing
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_port_converter.db")
	
	db := database.NewDatabase()
	err := db.CreateDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	tests := []struct {
		name        string
		sectorID    int
		dbPort      database.TPort
		expectedInfo *api.PortInfo
	}{
		{
			name:     "BBS Port - Full Integration",
			sectorID: 1,
			dbPort: database.TPort{
				Name:           "Earth Trading Station",
				ClassIndex:     1, // BBS
				BuildTime:      10,
				Dead:           false,
				BuyProduct:     [3]bool{true, true, false}, // Buy FuelOre and Organics
				ProductPercent: [3]int{85, 90, 0},
				ProductAmount:  [3]int{0, 0, 750}, // Selling Equipment
				UpDate:         time.Date(2025, 8, 7, 15, 30, 0, 0, time.UTC),
			},
			expectedInfo: &api.PortInfo{
				SectorID:   1,
				Name:       "Earth Trading Station",
				Class:      1,
				ClassType:  api.PortClassBBS,
				BuildTime:  10,
				Products: []api.ProductInfo{
					{Type: api.ProductTypeFuelOre, Status: api.ProductStatusBuying, Quantity: 0, Percentage: 85},
					{Type: api.ProductTypeOrganics, Status: api.ProductStatusBuying, Quantity: 0, Percentage: 90},
					{Type: api.ProductTypeEquipment, Status: api.ProductStatusSelling, Quantity: 750, Percentage: 0},
				},
				LastUpdate: time.Date(2025, 8, 7, 15, 30, 0, 0, time.UTC),
				Dead:       false,
			},
		},
		{
			name:     "Stardock Integration",
			sectorID: 100,
			dbPort: database.TPort{
				Name:           "Sol Federation Stardock",
				ClassIndex:     9, // STD
				BuildTime:      0,
				Dead:           false,
				BuyProduct:     [3]bool{false, false, false},
				ProductPercent: [3]int{0, 0, 0},
				ProductAmount:  [3]int{0, 0, 0},
				UpDate:         time.Date(2025, 8, 7, 16, 0, 0, 0, time.UTC),
			},
			expectedInfo: &api.PortInfo{
				SectorID:   100,
				Name:       "Sol Federation Stardock",
				Class:      9,
				ClassType:  api.PortClassSTD,
				BuildTime:  0,
				Products: []api.ProductInfo{
					{Type: api.ProductTypeFuelOre, Status: api.ProductStatusNone, Quantity: 0, Percentage: 0},
					{Type: api.ProductTypeOrganics, Status: api.ProductStatusNone, Quantity: 0, Percentage: 0},
					{Type: api.ProductTypeEquipment, Status: api.ProductStatusNone, Quantity: 0, Percentage: 0},
				},
				LastUpdate: time.Date(2025, 8, 7, 16, 0, 0, 0, time.UTC),
				Dead:       false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First create a basic sector to satisfy foreign key constraints
			basicSector := database.NULLSector()
			basicSector.Constellation = "Test Space"
			basicSector.Beacon = "Test beacon"
			basicSector.Warp[0] = tt.sectorID + 1
			
			err := db.SaveSector(basicSector, tt.sectorID)
			if err != nil {
				t.Fatalf("Failed to create sector %d: %v", tt.sectorID, err)
			}
			
			// Save port to database
			err = db.SavePort(tt.dbPort, tt.sectorID)
			if err != nil {
				t.Fatalf("Failed to save port to database: %v", err)
			}

			// Load port from database
			loadedPort, err := db.LoadPort(tt.sectorID)
			if err != nil {
				t.Fatalf("Failed to load port from database: %v", err)
			}

			// Convert loaded port to PortInfo
			result, err := ConvertTPortToPortInfo(tt.sectorID, loadedPort)
			if err != nil {
				t.Fatalf("Conversion failed: %v", err)
			}

			// Validate conversion result
			if result == nil {
				t.Fatal("Expected PortInfo result, got nil")
			}

			// Verify all fields match expected
			validatePortInfo(t, result, tt.expectedInfo)

			// Clean up for next test
			err = db.DeletePort(tt.sectorID)
			if err != nil {
				t.Logf("Warning: Failed to clean up port: %v", err)
			}
		})
	}
}

func TestSectorConverter_WithDatabase(t *testing.T) {

	// Create temporary database for testing
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_sector_converter.db")
	
	db := database.NewDatabase()
	err := db.CreateDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	tests := []struct {
		name         string
		sectorNum    int
		dbSector     database.TSector
		expectedInfo api.SectorInfo
	}{
		{
			name:      "Complex Sector with Multiple Traders",
			sectorNum: 50,
			dbSector: database.TSector{
				Warp:          [6]int{49, 51, 75, 100, 0, 0},
				NavHaz:        35,
				Constellation: "Beta Quadrant",
				Beacon:        "Trade route intersection - safe harbor",
				Traders: []database.TTrader{
					{Name: "Captain Reynolds", ShipType: "Firefly Transport", ShipName: "Serenity", Figs: 100},
					{Name: "Trader Joe", ShipType: "Merchant Cruiser", ShipName: "Profit Margin", Figs: 50},
					{Name: "Admiral Chen", ShipType: "Federation Destroyer", ShipName: "Vigilant", Figs: 500},
				},
				Ships:    []database.TShip{},
				Planets:  []database.TPlanet{},
				Vars:     []database.TSectorVar{},
				Figs:     database.TSpaceObject{Quantity: 0, Owner: "", FigType: database.FtNone},
				MinesArmid: database.TSpaceObject{Quantity: 0, Owner: "", FigType: database.FtNone},
				MinesLimpet: database.TSpaceObject{Quantity: 0, Owner: "", FigType: database.FtNone},
				Anomaly:   false,
				Density:   2500,
				Explored:  database.EtHolo,
				UpDate:    time.Date(2025, 8, 7, 17, 0, 0, 0, time.UTC),
			},
			expectedInfo: api.SectorInfo{
				Number:        50,
				NavHaz:        35,
				HasTraders:    3,
				Constellation: "Beta Quadrant",
				Beacon:        "Trade route intersection - safe harbor",
				Warps:         []int{49, 51, 75, 100},
				HasPort:       false,
			},
		},
		{
			name:      "Dead-end Sector",
			sectorNum: 999,
			dbSector: database.TSector{
				Warp:          [6]int{998, 0, 0, 0, 0, 0},
				NavHaz:        95,
				Constellation: "Void Space",
				Beacon:        "WARNING: Extreme hazard zone",
				Traders:      []database.TTrader{},
				Ships:        []database.TShip{},
				Planets:      []database.TPlanet{},
				Vars:         []database.TSectorVar{},
				Figs:         database.TSpaceObject{Quantity: 0, Owner: "", FigType: database.FtNone},
				MinesArmid:   database.TSpaceObject{Quantity: 0, Owner: "", FigType: database.FtNone},
				MinesLimpet:  database.TSpaceObject{Quantity: 0, Owner: "", FigType: database.FtNone},
				Anomaly:      true,
				Density:      -1,
				Explored:     database.EtNo,
				UpDate:       time.Date(2025, 8, 7, 18, 0, 0, 0, time.UTC),
			},
			expectedInfo: api.SectorInfo{
				Number:        999,
				NavHaz:        95,
				HasTraders:    0,
				Constellation: "Void Space",
				Beacon:        "WARNING: Extreme hazard zone",
				Warps:         []int{998},
				HasPort:       false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save sector to database
			err := db.SaveSector(tt.dbSector, tt.sectorNum)
			if err != nil {
				t.Fatalf("Failed to save sector to database: %v", err)
			}

			// Load sector from database
			loadedSector, err := db.LoadSector(tt.sectorNum)
			if err != nil {
				t.Fatalf("Failed to load sector from database: %v", err)
			}

			// Convert loaded sector to SectorInfo
			result, err := ConvertTSectorToSectorInfo(tt.sectorNum, loadedSector)
			if err != nil {
				t.Fatalf("Conversion failed: %v", err)
			}

			// Validate conversion result
			validateSectorInfo(t, &result, &tt.expectedInfo)
		})
	}
}

func TestPlayerConverter_AllScenarios(t *testing.T) {

	tests := []struct {
		name           string
		currentSector  int
		playerName     string
		expectedPlayer api.PlayerInfo
	}{
		{
			name:          "Active Player",
			currentSector: 42,
			playerName:    "CaptainKirk",
			expectedPlayer: api.PlayerInfo{
				Name:          "CaptainKirk",
				CurrentSector: 42,
			},
		},
		{
			name:          "Anonymous Player",
			currentSector: 1,
			playerName:    "",
			expectedPlayer: api.PlayerInfo{
				Name:          "",
				CurrentSector: 1,
			},
		},
		{
			name:          "High Sector Number",
			currentSector: 9999,
			playerName:    "ExplorerX",
			expectedPlayer: api.PlayerInfo{
				Name:          "ExplorerX",
				CurrentSector: 9999,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to PlayerInfo
			result := ConvertToPlayerInfo(tt.currentSector, tt.playerName)

			// Validate conversion result
			if result.Name != tt.expectedPlayer.Name {
				t.Errorf("Name: expected %s, got %s", tt.expectedPlayer.Name, result.Name)
			}
			if result.CurrentSector != tt.expectedPlayer.CurrentSector {
				t.Errorf("CurrentSector: expected %d, got %d", tt.expectedPlayer.CurrentSector, result.CurrentSector)
			}
		})
	}
}

// Helper functions for validation

func validatePortInfo(t *testing.T, actual, expected *api.PortInfo) {
	if actual.SectorID != expected.SectorID {
		t.Errorf("SectorID: expected %d, got %d", expected.SectorID, actual.SectorID)
	}
	if actual.Name != expected.Name {
		t.Errorf("Name: expected %s, got %s", expected.Name, actual.Name)
	}
	if actual.Class != expected.Class {
		t.Errorf("Class: expected %d, got %d", expected.Class, actual.Class)
	}
	if actual.ClassType != expected.ClassType {
		t.Errorf("ClassType: expected %v, got %v", expected.ClassType, actual.ClassType)
	}
	if actual.BuildTime != expected.BuildTime {
		t.Errorf("BuildTime: expected %d, got %d", expected.BuildTime, actual.BuildTime)
	}
	if actual.Dead != expected.Dead {
		t.Errorf("Dead: expected %t, got %t", expected.Dead, actual.Dead)
	}
	// For integration tests, just check that LastUpdate is not zero (database sets this)
	if actual.LastUpdate.IsZero() {
		t.Errorf("LastUpdate should not be zero, got %v", actual.LastUpdate)
	}

	// Validate products
	if len(actual.Products) != len(expected.Products) {
		t.Errorf("Products length: expected %d, got %d", len(expected.Products), len(actual.Products))
		return
	}

	for i, expectedProduct := range expected.Products {
		if i >= len(actual.Products) {
			t.Errorf("Missing product at index %d", i)
			continue
		}

		actualProduct := actual.Products[i]
		if actualProduct.Type != expectedProduct.Type {
			t.Errorf("Product %d Type: expected %v, got %v", i, expectedProduct.Type, actualProduct.Type)
		}
		if actualProduct.Status != expectedProduct.Status {
			t.Errorf("Product %d Status: expected %v, got %v", i, expectedProduct.Status, actualProduct.Status)
		}
		if actualProduct.Quantity != expectedProduct.Quantity {
			t.Errorf("Product %d Quantity: expected %d, got %d", i, expectedProduct.Quantity, actualProduct.Quantity)
		}
		if actualProduct.Percentage != expectedProduct.Percentage {
			t.Errorf("Product %d Percentage: expected %d, got %d", i, expectedProduct.Percentage, actualProduct.Percentage)
		}
	}
}

func validateSectorInfo(t *testing.T, actual, expected *api.SectorInfo) {
	if actual.Number != expected.Number {
		t.Errorf("Number: expected %d, got %d", expected.Number, actual.Number)
	}
	if actual.NavHaz != expected.NavHaz {
		t.Errorf("NavHaz: expected %d, got %d", expected.NavHaz, actual.NavHaz)
	}
	if actual.HasTraders != expected.HasTraders {
		t.Errorf("HasTraders: expected %d, got %d", expected.HasTraders, actual.HasTraders)
	}
	if actual.Constellation != expected.Constellation {
		t.Errorf("Constellation: expected %s, got %s", expected.Constellation, actual.Constellation)
	}
	if actual.Beacon != expected.Beacon {
		t.Errorf("Beacon: expected %s, got %s", expected.Beacon, actual.Beacon)
	}
	if actual.HasPort != expected.HasPort {
		t.Errorf("HasPort: expected %t, got %t", expected.HasPort, actual.HasPort)
	}

	// Validate warps array
	if len(actual.Warps) != len(expected.Warps) {
		t.Errorf("Warps length: expected %d, got %d", len(expected.Warps), len(actual.Warps))
		t.Errorf("Expected warps: %v, got warps: %v", expected.Warps, actual.Warps)
		return
	}

	for i, expectedWarp := range expected.Warps {
		if i >= len(actual.Warps) {
			t.Errorf("Missing warp at index %d", i)
			continue
		}
		if actual.Warps[i] != expectedWarp {
			t.Errorf("Warp %d: expected %d, got %d", i, expectedWarp, actual.Warps[i])
		}
	}
}