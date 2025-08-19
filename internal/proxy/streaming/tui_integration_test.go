package streaming

import (
	"testing"
	"twist/internal/api"
	"twist/internal/proxy/database"
)

// MockPanelComponent simulates the panel component for testing
type MockPanelComponent struct {
	TraderDataUpdates   []TraderDataUpdate
	PlayerStatsUpdates  []PlayerStatsUpdate
}

type TraderDataUpdate struct {
	SectorNumber int
	Traders      []api.TraderInfo
}

type PlayerStatsUpdate struct {
	Stats api.PlayerStatsInfo
}

// Mock methods that would be called by the TUI
func (m *MockPanelComponent) UpdateTraderData(sectorNumber int, traders []api.TraderInfo) {
	m.TraderDataUpdates = append(m.TraderDataUpdates, TraderDataUpdate{
		SectorNumber: sectorNumber,
		Traders:      traders,
	})
}

func (m *MockPanelComponent) UpdatePlayerStats(stats api.PlayerStatsInfo) {
	m.PlayerStatsUpdates = append(m.PlayerStatsUpdates, PlayerStatsUpdate{
		Stats: stats,
	})
}

// MockTwistApp simulates the TwistApp for testing the complete chain
type MockTwistApp struct {
	PanelComponent *MockPanelComponent
	
	// Track handler calls
	TraderDataHandlerCalls []TraderDataHandlerCall
	PlayerStatsHandlerCalls []PlayerStatsHandlerCall
}

type TraderDataHandlerCall struct {
	SectorNumber int
	Traders      []api.TraderInfo
}

type PlayerStatsHandlerCall struct {
	Stats api.PlayerStatsInfo
}

func NewMockTwistApp() *MockTwistApp {
	return &MockTwistApp{
		PanelComponent: &MockPanelComponent{},
	}
}

// Simulate the handler methods in TwistApp
func (m *MockTwistApp) HandleTraderDataUpdated(sectorNumber int, traders []api.TraderInfo) {
	m.TraderDataHandlerCalls = append(m.TraderDataHandlerCalls, TraderDataHandlerCall{
		SectorNumber: sectorNumber,
		Traders:      traders,
	})
	
	// Simulate calling the panel component
	if m.PanelComponent != nil {
		m.PanelComponent.UpdateTraderData(sectorNumber, traders)
	}
}

func (m *MockTwistApp) HandlePlayerStatsUpdated(stats api.PlayerStatsInfo) {
	m.PlayerStatsHandlerCalls = append(m.PlayerStatsHandlerCalls, PlayerStatsHandlerCall{
		Stats: stats,
	})
	
	// Simulate calling the panel component
	if m.PanelComponent != nil {
		m.PanelComponent.UpdatePlayerStats(stats)
	}
}

func (m *MockTwistApp) HandlePortUpdated(portInfo api.PortInfo) {
	// Mock implementation for port updates
}

func (m *MockTwistApp) HandleSectorUpdated(sectorInfo api.SectorInfo) {
	// Mock implementation for sector updates
}

// MockTuiAPI that integrates with MockTwistApp
type FullMockTuiAPI struct {
	App *MockTwistApp
}

func NewFullMockTuiAPI() *FullMockTuiAPI {
	return &FullMockTuiAPI{
		App: NewMockTwistApp(),
	}
}

// Implement TuiAPI interface - just the methods we care about
func (m *FullMockTuiAPI) OnConnectionStatusChanged(status api.ConnectionStatus, address string) {}
func (m *FullMockTuiAPI) OnConnectionError(err error) {}
func (m *FullMockTuiAPI) OnData(data []byte) {}
func (m *FullMockTuiAPI) OnScriptStatusChanged(status api.ScriptStatusInfo) {}
func (m *FullMockTuiAPI) OnScriptError(scriptName string, err error) {}
func (m *FullMockTuiAPI) OnDatabaseStateChanged(info api.DatabaseStateInfo) {}
func (m *FullMockTuiAPI) OnCurrentSectorChanged(sectorInfo api.SectorInfo) {}

func (m *FullMockTuiAPI) OnTraderDataUpdated(sectorNumber int, traders []api.TraderInfo) {
	// Simulate TuiApiImpl calling the app handler
	m.App.HandleTraderDataUpdated(sectorNumber, traders)
}

func (m *FullMockTuiAPI) OnPlayerStatsUpdated(stats api.PlayerStatsInfo) {
	// Simulate TuiApiImpl calling the app handler
	m.App.HandlePlayerStatsUpdated(stats)
}

func (m *FullMockTuiAPI) OnPortUpdated(portInfo api.PortInfo) {
	// Simulate TuiApiImpl calling the app handler
	m.App.HandlePortUpdated(portInfo)
}

func (m *FullMockTuiAPI) OnSectorUpdated(sectorInfo api.SectorInfo) {
	// Simulate TuiApiImpl calling the app handler
	m.App.HandleSectorUpdated(sectorInfo)
}

func TestFullTUIIntegrationChain(t *testing.T) {
	// Test the complete chain: Parser -> TuiAPI -> TwistApp -> PanelComponent
	
	t.Run("CompleteTraderDataFlow", func(t *testing.T) {
		// Create full mock chain
		db := database.NewDatabase()
		if err := db.CreateDatabase(":memory:"); err != nil {
			t.Fatalf("Failed to create test database: %v", err)
		}
		mockTuiAPI := NewFullMockTuiAPI()
		parser := NewTWXParser(db, mockTuiAPI)

		// Process trader data like it would happen in real parsing
		traderLines := []string{
			"Traders : Captain Kirk, w/ 1,000 ftrs",
			"        in Enterprise (Constitution Class)",
		}

		// Reset parser state
		parser.currentTraders = nil
		parser.currentTrader = TraderInfo{}
		parser.sectorPosition = SectorPosNormal
		parser.currentSectorIndex = 1234

		// Process trader parsing
		parser.parseSectorTraders(traderLines[0])
		parser.handleSectorContinuation(traderLines[1])

		// Trigger sector completion (which should fire events)
		parser.sectorCompleted()

		// Verify the complete chain worked
		
		// 1. Check TwistApp handler was called
		if len(mockTuiAPI.App.TraderDataHandlerCalls) != 1 {
			t.Fatalf("Expected 1 trader data handler call, got %d", len(mockTuiAPI.App.TraderDataHandlerCalls))
		}

		handlerCall := mockTuiAPI.App.TraderDataHandlerCalls[0]
		if handlerCall.SectorNumber != 1234 {
			t.Errorf("Expected sector 1234, got %d", handlerCall.SectorNumber)
		}
		if len(handlerCall.Traders) != 1 {
			t.Fatalf("Expected 1 trader, got %d", len(handlerCall.Traders))
		}
		if handlerCall.Traders[0].Name != "Captain Kirk" {
			t.Errorf("Expected trader 'Captain Kirk', got '%s'", handlerCall.Traders[0].Name)
		}

		// 2. Check PanelComponent was updated
		if len(mockTuiAPI.App.PanelComponent.TraderDataUpdates) != 1 {
			t.Fatalf("Expected 1 panel update, got %d", len(mockTuiAPI.App.PanelComponent.TraderDataUpdates))
		}

		panelUpdate := mockTuiAPI.App.PanelComponent.TraderDataUpdates[0]
		if panelUpdate.SectorNumber != 1234 {
			t.Errorf("Expected panel sector 1234, got %d", panelUpdate.SectorNumber)
		}
		if panelUpdate.Traders[0].ShipName != "Enterprise" {
			t.Errorf("Expected ship 'Enterprise', got '%s'", panelUpdate.Traders[0].ShipName)
		}
		if panelUpdate.Traders[0].ShipType != "Constitution Class" {
			t.Errorf("Expected ship type 'Constitution Class', got '%s'", panelUpdate.Traders[0].ShipType)
		}
	})

	t.Run("CompletePlayerStatsFlow", func(t *testing.T) {
		// Create full mock chain
		db := database.NewDatabase()
		if err := db.CreateDatabase(":memory:"); err != nil {
			t.Fatalf("Failed to create test database: %v", err)
		}
		mockTuiAPI := NewFullMockTuiAPI()
		parser := NewTWXParser(db, mockTuiAPI)

		// Process player stats like it would happen in real parsing
		quickStatsLine := " Turns 150│Creds 50,000│Figs 1000│Ship 1 MerCru"
		parser.processQuickStats(quickStatsLine)

		// Verify the complete chain worked
		
		// 1. Check TwistApp handler was called
		if len(mockTuiAPI.App.PlayerStatsHandlerCalls) != 1 {
			t.Fatalf("Expected 1 player stats handler call, got %d", len(mockTuiAPI.App.PlayerStatsHandlerCalls))
		}

		handlerCall := mockTuiAPI.App.PlayerStatsHandlerCalls[0]
		if handlerCall.Stats.Turns != 150 {
			t.Errorf("Expected turns 150, got %d", handlerCall.Stats.Turns)
		}
		if handlerCall.Stats.Credits != 50000 {
			t.Errorf("Expected credits 50000, got %d", handlerCall.Stats.Credits)
		}
		if handlerCall.Stats.ShipClass != "MerCru" {
			t.Errorf("Expected ship class 'MerCru', got '%s'", handlerCall.Stats.ShipClass)
		}

		// 2. Check PanelComponent was updated
		if len(mockTuiAPI.App.PanelComponent.PlayerStatsUpdates) != 1 {
			t.Fatalf("Expected 1 panel stats update, got %d", len(mockTuiAPI.App.PanelComponent.PlayerStatsUpdates))
		}

		panelUpdate := mockTuiAPI.App.PanelComponent.PlayerStatsUpdates[0]
		if panelUpdate.Stats.Fighters != 1000 {
			t.Errorf("Expected panel fighters 1000, got %d", panelUpdate.Stats.Fighters)
		}
		if panelUpdate.Stats.ShipNumber != 1 {
			t.Errorf("Expected ship number 1, got %d", panelUpdate.Stats.ShipNumber)
		}
	})

	t.Run("MultipleTraderFlow", func(t *testing.T) {
		// Test multiple traders in the same sector
		db := database.NewDatabase()
		if err := db.CreateDatabase(":memory:"); err != nil {
			t.Fatalf("Failed to create test database: %v", err)
		}
		mockTuiAPI := NewFullMockTuiAPI()
		parser := NewTWXParser(db, mockTuiAPI)

		traderLines := []string{
			"Traders : Captain Kirk, w/ 1,000 ftrs",
			"        in Enterprise (Constitution Class)",
			"        Admiral Picard, w/ 800 ftrs",
			"        in Enterprise-D (Galaxy Class)",
		}

		// Reset parser state
		parser.currentTraders = nil
		parser.currentTrader = TraderInfo{}
		parser.sectorPosition = SectorPosNormal
		parser.currentSectorIndex = 2000

		// Process all trader lines
		parser.parseSectorTraders(traderLines[0])
		for i := 1; i < len(traderLines); i++ {
			parser.handleSectorContinuation(traderLines[i])
		}

		// Trigger sector completion
		parser.sectorCompleted()

		// Verify multiple traders were processed
		if len(mockTuiAPI.App.TraderDataHandlerCalls) != 1 {
			t.Fatalf("Expected 1 trader data handler call, got %d", len(mockTuiAPI.App.TraderDataHandlerCalls))
		}

		handlerCall := mockTuiAPI.App.TraderDataHandlerCalls[0]
		if len(handlerCall.Traders) != 2 {
			t.Fatalf("Expected 2 traders, got %d", len(handlerCall.Traders))
		}

		// Check both traders
		if handlerCall.Traders[0].Name != "Captain Kirk" {
			t.Errorf("Expected first trader 'Captain Kirk', got '%s'", handlerCall.Traders[0].Name)
		}
		if handlerCall.Traders[1].Name != "Admiral Picard" {
			t.Errorf("Expected second trader 'Admiral Picard', got '%s'", handlerCall.Traders[1].Name)
		}

		// Check panel component got both traders
		panelUpdate := mockTuiAPI.App.PanelComponent.TraderDataUpdates[0]
		if len(panelUpdate.Traders) != 2 {
			t.Fatalf("Expected 2 traders in panel update, got %d", len(panelUpdate.Traders))
		}
	})
}

func TestRealWorldParsingWithTUIEvents(t *testing.T) {
	// Test with more realistic sector data that includes multiple types of information
	
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	mockTuiAPI := NewFullMockTuiAPI()
	parser := NewTWXParser(db, mockTuiAPI)

	t.Run("ComplexSectorWithTradersAndStats", func(t *testing.T) {
		// Simulate a complex sector display followed by quick stats
		sectorLines := []string{
			"Sector  : 1234 in The Void",
			"Beacon  : None",
			"Traders : Captain Kirk, w/ 1,500 ftrs",
			"        in USS Enterprise (Imperial StarShip)",
			"        Merchant Bob, w/ 500 ftrs", 
			"        in Cargo Hauler (Merchant)",
			"Warps to Sector(s) : 1235 - 1236 - 1237",
		}

		// Process all sector lines
		for _, line := range sectorLines {
			parser.ProcessString(line + "\r")
		}

		// Trigger sector completion
		parser.sectorCompleted()

		// Then process quick stats
		parser.processQuickStats(" Sect 1234│Turns 100│Creds 25,000│Figs 750│Ship 2 Cruiser")

		// Verify both trader and player stats events were fired
		
		// Check trader events
		if len(mockTuiAPI.App.TraderDataHandlerCalls) != 1 {
			t.Fatalf("Expected 1 trader data handler call, got %d", len(mockTuiAPI.App.TraderDataHandlerCalls))
		}

		traderCall := mockTuiAPI.App.TraderDataHandlerCalls[0]
		if traderCall.SectorNumber != 1234 {
			t.Errorf("Expected sector 1234, got %d", traderCall.SectorNumber)
		}
		if len(traderCall.Traders) != 2 {
			t.Fatalf("Expected 2 traders, got %d", len(traderCall.Traders))
		}

		// Check player stats events
		if len(mockTuiAPI.App.PlayerStatsHandlerCalls) != 1 {
			t.Fatalf("Expected 1 player stats handler call, got %d", len(mockTuiAPI.App.PlayerStatsHandlerCalls))
		}

		statsCall := mockTuiAPI.App.PlayerStatsHandlerCalls[0]
		if statsCall.Stats.CurrentSector != 1234 {
			t.Errorf("Expected current sector 1234, got %d", statsCall.Stats.CurrentSector)
		}
		if statsCall.Stats.Turns != 100 {
			t.Errorf("Expected turns 100, got %d", statsCall.Stats.Turns)
		}
		if statsCall.Stats.Credits != 25000 {
			t.Errorf("Expected credits 25000, got %d", statsCall.Stats.Credits)
		}

		// Verify panel components were updated correctly
		traderUpdate := mockTuiAPI.App.PanelComponent.TraderDataUpdates[0]
		if len(traderUpdate.Traders) != 2 {
			t.Fatalf("Expected 2 traders in panel, got %d", len(traderUpdate.Traders))
		}

		if traderUpdate.Traders[0].ShipType != "Imperial StarShip" {
			t.Errorf("Expected ship type 'Imperial StarShip', got '%s'", traderUpdate.Traders[0].ShipType)
		}

		statsUpdate := mockTuiAPI.App.PanelComponent.PlayerStatsUpdates[0]
		if statsUpdate.Stats.ShipClass != "Cruiser" {
			t.Errorf("Expected ship class 'Cruiser', got '%s'", statsUpdate.Stats.ShipClass)
		}
	})

	t.Run("InventoryCommandDetection", func(t *testing.T) {
		// Test that 'I' inventory command triggers player stats updates
		
		// Clear previous events
		mockTuiAPI.App.PlayerStatsHandlerCalls = nil
		mockTuiAPI.App.PanelComponent.PlayerStatsUpdates = nil

		// Simulate 'I' command response (inventory display)
		inventoryLines := []string{
			"Command [TL=150] (1234) ? I\r",
			" \r", // Often there's a blank line
			" Turns 200│Creds 75,000│Figs 1,200│Shlds 150│Hlds 60│Ore 10│Org 15│Equ 20│Ship 3 Destroyer\r",
		}

		for _, line := range inventoryLines {
			parser.ProcessString(line)
		}

		// Verify player stats were updated from inventory
		if len(mockTuiAPI.App.PlayerStatsHandlerCalls) < 1 {
			t.Fatalf("Expected at least 1 player stats handler call, got %d", len(mockTuiAPI.App.PlayerStatsHandlerCalls))
		}

		// Get the latest stats update
		statsCall := mockTuiAPI.App.PlayerStatsHandlerCalls[len(mockTuiAPI.App.PlayerStatsHandlerCalls)-1]
		if statsCall.Stats.Turns != 200 {
			t.Errorf("Expected turns 200 from inventory, got %d", statsCall.Stats.Turns)
		}
		if statsCall.Stats.Credits != 75000 {
			t.Errorf("Expected credits 75000 from inventory, got %d", statsCall.Stats.Credits)
		}
		if statsCall.Stats.ShipClass != "Destroyer" {
			t.Errorf("Expected ship class 'Destroyer' from inventory, got '%s'", statsCall.Stats.ShipClass)
		}

		// Verify panel was updated
		if len(mockTuiAPI.App.PanelComponent.PlayerStatsUpdates) < 1 {
			t.Fatalf("Expected at least 1 panel stats update, got %d", len(mockTuiAPI.App.PanelComponent.PlayerStatsUpdates))
		}
	})
}