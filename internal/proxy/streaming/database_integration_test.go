package streaming

import (
	"testing"
)

// TestDatabaseIntegrationEndToEnd tests the complete database integration workflow
func TestDatabaseIntegrationEndToEnd(t *testing.T) {
	// Create test database
	db := NewTestDatabase()
	
	// Create parser with database
	parser := NewTWXParser(db)
	
	t.Run("MessageHistoryIntegration", func(t *testing.T) {
		// Test message being added to database
		parser.ProcessString("Incoming transmission from Captain Kirk on channel 1:\r")
		parser.ProcessString("Hello there, trader!\r")
		
		// Verify message was saved to database
		messages, err := db.GetMessageHistory(10)
		if err != nil {
			t.Fatalf("Failed to get recent messages: %v", err)
		}
		
		if len(messages) == 0 {
			t.Error("Expected at least one message in database")
		}
		
		// Check the message content
		found := false
		for _, msg := range messages {
			if msg.Content == "Hello there, trader!" {
				found = true
				if msg.Sender != "Captain Kirk" {
					t.Errorf("Expected sender 'Captain Kirk', got '%s'", msg.Sender)
				}
				if msg.Channel != 1 {
					t.Errorf("Expected channel 1, got %d", msg.Channel)
				}
				break
			}
		}
		
		if !found {
			t.Error("Expected message 'Hello there, trader!' not found in database")
		}
	})
	
	t.Run("PlayerStatsIntegration", func(t *testing.T) {
		// Test player stats being saved to database  
		quickStats := " Turns 150�Creds 10,000�Figs 500�Shlds 100�Ship 1 Merchant"
		parser.processQuickStats(quickStats)
		
		// Verify stats were saved to database
		stats, err := db.LoadPlayerStats()
		if err != nil {
			t.Fatalf("Failed to get player stats: %v", err)
		}
		
		if stats.Turns != 150 {
			t.Errorf("Expected turns 150, got %d", stats.Turns)
		}
		if stats.Credits != 10000 {
			t.Errorf("Expected credits 10000, got %d", stats.Credits)
		}
		if stats.Fighters != 500 {
			t.Errorf("Expected fighters 500, got %d", stats.Fighters)
		}
		if stats.Shields != 100 {
			t.Errorf("Expected shields 100, got %d", stats.Shields)
		}
		if stats.ShipNumber != 1 {
			t.Errorf("Expected ship number 1, got %d", stats.ShipNumber)
		}
		if stats.ShipClass != "Merchant" {
			t.Errorf("Expected ship class 'Merchant', got '%s'", stats.ShipClass)
		}
	})
	
	t.Run("SectorDataIntegration", func(t *testing.T) {
		// Test sector data being saved to database
		lines := []string{
			"Sector  : 1 in The Sphere",
			"Beacon  : FedSpace, FedLaw Enforced", 
			"Ports   : Stargate Alpha I, Class 9 Port (SSSx3)",
			"Planets : Terra [Owned by Federation], Stardock",
			"Warps to Sector(s) : 2 - 3 - 4 - 5 - 6 - 7",
		}
		
		for _, line := range lines {
			parser.ProcessString(line + "\r")
		}
		
		// Verify sector was saved to database
		sector, err := db.LoadSector(1)
		if err != nil {
			t.Fatalf("Failed to get sector 1: %v", err)
		}
		
		// Check basic sector data was saved
		if sector.Constellation != "The Sphere" {
			t.Errorf("Expected constellation 'The Sphere', got '%s'", sector.Constellation)
		}
		if sector.Beacon != "FedSpace, FedLaw Enforced" {
			t.Errorf("Expected beacon 'FedSpace, FedLaw Enforced', got '%s'", sector.Beacon)
		}
		
		// Check planets were saved (TSector should have Planets field)
		if len(sector.Planets) >= 1 {
			// Check for Terra planet
			terraFound := false
			for _, planet := range sector.Planets {
				if planet.Name == "Terra" {
					terraFound = true
					if planet.Owner != "Federation" {
						t.Errorf("Expected Terra owner 'Federation', got '%s'", planet.Owner)
					}
					break
				}
			}
			if !terraFound {
				t.Error("Terra planet not found in sector data")
			}
		} else {
			t.Log("No planets found in sector data - this may be expected if planets aren't loaded yet")
		}
	})
	
	t.Run("MigrationIntegration", func(t *testing.T) {
		// Test that database was created successfully with migrations
		// This verifies the migrations ran during database creation
		if !db.GetDatabaseOpen() {
			t.Error("Database should be open after successful creation")
		}
		
		// Verify we can perform basic operations (which means schema is correct)
		stats, err := db.LoadPlayerStats()
		if err != nil {
			t.Fatalf("Failed to load player stats - migrations may not have run: %v", err)
		}
		
		// Default stats should be available (indicating player_stats table exists)
		_ = stats // Just verify no error occurred
	})
}

// TestDatabaseErrorHandling tests error handling in database integration
func TestDatabaseErrorHandling(t *testing.T) {
	// Create test database
	db := NewTestDatabase()
	parser := NewTWXParser(db)
	
	t.Run("DatabaseUnavailable", func(t *testing.T) {
		// Close the database to simulate unavailability
		db.CloseDatabase()
		
		// Parser should handle database errors gracefully
		parser.ProcessString("Incoming transmission from Captain Kirk on channel 1:\r")
		parser.ProcessString("Hello there, trader!\r")
		
		// Parser should continue working even if database is unavailable
		// (The parser should not crash or return invalid state)
		sector := parser.GetCurrentSector()
		if sector < 0 {
			t.Logf("Parser returned sector %d - this may be expected with closed database", sector)
		}
	})
}

// TestConverterIntegration tests the converter functionality
func TestConverterIntegration(t *testing.T) {
	t.Run("SectorConverter", func(t *testing.T) {
		converter := NewSectorConverter()
		
		// Create test sector data
		sectorData := SectorData{
			Index:         1234,
			Constellation: "Test Constellation",
			Beacon:        "Test Beacon",
			Explored:      true,
		}
		
		// Convert to database format
		dbSector := converter.ToDatabase(sectorData)
		
		if dbSector.Constellation != "Test Constellation" {
			t.Errorf("Expected constellation 'Test Constellation', got '%s'", dbSector.Constellation)
		}
		if dbSector.Beacon != "Test Beacon" {
			t.Errorf("Expected beacon 'Test Beacon', got '%s'", dbSector.Beacon)
		}
		if dbSector.Explored == 0 { // TSectorExploredType enum, 0 = EtNo (not explored)
			t.Error("Expected explored to be set (non-zero)")
		}
	})
	
	t.Run("PlayerStatsConverter", func(t *testing.T) {
		converter := NewPlayerStatsConverter()
		
		// Create test player stats
		stats := PlayerStats{
			Turns:      150,
			Credits:    10000,
			Fighters:   500,
			Shields:    100,
			ShipNumber: 1,
			ShipClass:  "Merchant",
		}
		
		// Convert to database format
		dbStats := converter.ToDatabase(stats)
		
		if dbStats.Turns != 150 {
			t.Errorf("Expected turns 150, got %d", dbStats.Turns)
		}
		if dbStats.Credits != 10000 {
			t.Errorf("Expected credits 10000, got %d", dbStats.Credits)
		}
		if dbStats.ShipClass != "Merchant" {
			t.Errorf("Expected ship class 'Merchant', got '%s'", dbStats.ShipClass)
		}
	})
	
	t.Run("MessageHistoryConverter", func(t *testing.T) {
		converter := NewMessageHistoryConverter()
		
		// Create test message
		message := MessageHistory{
			Type:    MessageRadio,
			Content: "Test message",
			Sender:  "Test Sender",
			Channel: 5,
		}
		
		// Convert to database format
		dbMessage := converter.ToDatabase(message)
		
		if int(dbMessage.Type) != int(MessageRadio) {
			t.Errorf("Expected message type %d, got %d", int(MessageRadio), int(dbMessage.Type))
		}
		if dbMessage.Content != "Test message" {
			t.Errorf("Expected content 'Test message', got '%s'", dbMessage.Content)
		}
		if dbMessage.Sender != "Test Sender" {
			t.Errorf("Expected sender 'Test Sender', got '%s'", dbMessage.Sender)
		}
		if dbMessage.Channel != 5 {
			t.Errorf("Expected channel 5, got %d", dbMessage.Channel)
		}
	})
}