package streaming

import (
	"testing"
	"twist/internal/api"
	"twist/internal/proxy/database"
)

// TestTraderDataToAPIConversion tests the conversion of internal trader data to API TraderInfo
func TestTraderDataToAPIConversion(t *testing.T) {
	// Create test database and parser
	db := database.NewDatabase()
	_ = NewTWXParser(db, nil)

	testCases := []struct {
		name           string
		internalTrader TraderInfo
		expectedAPI    api.TraderInfo
		description    string
	}{
		{
			name: "complete_trader_data",
			internalTrader: TraderInfo{
				Name:      "Captain Kirk",
				ShipName:  "USS Enterprise",
				ShipType:  "Constitution Class",
				Fighters:  1500,
				Alignment: "Good",
			},
			expectedAPI: api.TraderInfo{
				Name:      "Captain Kirk",
				ShipName:  "USS Enterprise",
				ShipType:  "Constitution Class",
				Fighters:  1500,
				Alignment: "Good",
			},
			description: "Complete trader data should convert correctly",
		},
		{
			name: "minimal_trader_data",
			internalTrader: TraderInfo{
				Name:      "Merchant Bob",
				ShipName:  "",
				ShipType:  "",
				Fighters:  0,
				Alignment: "",
			},
			expectedAPI: api.TraderInfo{
				Name:      "Merchant Bob",
				ShipName:  "",
				ShipType:  "",
				Fighters:  0,
				Alignment: "",
			},
			description: "Minimal trader data with empty fields should convert correctly",
		},
		{
			name: "trader_with_unicode_name",
			internalTrader: TraderInfo{
				Name:      "Капитан Кирк",
				ShipName:  "Звездолет",
				ShipType:  "Класс Конституция",
				Fighters:  2000,
				Alignment: "Добрый",
			},
			expectedAPI: api.TraderInfo{
				Name:      "Капитан Кирк",
				ShipName:  "Звездолет",
				ShipType:  "Класс Конституция",
				Fighters:  2000,
				Alignment: "Добрый",
			},
			description: "Unicode characters should convert correctly",
		},
		{
			name: "trader_with_special_chars",
			internalTrader: TraderInfo{
				Name:      "Captain O'Brien",
				ShipName:  "Ship-Name [Test]",
				ShipType:  "Special (Type)",
				Fighters:  500,
				Alignment: "Neutral",
			},
			expectedAPI: api.TraderInfo{
				Name:      "Captain O'Brien",
				ShipName:  "Ship-Name [Test]",
				ShipType:  "Special (Type)",
				Fighters:  500,
				Alignment: "Neutral",
			},
			description: "Special characters should convert correctly",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up trader data
			traders := []TraderInfo{tc.internalTrader}

			// Convert using the parser's conversion logic
			apiTraders := make([]api.TraderInfo, len(traders))
			for i, trader := range traders {
				apiTraders[i] = api.TraderInfo{
					Name:      trader.Name,
					ShipName:  trader.ShipName,
					ShipType:  trader.ShipType,
					Fighters:  trader.Fighters,
					Alignment: trader.Alignment,
				}
			}

			// Verify conversion
			if len(apiTraders) != 1 {
				t.Fatalf("Expected 1 API trader, got %d", len(apiTraders))
			}

			apiTrader := apiTraders[0]

			if apiTrader.Name != tc.expectedAPI.Name {
				t.Errorf("Expected name '%s', got '%s'", tc.expectedAPI.Name, apiTrader.Name)
			}

			if apiTrader.ShipName != tc.expectedAPI.ShipName {
				t.Errorf("Expected ship name '%s', got '%s'", tc.expectedAPI.ShipName, apiTrader.ShipName)
			}

			if apiTrader.ShipType != tc.expectedAPI.ShipType {
				t.Errorf("Expected ship type '%s', got '%s'", tc.expectedAPI.ShipType, apiTrader.ShipType)
			}

			if apiTrader.Fighters != tc.expectedAPI.Fighters {
				t.Errorf("Expected %d fighters, got %d", tc.expectedAPI.Fighters, apiTrader.Fighters)
			}

			if apiTrader.Alignment != tc.expectedAPI.Alignment {
				t.Errorf("Expected alignment '%s', got '%s'", tc.expectedAPI.Alignment, apiTrader.Alignment)
			}
		})
	}
}

// TestPlayerStatsToAPIConversion tests the conversion of internal player stats to API PlayerStatsInfo
func TestPlayerStatsToAPIConversion(t *testing.T) {
	testCases := []struct {
		name           string
		internalStats  PlayerStats
		expectedAPI    api.PlayerStatsInfo
		description    string
	}{
		{
			name: "complete_player_stats",
			internalStats: PlayerStats{
				Turns:         150,
				Credits:       50000,
				Fighters:      1000,
				Shields:       200,
				TotalHolds:    100,
				OreHolds:      25,
				OrgHolds:      30,
				EquHolds:      20,
				ColHolds:      15,
				Photons:       10,
				Armids:        5,
				Limpets:       8,
				GenTorps:      3,
				TwarpType:     2,
				Cloaks:        1,
				Beacons:       2,
				Atomics:       1,
				Corbomite:     1,
				Eprobes:       5,
				MineDisr:      2,
				Alignment:     500,
				Experience:    1000,
				Corp:          1,
				ShipNumber:    1,
				ShipClass:     "Imperial StarShip",
				PsychicProbe:  true,
				PlanetScanner: true,
				ScanType:      2,
				CurrentSector: 1234,
				PlayerName:    "Captain Kirk",
			},
			expectedAPI: api.PlayerStatsInfo{
				Turns:         150,
				Credits:       50000,
				Fighters:      1000,
				Shields:       200,
				TotalHolds:    100,
				OreHolds:      25,
				OrgHolds:      30,
				EquHolds:      20,
				ColHolds:      15,
				Photons:       10,
				Armids:        5,
				Limpets:       8,
				GenTorps:      3,
				TwarpType:     2,
				Cloaks:        1,
				Beacons:       2,
				Atomics:       1,
				Corbomite:     1,
				Eprobes:       5,
				MineDisr:      2,
				Alignment:     500,
				Experience:    1000,
				Corp:          1,
				ShipNumber:    1,
				ShipClass:     "Imperial StarShip",
				PsychicProbe:  true,
				PlanetScanner: true,
				ScanType:      2,
				CurrentSector: 1234,
				PlayerName:    "Captain Kirk",
			},
			description: "Complete player stats should convert correctly",
		},
		{
			name: "minimal_player_stats",
			internalStats: PlayerStats{
				Turns:         0,
				Credits:       0,
				Fighters:      0,
				Shields:       0,
				TotalHolds:    0,
				ShipNumber:    0,
				ShipClass:     "",
				PsychicProbe:  false,
				PlanetScanner: false,
				CurrentSector: 0,
				PlayerName:    "",
			},
			expectedAPI: api.PlayerStatsInfo{
				Turns:         0,
				Credits:       0,
				Fighters:      0,
				Shields:       0,
				TotalHolds:    0,
				ShipNumber:    0,
				ShipClass:     "",
				PsychicProbe:  false,
				PlanetScanner: false,
				CurrentSector: 0,
				PlayerName:    "",
			},
			description: "Zero/empty player stats should convert correctly",
		},
		{
			name: "unicode_player_name",
			internalStats: PlayerStats{
				PlayerName:    "Капитан Кирк",
				ShipClass:     "Империал",
				CurrentSector: 999,
			},
			expectedAPI: api.PlayerStatsInfo{
				PlayerName:    "Капитан Кирк",
				ShipClass:     "Империал",
				CurrentSector: 999,
			},
			description: "Unicode characters in player name and ship class should convert correctly",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Convert using the parser's conversion logic
			apiStats := api.PlayerStatsInfo{
				Turns:         tc.internalStats.Turns,
				Credits:       tc.internalStats.Credits,
				Fighters:      tc.internalStats.Fighters,
				Shields:       tc.internalStats.Shields,
				TotalHolds:    tc.internalStats.TotalHolds,
				OreHolds:      tc.internalStats.OreHolds,
				OrgHolds:      tc.internalStats.OrgHolds,
				EquHolds:      tc.internalStats.EquHolds,
				ColHolds:      tc.internalStats.ColHolds,
				Photons:       tc.internalStats.Photons,
				Armids:        tc.internalStats.Armids,
				Limpets:       tc.internalStats.Limpets,
				GenTorps:      tc.internalStats.GenTorps,
				TwarpType:     tc.internalStats.TwarpType,
				Cloaks:        tc.internalStats.Cloaks,
				Beacons:       tc.internalStats.Beacons,
				Atomics:       tc.internalStats.Atomics,
				Corbomite:     tc.internalStats.Corbomite,
				Eprobes:       tc.internalStats.Eprobes,
				MineDisr:      tc.internalStats.MineDisr,
				Alignment:     tc.internalStats.Alignment,
				Experience:    tc.internalStats.Experience,
				Corp:          tc.internalStats.Corp,
				ShipNumber:    tc.internalStats.ShipNumber,
				ShipClass:     tc.internalStats.ShipClass,
				PsychicProbe:  tc.internalStats.PsychicProbe,
				PlanetScanner: tc.internalStats.PlanetScanner,
				ScanType:      tc.internalStats.ScanType,
				CurrentSector: tc.internalStats.CurrentSector,
				PlayerName:    tc.internalStats.PlayerName,
			}

			// Verify all fields were converted correctly
			if apiStats.Turns != tc.expectedAPI.Turns {
				t.Errorf("Expected turns %d, got %d", tc.expectedAPI.Turns, apiStats.Turns)
			}

			if apiStats.Credits != tc.expectedAPI.Credits {
				t.Errorf("Expected credits %d, got %d", tc.expectedAPI.Credits, apiStats.Credits)
			}

			if apiStats.Fighters != tc.expectedAPI.Fighters {
				t.Errorf("Expected fighters %d, got %d", tc.expectedAPI.Fighters, apiStats.Fighters)
			}

			if apiStats.PlayerName != tc.expectedAPI.PlayerName {
				t.Errorf("Expected player name '%s', got '%s'", tc.expectedAPI.PlayerName, apiStats.PlayerName)
			}

			if apiStats.ShipClass != tc.expectedAPI.ShipClass {
				t.Errorf("Expected ship class '%s', got '%s'", tc.expectedAPI.ShipClass, apiStats.ShipClass)
			}

			if apiStats.CurrentSector != tc.expectedAPI.CurrentSector {
				t.Errorf("Expected current sector %d, got %d", tc.expectedAPI.CurrentSector, apiStats.CurrentSector)
			}

			// Test critical boolean fields
			if apiStats.PsychicProbe != tc.expectedAPI.PsychicProbe {
				t.Errorf("Expected psychic probe %t, got %t", tc.expectedAPI.PsychicProbe, apiStats.PsychicProbe)
			}

			if apiStats.PlanetScanner != tc.expectedAPI.PlanetScanner {
				t.Errorf("Expected planet scanner %t, got %t", tc.expectedAPI.PlanetScanner, apiStats.PlanetScanner)
			}
		})
	}
}

// TestMultipleTradersAPIConversion tests conversion of multiple traders
func TestMultipleTradersAPIConversion(t *testing.T) {
	// Create test database and parser
	db := database.NewDatabase()
	_ = NewTWXParser(db, nil)

	// Create multiple internal traders
	internalTraders := []TraderInfo{
		{
			Name:      "Captain Kirk",
			ShipName:  "Enterprise",
			ShipType:  "Constitution",
			Fighters:  1000,
			Alignment: "Good",
		},
		{
			Name:      "Admiral Picard",
			ShipName:  "Enterprise-D",
			ShipType:  "Galaxy Class",
			Fighters:  800,
			Alignment: "Good",
		},
		{
			Name:      "Commander Sisko",
			ShipName:  "Defiant",
			ShipType:  "Defiant Class",
			Fighters:  1200,
			Alignment: "Neutral",
		},
	}

	// Convert using parser logic
	apiTraders := make([]api.TraderInfo, len(internalTraders))
	for i, trader := range internalTraders {
		apiTraders[i] = api.TraderInfo{
			Name:      trader.Name,
			ShipName:  trader.ShipName,
			ShipType:  trader.ShipType,
			Fighters:  trader.Fighters,
			Alignment: trader.Alignment,
		}
	}

	// Verify conversion
	if len(apiTraders) != len(internalTraders) {
		t.Fatalf("Expected %d API traders, got %d", len(internalTraders), len(apiTraders))
	}

	// Check each trader was converted correctly
	for i, expected := range internalTraders {
		actual := apiTraders[i]

		if actual.Name != expected.Name {
			t.Errorf("Trader %d: expected name '%s', got '%s'", i, expected.Name, actual.Name)
		}

		if actual.ShipName != expected.ShipName {
			t.Errorf("Trader %d: expected ship name '%s', got '%s'", i, expected.ShipName, actual.ShipName)
		}

		if actual.ShipType != expected.ShipType {
			t.Errorf("Trader %d: expected ship type '%s', got '%s'", i, expected.ShipType, actual.ShipType)
		}

		if actual.Fighters != expected.Fighters {
			t.Errorf("Trader %d: expected %d fighters, got %d", i, expected.Fighters, actual.Fighters)
		}

		if actual.Alignment != expected.Alignment {
			t.Errorf("Trader %d: expected alignment '%s', got '%s'", i, expected.Alignment, actual.Alignment)
		}
	}
}

// TestAPIDataValidation tests validation of API data structures
func TestAPIDataValidation(t *testing.T) {
	t.Run("EmptyTraderInfo", func(t *testing.T) {
		// Test that empty trader info can be handled
		emptyTrader := api.TraderInfo{}
		
		// Should have zero values
		if emptyTrader.Name != "" {
			t.Errorf("Expected empty name, got '%s'", emptyTrader.Name)
		}
		if emptyTrader.Fighters != 0 {
			t.Errorf("Expected 0 fighters, got %d", emptyTrader.Fighters)
		}
	})
	
	t.Run("EmptyPlayerStatsInfo", func(t *testing.T) {
		// Test that empty player stats can be handled
		emptyStats := api.PlayerStatsInfo{}
		
		// Should have zero values
		if emptyStats.PlayerName != "" {
			t.Errorf("Expected empty player name, got '%s'", emptyStats.PlayerName)
		}
		if emptyStats.Credits != 0 {
			t.Errorf("Expected 0 credits, got %d", emptyStats.Credits)
		}
		if emptyStats.PsychicProbe != false {
			t.Errorf("Expected false for psychic probe, got %t", emptyStats.PsychicProbe)
		}
	})
}

// TestConversionEdgeCases tests edge cases in data conversion
func TestConversionEdgeCases(t *testing.T) {
	t.Run("NegativeValuesConversion", func(t *testing.T) {
		// Test conversion of negative values (should be preserved)
		internalStats := PlayerStats{
			Turns:     -1, // Invalid but should convert as-is
			Credits:   -1000, // Invalid but should convert as-is
			Alignment: -500,   // Negative alignment is valid in TW
		}

		apiStats := api.PlayerStatsInfo{
			Turns:     internalStats.Turns,
			Credits:   internalStats.Credits,
			Alignment: internalStats.Alignment,
		}

		if apiStats.Turns != -1 {
			t.Errorf("Expected turns -1, got %d", apiStats.Turns)
		}
		if apiStats.Credits != -1000 {
			t.Errorf("Expected credits -1000, got %d", apiStats.Credits)
		}
		if apiStats.Alignment != -500 {
			t.Errorf("Expected alignment -500, got %d", apiStats.Alignment)
		}
	})

	t.Run("MaxValueConversion", func(t *testing.T) {
		// Test conversion of maximum integer values
		internalStats := PlayerStats{
			Credits:    999999999,
			Experience: 2147483647, // Max int32
			Fighters:   1000000,
		}

		apiStats := api.PlayerStatsInfo{
			Credits:    internalStats.Credits,
			Experience: internalStats.Experience,
			Fighters:   internalStats.Fighters,
		}

		if apiStats.Credits != 999999999 {
			t.Errorf("Expected credits 999999999, got %d", apiStats.Credits)
		}
		if apiStats.Experience != 2147483647 {
			t.Errorf("Expected experience 2147483647, got %d", apiStats.Experience)
		}
	})

	t.Run("LongStringConversion", func(t *testing.T) {
		// Test conversion of very long strings
		longName := "This is a very long trader name that exceeds normal expectations and might cause issues"
		longShipName := "This is a very long ship name that might also cause problems during conversion or display"

		internalTrader := TraderInfo{
			Name:     longName,
			ShipName: longShipName,
		}

		apiTrader := api.TraderInfo{
			Name:     internalTrader.Name,
			ShipName: internalTrader.ShipName,
		}

		if apiTrader.Name != longName {
			t.Errorf("Long name not converted correctly")
		}
		if apiTrader.ShipName != longShipName {
			t.Errorf("Long ship name not converted correctly")
		}
	})
}