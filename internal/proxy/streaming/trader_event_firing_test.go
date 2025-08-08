package streaming

import (
	"testing"
	"twist/internal/api"
	"twist/internal/proxy/database"
)

// MockTuiAPI implements TuiAPI interface for testing
type MockTuiAPI struct {
	// Track events that were fired
	TraderDataEvents    []TraderDataEvent
	PlayerStatsEvents   []PlayerStatsEvent
	ConnectionEvents    []ConnectionEvent
	DatabaseEvents      []DatabaseEvent
	SectorEvents        []SectorEvent
}

type TraderDataEvent struct {
	SectorNumber int
	Traders      []api.TraderInfo
}

type PlayerStatsEvent struct {
	Stats api.PlayerStatsInfo
}

type ConnectionEvent struct {
	Status  api.ConnectionStatus
	Address string
}

type DatabaseEvent struct {
	Info api.DatabaseStateInfo
}

type SectorEvent struct {
	SectorInfo api.SectorInfo
}

// Implement TuiAPI interface
func (m *MockTuiAPI) OnConnectionStatusChanged(status api.ConnectionStatus, address string) {
	m.ConnectionEvents = append(m.ConnectionEvents, ConnectionEvent{Status: status, Address: address})
}

func (m *MockTuiAPI) OnConnectionError(err error) {
	// Not used in these tests
}

func (m *MockTuiAPI) OnData(data []byte) {
	// Not used in these tests
}

func (m *MockTuiAPI) OnScriptStatusChanged(status api.ScriptStatusInfo) {
	// Not used in these tests
}

func (m *MockTuiAPI) OnScriptError(scriptName string, err error) {
	// Not used in these tests
}

func (m *MockTuiAPI) OnDatabaseStateChanged(info api.DatabaseStateInfo) {
	m.DatabaseEvents = append(m.DatabaseEvents, DatabaseEvent{Info: info})
}

func (m *MockTuiAPI) OnCurrentSectorChanged(sectorInfo api.SectorInfo) {
	m.SectorEvents = append(m.SectorEvents, SectorEvent{SectorInfo: sectorInfo})
}

func (m *MockTuiAPI) OnTraderDataUpdated(sectorNumber int, traders []api.TraderInfo) {
	m.TraderDataEvents = append(m.TraderDataEvents, TraderDataEvent{
		SectorNumber: sectorNumber,
		Traders:      traders,
	})
}

func (m *MockTuiAPI) OnPlayerStatsUpdated(stats api.PlayerStatsInfo) {
	m.PlayerStatsEvents = append(m.PlayerStatsEvents, PlayerStatsEvent{Stats: stats})
}

func TestTraderEventFiring(t *testing.T) {
	// Create test database and mock TUI API
	db := database.NewDatabase()
	mockAPI := &MockTuiAPI{}
	parser := NewTWXParser(db, mockAPI)

	t.Run("FireTraderDataEvent", func(t *testing.T) {
		// Clear previous events
		mockAPI.TraderDataEvents = nil

		// Create test trader data
		traders := []TraderInfo{
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
		}

		sectorNumber := 1234

		// Fire the trader data event
		parser.fireTraderDataEvent(sectorNumber, traders)

		// Verify event was fired
		if len(mockAPI.TraderDataEvents) != 1 {
			t.Fatalf("Expected 1 trader data event, got %d", len(mockAPI.TraderDataEvents))
		}

		event := mockAPI.TraderDataEvents[0]

		// Check sector number
		if event.SectorNumber != sectorNumber {
			t.Errorf("Expected sector number %d, got %d", sectorNumber, event.SectorNumber)
		}

		// Check trader count
		if len(event.Traders) != 2 {
			t.Fatalf("Expected 2 traders, got %d", len(event.Traders))
		}

		// Check first trader
		trader1 := event.Traders[0]
		if trader1.Name != "Captain Kirk" {
			t.Errorf("Expected first trader name 'Captain Kirk', got '%s'", trader1.Name)
		}
		if trader1.ShipName != "Enterprise" {
			t.Errorf("Expected first trader ship 'Enterprise', got '%s'", trader1.ShipName)
		}
		if trader1.Fighters != 1000 {
			t.Errorf("Expected first trader fighters 1000, got %d", trader1.Fighters)
		}

		// Check second trader
		trader2 := event.Traders[1]
		if trader2.Name != "Admiral Picard" {
			t.Errorf("Expected second trader name 'Admiral Picard', got '%s'", trader2.Name)
		}
		if trader2.ShipType != "Galaxy Class" {
			t.Errorf("Expected second trader ship type 'Galaxy Class', got '%s'", trader2.ShipType)
		}
		if trader2.Alignment != "Good" {
			t.Errorf("Expected second trader alignment 'Good', got '%s'", trader2.Alignment)
		}
	})

	t.Run("FireTraderDataEventEmptyList", func(t *testing.T) {
		// Clear previous events
		mockAPI.TraderDataEvents = nil

		// Fire event with empty trader list
		parser.fireTraderDataEvent(5678, []TraderInfo{})

		// Verify event was fired
		if len(mockAPI.TraderDataEvents) != 1 {
			t.Fatalf("Expected 1 trader data event, got %d", len(mockAPI.TraderDataEvents))
		}

		event := mockAPI.TraderDataEvents[0]
		if event.SectorNumber != 5678 {
			t.Errorf("Expected sector number 5678, got %d", event.SectorNumber)
		}
		if len(event.Traders) != 0 {
			t.Errorf("Expected 0 traders, got %d", len(event.Traders))
		}
	})

	t.Run("NoEventWhenTuiAPIIsNil", func(t *testing.T) {
		// Create parser without TUI API
		parserNoAPI := NewTWXParser(db, nil)

		// This should not crash
		traders := []TraderInfo{
			{Name: "Test Trader", Fighters: 100},
		}
		parserNoAPI.fireTraderDataEvent(1000, traders)

		// No verification needed - just ensure no crash
	})
}

func TestPlayerStatsEventFiring(t *testing.T) {
	// Create test database and mock TUI API
	db := database.NewDatabase()
	mockAPI := &MockTuiAPI{}
	parser := NewTWXParser(db, mockAPI)

	t.Run("FirePlayerStatsEvent", func(t *testing.T) {
		// Clear previous events
		mockAPI.PlayerStatsEvents = nil

		// Create test player stats
		stats := PlayerStats{
			Turns:         150,
			Credits:       50000,
			Fighters:      1000,
			Shields:       200,
			TotalHolds:    100,
			OreHolds:      25,
			OrgHolds:      30,
			EquHolds:      20,
			ColHolds:      15,
			ShipNumber:    1,
			ShipClass:     "Imperial StarShip",
			CurrentSector: 1234,
			PlayerName:    "Captain Kirk",
			PsychicProbe:  true,
			PlanetScanner: false,
			Alignment:     500,
		}

		// Fire the player stats event
		parser.firePlayerStatsEvent(stats)

		// Verify event was fired
		if len(mockAPI.PlayerStatsEvents) != 1 {
			t.Fatalf("Expected 1 player stats event, got %d", len(mockAPI.PlayerStatsEvents))
		}

		event := mockAPI.PlayerStatsEvents[0]

		// Check key fields
		if event.Stats.Turns != 150 {
			t.Errorf("Expected turns 150, got %d", event.Stats.Turns)
		}
		if event.Stats.Credits != 50000 {
			t.Errorf("Expected credits 50000, got %d", event.Stats.Credits)
		}
		if event.Stats.Fighters != 1000 {
			t.Errorf("Expected fighters 1000, got %d", event.Stats.Fighters)
		}
		if event.Stats.PlayerName != "Captain Kirk" {
			t.Errorf("Expected player name 'Captain Kirk', got '%s'", event.Stats.PlayerName)
		}
		if event.Stats.ShipClass != "Imperial StarShip" {
			t.Errorf("Expected ship class 'Imperial StarShip', got '%s'", event.Stats.ShipClass)
		}
		if event.Stats.CurrentSector != 1234 {
			t.Errorf("Expected current sector 1234, got %d", event.Stats.CurrentSector)
		}
		if event.Stats.PsychicProbe != true {
			t.Errorf("Expected psychic probe true, got %t", event.Stats.PsychicProbe)
		}
		if event.Stats.PlanetScanner != false {
			t.Errorf("Expected planet scanner false, got %t", event.Stats.PlanetScanner)
		}

		// Check cargo holds conversion
		if event.Stats.TotalHolds != 100 {
			t.Errorf("Expected total holds 100, got %d", event.Stats.TotalHolds)
		}
		if event.Stats.OreHolds != 25 {
			t.Errorf("Expected ore holds 25, got %d", event.Stats.OreHolds)
		}
		if event.Stats.OrgHolds != 30 {
			t.Errorf("Expected org holds 30, got %d", event.Stats.OrgHolds)
		}
	})

	t.Run("FirePlayerStatsEventMinimal", func(t *testing.T) {
		// Clear previous events
		mockAPI.PlayerStatsEvents = nil

		// Create minimal player stats
		stats := PlayerStats{
			Turns:   0,
			Credits: 0,
		}

		// Fire the player stats event
		parser.firePlayerStatsEvent(stats)

		// Verify event was fired
		if len(mockAPI.PlayerStatsEvents) != 1 {
			t.Fatalf("Expected 1 player stats event, got %d", len(mockAPI.PlayerStatsEvents))
		}

		event := mockAPI.PlayerStatsEvents[0]
		if event.Stats.Turns != 0 {
			t.Errorf("Expected turns 0, got %d", event.Stats.Turns)
		}
		if event.Stats.Credits != 0 {
			t.Errorf("Expected credits 0, got %d", event.Stats.Credits)
		}
	})

	t.Run("NoEventWhenTuiAPIIsNil", func(t *testing.T) {
		// Create parser without TUI API
		parserNoAPI := NewTWXParser(db, nil)

		// This should not crash
		stats := PlayerStats{
			PlayerName: "Test Player",
			Credits:    1000,
		}
		parserNoAPI.firePlayerStatsEvent(stats)

		// No verification needed - just ensure no crash
	})
}

func TestEventIntegrationWithParsing(t *testing.T) {
	// Test that events are fired during actual parsing scenarios
	db := database.NewDatabase()
	mockAPI := &MockTuiAPI{}
	parser := NewTWXParser(db, mockAPI)

	t.Run("TraderEventDuringSectorComplete", func(t *testing.T) {
		// Clear events
		mockAPI.TraderDataEvents = nil

		// Set up parser state with traders
		parser.currentSectorIndex = 1234
		parser.currentTraders = []TraderInfo{
			{
				Name:      "Captain Kirk",
				ShipName:  "Enterprise",
				Fighters:  1000,
				Alignment: "Good",
			},
		}
		parser.sectorSaved = false

		// Trigger sector completion (which should fire trader event)
		parser.sectorCompleted()

		// Verify trader event was fired
		if len(mockAPI.TraderDataEvents) != 1 {
			t.Fatalf("Expected 1 trader data event, got %d", len(mockAPI.TraderDataEvents))
		}

		event := mockAPI.TraderDataEvents[0]
		if event.SectorNumber != 1234 {
			t.Errorf("Expected sector 1234, got %d", event.SectorNumber)
		}
		if len(event.Traders) != 1 {
			t.Fatalf("Expected 1 trader, got %d", len(event.Traders))
		}
		if event.Traders[0].Name != "Captain Kirk" {
			t.Errorf("Expected trader 'Captain Kirk', got '%s'", event.Traders[0].Name)
		}
	})

	t.Run("PlayerStatsEventDuringQuickStats", func(t *testing.T) {
		// Clear events
		mockAPI.PlayerStatsEvents = nil

		// Process quick stats line
		quickStatsLine := " Turns 150│Creds 50,000│Figs 1000│Ship 1 MerCru"
		parser.processQuickStats(quickStatsLine)

		// Verify player stats event was fired
		if len(mockAPI.PlayerStatsEvents) != 1 {
			t.Fatalf("Expected 1 player stats event, got %d", len(mockAPI.PlayerStatsEvents))
		}

		event := mockAPI.PlayerStatsEvents[0]
		if event.Stats.Turns != 150 {
			t.Errorf("Expected turns 150, got %d", event.Stats.Turns)
		}
		if event.Stats.Credits != 50000 {
			t.Errorf("Expected credits 50000, got %d", event.Stats.Credits)
		}
		if event.Stats.Fighters != 1000 {
			t.Errorf("Expected fighters 1000, got %d", event.Stats.Fighters)
		}
		if event.Stats.ShipClass != "MerCru" {
			t.Errorf("Expected ship class 'MerCru', got '%s'", event.Stats.ShipClass)
		}
	})

	t.Run("NoTraderEventWhenNoTraders", func(t *testing.T) {
		// Clear events
		mockAPI.TraderDataEvents = nil

		// Set up parser state without traders
		parser.currentSectorIndex = 5678
		parser.currentTraders = []TraderInfo{} // Empty
		parser.sectorSaved = false

		// Trigger sector completion
		parser.sectorCompleted()

		// No trader event should be fired for empty trader list
		if len(mockAPI.TraderDataEvents) != 0 {
			t.Errorf("Expected 0 trader data events, got %d", len(mockAPI.TraderDataEvents))
		}
	})
}

func TestEventDataIntegrity(t *testing.T) {
	// Test that event data maintains integrity during conversion
	db := database.NewDatabase()
	mockAPI := &MockTuiAPI{}
	parser := NewTWXParser(db, mockAPI)

	t.Run("UnicodeDataIntegrity", func(t *testing.T) {
		// Clear events
		mockAPI.TraderDataEvents = nil

		// Create traders with Unicode names
		traders := []TraderInfo{
			{
				Name:      "Капитан Кирк",
				ShipName:  "Энтерпрайз",
				ShipType:  "Конституция",
				Alignment: "Добрый",
				Fighters:  500,
			},
		}

		parser.fireTraderDataEvent(1000, traders)

		// Verify Unicode data was preserved
		if len(mockAPI.TraderDataEvents) != 1 {
			t.Fatalf("Expected 1 trader data event, got %d", len(mockAPI.TraderDataEvents))
		}

		event := mockAPI.TraderDataEvents[0]
		trader := event.Traders[0]

		if trader.Name != "Капитан Кирк" {
			t.Errorf("Unicode trader name not preserved: expected 'Капитан Кирк', got '%s'", trader.Name)
		}
		if trader.ShipName != "Энтерпрайз" {
			t.Errorf("Unicode ship name not preserved: expected 'Энтерпрайз', got '%s'", trader.ShipName)
		}
		if trader.Alignment != "Добрый" {
			t.Errorf("Unicode alignment not preserved: expected 'Добрый', got '%s'", trader.Alignment)
		}
	})

	t.Run("SpecialCharacterIntegrity", func(t *testing.T) {
		// Clear events
		mockAPI.PlayerStatsEvents = nil

		// Create player stats with special characters
		stats := PlayerStats{
			PlayerName: "Captain O'Brien [Test]",
			ShipClass:  "Ship-Type (Special)",
		}

		parser.firePlayerStatsEvent(stats)

		// Verify special characters were preserved
		if len(mockAPI.PlayerStatsEvents) != 1 {
			t.Fatalf("Expected 1 player stats event, got %d", len(mockAPI.PlayerStatsEvents))
		}

		event := mockAPI.PlayerStatsEvents[0]
		if event.Stats.PlayerName != "Captain O'Brien [Test]" {
			t.Errorf("Special characters in player name not preserved: expected 'Captain O'Brien [Test]', got '%s'", event.Stats.PlayerName)
		}
		if event.Stats.ShipClass != "Ship-Type (Special)" {
			t.Errorf("Special characters in ship class not preserved: expected 'Ship-Type (Special)', got '%s'", event.Stats.ShipClass)
		}
	})

	t.Run("NumericalRangeIntegrity", func(t *testing.T) {
		// Test that numerical values maintain their full range
		mockAPI.TraderDataEvents = nil
		mockAPI.PlayerStatsEvents = nil

		// Test large fighter count
		traders := []TraderInfo{
			{Name: "High Fighter Trader", Fighters: 999999},
		}
		parser.fireTraderDataEvent(1, traders)

		// Test large credits and experience
		stats := PlayerStats{
			Credits:    2147483647, // Max int32
			Experience: 999999999,
			Alignment:  -1000, // Negative alignment
		}
		parser.firePlayerStatsEvent(stats)

		// Verify large numbers were preserved
		traderEvent := mockAPI.TraderDataEvents[0]
		if traderEvent.Traders[0].Fighters != 999999 {
			t.Errorf("Large fighter count not preserved: expected 999999, got %d", traderEvent.Traders[0].Fighters)
		}

		statsEvent := mockAPI.PlayerStatsEvents[0]
		if statsEvent.Stats.Credits != 2147483647 {
			t.Errorf("Large credits not preserved: expected 2147483647, got %d", statsEvent.Stats.Credits)
		}
		if statsEvent.Stats.Alignment != -1000 {
			t.Errorf("Negative alignment not preserved: expected -1000, got %d", statsEvent.Stats.Alignment)
		}
	})
}