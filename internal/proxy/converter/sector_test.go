package converter

import (
	"testing"
	"twist/internal/api"
	"twist/internal/proxy/database"
)

func TestConvertTSectorToSectorInfo(t *testing.T) {

	tests := []struct {
		name      string
		sectorNum int
		dbSector  database.TSector
		expected  api.SectorInfo
		shouldErr bool
	}{
		{
			name:      "Basic sector with warps",
			sectorNum: 1,
			dbSector: database.TSector{
				Warp:          [6]int{2, 5, 0, 0, 0, 0},
				NavHaz:        25,
				Warps:         2,
				Constellation: "Sol",
				Beacon:        "Welcome to Earth space",
				Traders:       []database.TTrader{{Name: "Bob", ShipType: "Merchant"}},
			},
			expected: api.SectorInfo{
				Number:        1,
				NavHaz:        25,
				HasTraders:    1,
				Constellation: "Sol",
				Beacon:        "Welcome to Earth space",
				Warps:         []int{2, 5},
				HasPort:       false,
			},
			shouldErr: false,
		},
		{
			name:      "Sector with maximum warps",
			sectorNum: 100,
			dbSector: database.TSector{
				Warp:          [6]int{99, 101, 150, 200, 300, 400},
				NavHaz:        0,
				Warps:         6,
				Constellation: "Alpha Centauri",
				Beacon:        "Major hub sector",
				Traders:       []database.TTrader{},
			},
			expected: api.SectorInfo{
				Number:        100,
				NavHaz:        0,
				HasTraders:    0,
				Constellation: "Alpha Centauri",
				Beacon:        "Major hub sector",
				Warps:         []int{99, 101, 150, 200, 300, 400},
				HasPort:       false,
			},
			shouldErr: false,
		},
		{
			name:      "Dead-end sector with no warps",
			sectorNum: 999,
			dbSector: database.TSector{
				Warp:          [6]int{0, 0, 0, 0, 0, 0},
				NavHaz:        100,
				Warps:         0,
				Constellation: "Unknown",
				Beacon:        "",
				Traders:       []database.TTrader{},
			},
			expected: api.SectorInfo{
				Number:        999,
				NavHaz:        100,
				HasTraders:    0,
				Constellation: "Unknown",
				Beacon:        "",
				Warps:         []int{},
				HasPort:       false,
			},
			shouldErr: false,
		},
		{
			name:      "Sector with partial warps and multiple traders",
			sectorNum: 50,
			dbSector: database.TSector{
				Warp:          [6]int{49, 51, 75, 0, 0, 0},
				NavHaz:        50,
				Warps:         3,
				Constellation: "Beta Sector",
				Beacon:        "Trade route intersection",
				Traders: []database.TTrader{
					{Name: "Trader Alpha", ShipType: "Freighter"},
					{Name: "Trader Beta", ShipType: "Scout"},
					{Name: "Trader Gamma", ShipType: "Merchant"},
				},
			},
			expected: api.SectorInfo{
				Number:        50,
				NavHaz:        50,
				HasTraders:    3,
				Constellation: "Beta Sector",
				Beacon:        "Trade route intersection",
				Warps:         []int{49, 51, 75},
				HasPort:       false,
			},
			shouldErr: false,
		},
		{
			name:      "Sector with invalid warp handling",
			sectorNum: 25,
			dbSector: database.TSector{
				Warp:          [6]int{24, 26, -1, 0, 999999, 0}, // Test negative and large values
				NavHaz:        15,
				Warps:         5, // Claims 5 warps but only first 2 are valid
				Constellation: "Test Sector",
				Beacon:        "Test beacon",
				Traders:       []database.TTrader{},
			},
			expected: api.SectorInfo{
				Number:        25,
				NavHaz:        15,
				HasTraders:    0,
				Constellation: "Test Sector",
				Beacon:        "Test beacon",
				Warps:         []int{24, 26, 999999}, // Only positive values should be included
				HasPort:       false,
			},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertTSectorToSectorInfo(tt.sectorNum, tt.dbSector)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

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
				t.Errorf("Expected warps: %v, got warps: %v", tt.expected.Warps, result.Warps)
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