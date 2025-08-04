package streaming

import (
	"testing"
	"twist/internal/proxy/database"
)

func TestTWXParser_EnhancedPlanetParsing(t *testing.T) {
	// Create parser
	db := database.NewDatabase()
	parser := NewTWXParser(db)
	
	// Test Pascal-compliant planet parsing
	testCases := []struct {
		name           string
		planetLine     string
		expectedPlanet PlanetInfo
		description    string
	}{
		{
			name:       "basic_planet_with_owner",
			planetLine: "Planets : Terra [Owned by Federation]",
			expectedPlanet: PlanetInfo{
				Name:     "Terra",
				Owner:    "Federation",
				Fighters: 0,
				Citadel:  false,
				Stardock: false,
			},
			description: "Basic planet with owner in brackets",
		},
		{
			name:       "planet_with_fighters",
			planetLine: "Planets : Mars [Owned by Earth Alliance], w/ 2,500 ftrs",
			expectedPlanet: PlanetInfo{
				Name:     "Mars",
				Owner:    "Earth Alliance",
				Fighters: 2500,
				Citadel:  false,
				Stardock: false,
			},
			description: "Planet with fighter count",
		},
		{
			name:       "stardock_detection",
			planetLine: "Planets : Alpha Centauri Stardock",
			expectedPlanet: PlanetInfo{
				Name:     "Alpha Centauri",
				Owner:    "",
				Fighters: 0,
				Citadel:  false,
				Stardock: true,
			},
			description: "Planet with Stardock detection",
		},
		{
			name:       "standalone_stardock",
			planetLine: "Planets : Stardock",
			expectedPlanet: PlanetInfo{
				Name:     "Stardock",
				Owner:    "",
				Fighters: 0,
				Citadel:  false,
				Stardock: true,
			},
			description: "Standalone Stardock",
		},
		{
			name:       "citadel_detection",
			planetLine: "Planets : Fortress Citadel [Owned by Empire]",
			expectedPlanet: PlanetInfo{
				Name:     "Fortress Citadel",
				Owner:    "Empire",
				Fighters: 0,
				Citadel:  true,
				Stardock: false,
			},
			description: "Planet with Citadel detection",
		},
		{
			name:       "citadel_abbreviation",
			planetLine: "Planets : Defense Cit [Owned by Rebels]",
			expectedPlanet: PlanetInfo{
				Name:     "Defense Cit",
				Owner:    "Rebels",
				Fighters: 0,
				Citadel:  true,
				Stardock: false,
			},
			description: "Planet with Citadel abbreviation",
		},
		{
			name:       "multiple_planets",
			planetLine: "Planets : Earth [Owned by Humans], Mars, Stardock",
			expectedPlanet: PlanetInfo{
				Name:     "Earth",
				Owner:    "Humans",
				Fighters: 0,
				Citadel:  false,
				Stardock: false,
			},
			description: "First planet from multiple planets list",
		},
		{
			name:       "planet_no_owner_brackets",
			planetLine: "Planets : Unnamed Planet",
			expectedPlanet: PlanetInfo{
				Name:     "Unnamed Planet",
				Owner:    "",
				Fighters: 0,
				Citadel:  false,
				Stardock: false,
			},
			description: "Planet without owner brackets",
		},
		{
			name:       "planet_direct_owner",
			planetLine: "Planets : Colony [Player]",
			expectedPlanet: PlanetInfo{
				Name:     "Colony",
				Owner:    "Player",
				Fighters: 0,
				Citadel:  false,
				Stardock: false,
			},
			description: "Planet with direct owner (no 'Owned by' prefix)",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset parser state
			parser.currentPlanets = nil
			parser.sectorPosition = SectorPosNormal
			
			// Process planet line
			parser.parseSectorPlanets(tc.planetLine)
			
			// Verify results
			if len(parser.currentPlanets) == 0 {
				t.Error("Expected at least 1 planet, got 0")
				return
			}
			
			planet := parser.currentPlanets[0]
			
			if planet.Name != tc.expectedPlanet.Name {
				t.Errorf("Expected planet name '%s', got '%s'", tc.expectedPlanet.Name, planet.Name)
			}
			
			if planet.Owner != tc.expectedPlanet.Owner {
				t.Errorf("Expected planet owner '%s', got '%s'", tc.expectedPlanet.Owner, planet.Owner)
			}
			
			if planet.Fighters != tc.expectedPlanet.Fighters {
				t.Errorf("Expected %d fighters, got %d", tc.expectedPlanet.Fighters, planet.Fighters)
			}
			
			if planet.Citadel != tc.expectedPlanet.Citadel {
				t.Errorf("Expected citadel %v, got %v", tc.expectedPlanet.Citadel, planet.Citadel)
			}
			
			if planet.Stardock != tc.expectedPlanet.Stardock {
				t.Errorf("Expected stardock %v, got %v", tc.expectedPlanet.Stardock, planet.Stardock)
			}
		})
	}
}

func TestTWXParser_MultiplePlanetsParsing(t *testing.T) {
	// Test parsing multiple planets in one line
	db := database.NewDatabase()
	parser := NewTWXParser(db)
	
	// Reset parser state
	parser.currentPlanets = nil
	parser.sectorPosition = SectorPosNormal
	
	// Process line with multiple planets
	parser.parseSectorPlanets("Planets : Earth [Owned by Humans], Mars [Owned by Colonists], Stardock")
	
	// Verify we have 3 planets
	expectedPlanetCount := 3
	if len(parser.currentPlanets) != expectedPlanetCount {
		t.Errorf("Expected %d planets, got %d", expectedPlanetCount, len(parser.currentPlanets))
		for i, planet := range parser.currentPlanets {
			t.Errorf("Planet %d: %+v", i, planet)
		}
		return
	}
	
	// Verify first planet
	planet1 := parser.currentPlanets[0]
	if planet1.Name != "Earth" || planet1.Owner != "Humans" {
		t.Errorf("First planet not parsed correctly: %+v", planet1)
	}
	
	// Verify second planet
	planet2 := parser.currentPlanets[1]
	if planet2.Name != "Mars" || planet2.Owner != "Colonists" {
		t.Errorf("Second planet not parsed correctly: %+v", planet2)
	}
	
	// Verify third planet (Stardock)
	planet3 := parser.currentPlanets[2]
	if planet3.Name != "Stardock" || !planet3.Stardock {
		t.Errorf("Third planet (Stardock) not parsed correctly: %+v", planet3)
	}
}

func TestTWXParser_PlanetCitadelStardockDetection(t *testing.T) {
	// Test various citadel and stardock detection scenarios
	db := database.NewDatabase()
	parser := NewTWXParser(db)
	
	testCases := []struct {
		name            string
		planetLine      string
		expectedCitadel bool
		expectedStardock bool
		description     string
	}{
		{
			name:             "citadel_full_word",
			planetLine:       "Planets : Defense Citadel",
			expectedCitadel:  true,
			expectedStardock: false,
			description:      "Full 'Citadel' word detection",
		},
		{
			name:             "citadel_abbreviation_end",
			planetLine:       "Planets : Military Cit",
			expectedCitadel:  true,
			expectedStardock: false,
			description:      "Citadel abbreviation at end",
		},
		{
			name:             "stardock_full_word",
			planetLine:       "Planets : Commerce Stardock",
			expectedCitadel:  false,
			expectedStardock: true,
			description:      "Full 'Stardock' word detection",
		},
		{
			name:             "stardock_abbreviation",
			planetLine:       "Planets : Trading SD",
			expectedCitadel:  false,
			expectedStardock: true,
			description:      "Stardock abbreviation detection",
		},
		{
			name:             "both_citadel_stardock",
			planetLine:       "Planets : Fortress Citadel Stardock",
			expectedCitadel:  true,
			expectedStardock: true,
			description:      "Both citadel and stardock detected",
		},
		{
			name:             "neither_special",
			planetLine:       "Planets : Regular Planet",
			expectedCitadel:  false,
			expectedStardock: false,
			description:      "Regular planet with no special flags",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset parser state
			parser.currentPlanets = nil
			parser.sectorPosition = SectorPosNormal
			
			// Process planet line
			parser.parseSectorPlanets(tc.planetLine)
			
			// Verify results
			if len(parser.currentPlanets) != 1 {
				t.Errorf("Expected 1 planet, got %d", len(parser.currentPlanets))
				return
			}
			
			planet := parser.currentPlanets[0]
			
			if planet.Citadel != tc.expectedCitadel {
				t.Errorf("Expected citadel %v, got %v for planet: %s", tc.expectedCitadel, planet.Citadel, planet.Name)
			}
			
			if planet.Stardock != tc.expectedStardock {
				t.Errorf("Expected stardock %v, got %v for planet: %s", tc.expectedStardock, planet.Stardock, planet.Name)
			}
		})
	}
}

func TestTWXParser_PlanetContinuationParsing(t *testing.T) {
	// Test planet continuation line parsing
	db := database.NewDatabase()
	parser := NewTWXParser(db)
	
	// Reset parser state
	parser.currentPlanets = nil
	parser.sectorPosition = SectorPosNormal
	
	// Process main planet line
	parser.parseSectorPlanets("Planets : Earth [Owned by Humans]")
	
	// Process continuation line
	parser.handleSectorContinuation("        Mars [Owned by Colonists], Stardock")
	
	// Verify we have 3 planets total
	expectedPlanetCount := 3
	if len(parser.currentPlanets) != expectedPlanetCount {
		t.Errorf("Expected %d planets, got %d", expectedPlanetCount, len(parser.currentPlanets))
		for i, planet := range parser.currentPlanets {
			t.Errorf("Planet %d: %+v", i, planet)
		}
		return
	}
	
	// Verify first planet (from main line)
	planet1 := parser.currentPlanets[0]
	if planet1.Name != "Earth" || planet1.Owner != "Humans" {
		t.Errorf("First planet not parsed correctly: %+v", planet1)
	}
	
	// Verify second planet (from continuation)
	planet2 := parser.currentPlanets[1]
	if planet2.Name != "Mars" || planet2.Owner != "Colonists" {
		t.Errorf("Second planet not parsed correctly: %+v", planet2)
	}
	
	// Verify third planet (Stardock from continuation)
	planet3 := parser.currentPlanets[2]
	if planet3.Name != "Stardock" || !planet3.Stardock {
		t.Errorf("Third planet (Stardock) not parsed correctly: %+v", planet3)
	}
}

func TestTWXParser_PlanetParsingValidation(t *testing.T) {
	// Test validation and error handling
	db := database.NewDatabase()
	parser := NewTWXParser(db)
	
	testCases := []struct {
		name        string
		planetLine  string
		description string
		shouldPanic bool
	}{
		{
			name:        "empty_planet_line",
			planetLine:  "Planets : ",
			description: "Empty planet line should not crash",
			shouldPanic: false,
		},
		{
			name:        "malformed_brackets",
			planetLine:  "Planets : TestPlanet [Owned by Player",
			description: "Malformed brackets should not crash",
			shouldPanic: false,
		},
		{
			name:        "unicode_planet_name",
			planetLine:  "Planets : Планета [Owned by Игрок]",
			description: "Unicode characters in planet name",
			shouldPanic: false,
		},
		{
			name:        "special_characters",
			planetLine:  "Planets : Planet-X [Owned by Corp_Name]",
			description: "Special characters in names",
			shouldPanic: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset parser state
			parser.currentPlanets = nil
			parser.sectorPosition = SectorPosNormal
			
			// Test should not panic
			defer func() {
				if r := recover(); r != nil {
					if !tc.shouldPanic {
						t.Errorf("Test should not panic, but panicked with: %v", r)
					}
				}
			}()
			
			// Process planet line
			parser.parseSectorPlanets(tc.planetLine)
			
			// Should have at least attempted to parse something
			// The exact behavior for malformed input may vary
		})
	}
}

func TestTWXParser_PlanetDataConsistency(t *testing.T) {
	// Test data consistency validation
	db := database.NewDatabase()
	parser := NewTWXParser(db)
	
	// Test planet with both citadel and stardock
	parser.currentPlanets = nil
	parser.sectorPosition = SectorPosNormal
	
	parser.parseSectorPlanets("Planets : Fortress Citadel Stardock [Owned by Player]")
	
	if len(parser.currentPlanets) != 1 {
		t.Errorf("Expected 1 planet, got %d", len(parser.currentPlanets))
		return
	}
	
	planet := parser.currentPlanets[0]
	if planet.Name != "Fortress Citadel Stardock" {
		t.Errorf("Expected planet name 'Fortress Citadel Stardock', got '%s'", planet.Name)
	}
	
	if planet.Owner != "Player" {
		t.Errorf("Expected planet owner 'Player', got '%s'", planet.Owner)
	}
	
	if !planet.Citadel {
		t.Error("Expected citadel to be detected")
	}
	
	if !planet.Stardock {
		t.Error("Expected stardock to be detected")
	}
}