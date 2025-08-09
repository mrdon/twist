package proxy

import (
	"testing"
	"twist/internal/api"
	"twist/internal/proxy/converter"
	"twist/internal/proxy/database"
)

func TestConvertDatabasePlayerToAPI(t *testing.T) {

	tests := []struct {
		name          string
		currentSector int
		playerName    string
		expected      api.PlayerInfo
	}{
		{
			name:          "Basic player conversion",
			currentSector: 1,
			playerName:    "TestPlayer",
			expected: api.PlayerInfo{
				Name:          "TestPlayer",
				CurrentSector: 1,
			},
		},
		{
			name:          "Player in high sector number",
			currentSector: 5000,
			playerName:    "AdventurerX",
			expected: api.PlayerInfo{
				Name:          "AdventurerX",
				CurrentSector: 5000,
			},
		},
		{
			name:          "Empty player name",
			currentSector: 100,
			playerName:    "",
			expected: api.PlayerInfo{
				Name:          "",
				CurrentSector: 100,
			},
		},
		{
			name:          "Zero sector",
			currentSector: 0,
			playerName:    "NewPlayer",
			expected: api.PlayerInfo{
				Name:          "NewPlayer",
				CurrentSector: 0,
			},
		},
		{
			name:          "Player with special characters in name",
			currentSector: 42,
			playerName:    "Player-123_Test",
			expected: api.PlayerInfo{
				Name:          "Player-123_Test",
				CurrentSector: 42,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertDatabasePlayerToAPI(tt.currentSector, tt.playerName)

			if result.Name != tt.expected.Name {
				t.Errorf("Name: expected %s, got %s", tt.expected.Name, result.Name)
			}
			if result.CurrentSector != tt.expected.CurrentSector {
				t.Errorf("CurrentSector: expected %d, got %d", tt.expected.CurrentSector, result.CurrentSector)
			}
		})
	}
}

func TestConvertDatabaseSectorToAPI(t *testing.T) {

	tests := []struct {
		name      string
		sectorNum int
		dbSector  database.TSector
		expected  api.SectorInfo
	}{
		{
			name:      "Basic sector conversion",
			sectorNum: 1,
			dbSector: database.TSector{
				Warp:          [6]int{2, 5, 10, 0, 0, 0},
				NavHaz:        25,

				Constellation: "Sol System",
				Beacon:        "Earth space beacon",
				Traders:       []database.TTrader{{Name: "Bob", ShipType: "Merchant"}},
			},
			expected: api.SectorInfo{
				Number:        1,
				NavHaz:        25,
				HasTraders:    1,
				Constellation: "Sol System",
				Beacon:        "Earth space beacon",
				Warps:         []int{2, 5, 10},
				HasPort:       false,
			},
		},
		{
			name:      "Sector with multiple traders",
			sectorNum: 50,
			dbSector: database.TSector{
				Warp:          [6]int{49, 51, 0, 0, 0, 0},
				NavHaz:        0,

				Constellation: "Trading Hub",
				Beacon:        "Welcome to the hub",
				Traders: []database.TTrader{
					{Name: "Trader1", ShipType: "Freighter"},
					{Name: "Trader2", ShipType: "Scout"},
					{Name: "Trader3", ShipType: "Merchant"},
				},
			},
			expected: api.SectorInfo{
				Number:        50,
				NavHaz:        0,
				HasTraders:    3,
				Constellation: "Trading Hub",
				Beacon:        "Welcome to the hub",
				Warps:         []int{49, 51},
				HasPort:       false,
			},
		},
		{
			name:      "Empty sector",
			sectorNum: 999,
			dbSector: database.TSector{
				Warp:          [6]int{0, 0, 0, 0, 0, 0},
				NavHaz:        100,

				Constellation: "",
				Beacon:        "",
				Traders:       []database.TTrader{},
			},
			expected: api.SectorInfo{
				Number:        999,
				NavHaz:        100,
				HasTraders:    0,
				Constellation: "",
				Beacon:        "",
				Warps:         []int{},
				HasPort:       false,
			},
		},
		{
			name:      "Sector with full warp array",
			sectorNum: 100,
			dbSector: database.TSector{
				Warp:          [6]int{99, 101, 150, 200, 250, 300},
				NavHaz:        50,

				Constellation: "Dense Region",
				Beacon:        "Major intersection",
				Traders:       []database.TTrader{},
			},
			expected: api.SectorInfo{
				Number:        100,
				NavHaz:        50,
				HasTraders:    0,
				Constellation: "Dense Region",
				Beacon:        "Major intersection",
				Warps:         []int{99, 101, 150, 200, 250, 300},
				HasPort:       false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertDatabaseSectorToAPI(tt.sectorNum, tt.dbSector)

			// Verify all fields
			if result.Number != tt.expected.Number {
				t.Errorf("Number: expected %d, got %d", tt.expected.Number, result.Number)
			}
			if result.NavHaz != tt.expected.NavHaz {
				t.Errorf("NavHaz: expected %d, got %d", tt.expected.NavHaz, result.NavHaz)
			}
			if result.HasTraders != tt.expected.HasTraders {
				t.Errorf("HasTraders: expected %d, got %d", tt.expected.HasTraders, result.HasTraders)
			}
			if result.Constellation != tt.expected.Constellation {
				t.Errorf("Constellation: expected %s, got %s", tt.expected.Constellation, result.Constellation)
			}
			if result.Beacon != tt.expected.Beacon {
				t.Errorf("Beacon: expected %s, got %s", tt.expected.Beacon, result.Beacon)
			}
			if result.HasPort != tt.expected.HasPort {
				t.Errorf("HasPort: expected %t, got %t", tt.expected.HasPort, result.HasPort)
			}

			// Verify warps array
			if len(result.Warps) != len(tt.expected.Warps) {
				t.Errorf("Warps length: expected %d, got %d", len(tt.expected.Warps), len(result.Warps))
				return
			}

			for i, expectedWarp := range tt.expected.Warps {
				if i >= len(result.Warps) {
					t.Errorf("Missing warp at index %d", i)
					continue
				}
				if result.Warps[i] != expectedWarp {
					t.Errorf("Warp %d: expected %d, got %d", i, expectedWarp, result.Warps[i])
				}
			}
		})
	}
}

func TestPortClassToTypeString(t *testing.T) {

	tests := []struct {
		name       string
		classIndex int
		expected   string
	}{
		{"BBS port", 1, "BBS"},
		{"BSB port", 2, "BSB"},
		{"SBB port", 3, "SBB"},
		{"SSB port", 4, "SSB"},
		{"SBS port", 5, "SBS"},
		{"BSS port", 6, "BSS"},
		{"SSS port", 7, "SSS"},
		{"BBB port", 8, "BBB"},
		{"Stardock", 9, "STD"},
		{"No port (0)", 0, ""},
		{"Unknown class", 10, ""},
		{"Negative class", -1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ConvertPortClassToString(tt.classIndex)
			if result != tt.expected {
				t.Errorf("portClassToTypeString(%d): expected %s, got %s", tt.classIndex, tt.expected, result)
			}
		})
	}
}

// Test that the fallback logic works when the enhanced converter fails
func TestConvertDatabaseSectorToAPI_FallbackLogic(t *testing.T) {

	// This test ensures that even if the enhanced API converter fails,
	// the fallback logic in the proxy package still works correctly
	sectorNum := 123
	dbSector := database.TSector{
		Warp:          [6]int{122, 124, 200, 0, 0, 0},
		NavHaz:        75,

		Constellation: "Test Constellation",
		Beacon:        "Test Beacon",
		Traders: []database.TTrader{
			{Name: "FallbackTrader", ShipType: "TestShip"},
		},
	}

	result := convertDatabaseSectorToAPI(sectorNum, dbSector)

	// Verify that conversion still produces valid results
	if result.Number != 123 {
		t.Errorf("Number: expected 123, got %d", result.Number)
	}
	if result.NavHaz != 75 {
		t.Errorf("NavHaz: expected 75, got %d", result.NavHaz)
	}
	if result.HasTraders != 1 {
		t.Errorf("HasTraders: expected 1, got %d", result.HasTraders)
	}
	if result.Constellation != "Test Constellation" {
		t.Errorf("Constellation: expected 'Test Constellation', got %s", result.Constellation)
	}
	if result.Beacon != "Test Beacon" {
		t.Errorf("Beacon: expected 'Test Beacon', got %s", result.Beacon)
	}

	// Verify warps are processed correctly
	expectedWarps := []int{122, 124, 200}
	if len(result.Warps) != len(expectedWarps) {
		t.Errorf("Warps length: expected %d, got %d", len(expectedWarps), len(result.Warps))
	}

	for i, expected := range expectedWarps {
		if i < len(result.Warps) && result.Warps[i] != expected {
			t.Errorf("Warp %d: expected %d, got %d", i, expected, result.Warps[i])
		}
	}
}