package proxy

import (
	"testing"
	"time"
	"twist/internal/api"
)

// MockTuiAPI for testing
type mockTuiAPI struct {
	dataReceived []string
}

func (m *mockTuiAPI) OnData(data []byte) {
	m.dataReceived = append(m.dataReceived, string(data))
}

func (m *mockTuiAPI) OnConnectionStatusChanged(status api.ConnectionStatus, address string) {}
func (m *mockTuiAPI) OnConnectionError(err error)                                           {}
func (m *mockTuiAPI) OnScriptStatusChanged(status api.ScriptStatusInfo)                     {}
func (m *mockTuiAPI) OnScriptError(scriptName string, err error)                            {}
func (m *mockTuiAPI) OnDatabaseStateChanged(info api.DatabaseStateInfo)                     {}
func (m *mockTuiAPI) OnCurrentSectorChanged(sector api.SectorInfo)                          {}
func (m *mockTuiAPI) OnTraderDataUpdated(sectorNumber int, traders []api.TraderInfo)        {}
func (m *mockTuiAPI) OnPlayerStatsUpdated(stats api.PlayerStatsInfo)                        {}
func (m *mockTuiAPI) OnPortUpdated(portInfo api.PortInfo)                                   {}
func (m *mockTuiAPI) OnSectorUpdated(sectorInfo api.SectorInfo)                             {}

func TestTerminalMenuIntegration(t *testing.T) {
	// Create mock TuiAPI
	mockAPI := &mockTuiAPI{
		dataReceived: make([]string, 0),
	}

	// Create proxy with menu manager
	proxy := New(mockAPI)

	// Verify terminal menu manager is initialized
	if proxy.terminalMenuManager == nil {
		t.Fatal("Terminal menu manager should be initialized")
	}

	// Verify menu is not initially active
	if proxy.terminalMenuManager.IsActive() {
		t.Error("Menu should not be active initially")
	}

	// Simulate menu activation with $ character
	proxy.SendInput("$")

	// Give some time for processing
	time.Sleep(10 * time.Millisecond)

	// Verify menu is now active
	if !proxy.terminalMenuManager.IsActive() {
		t.Error("Menu should be active after $ input")
	}

	// Test menu navigation
	proxy.SendInput("?")
	time.Sleep(10 * time.Millisecond)

	// Menu should still be active after help
	if !proxy.terminalMenuManager.IsActive() {
		t.Error("Menu should still be active after help command")
	}

	// Test quit menu
	proxy.SendInput("q")
	time.Sleep(10 * time.Millisecond)

	// Menu should be inactive after quit
	if proxy.terminalMenuManager.IsActive() {
		t.Error("Menu should be inactive after quit command")
	}
}

func TestMenuDataInjection(t *testing.T) {
	// Create mock TuiAPI
	mockAPI := &mockTuiAPI{
		dataReceived: make([]string, 0),
	}

	// Create proxy with menu manager
	proxy := New(mockAPI)

	// Test data injection directly
	testData := []byte("Test menu output\r\n")
	proxy.injectInboundData(testData)

	// Note: Without a real connection and pipeline, the data won't reach the TUI
	// But we can verify the method doesn't panic and handles nil pipeline gracefully

	// This test mainly ensures the injection mechanism exists and doesn't crash
	if proxy.terminalMenuManager == nil {
		t.Error("Menu manager should be available for data injection")
	}
}

func TestMenuInputRouting(t *testing.T) {
	// Create mock TuiAPI
	mockAPI := &mockTuiAPI{
		dataReceived: make([]string, 0),
	}

	// Create proxy with menu manager
	proxy := New(mockAPI)

	// Activate menu
	proxy.SendInput("$")
	time.Sleep(10 * time.Millisecond)

	if !proxy.terminalMenuManager.IsActive() {
		t.Fatal("Menu should be active for routing test")
	}

	// Test that menu input is routed correctly
	proxy.SendInput("?") // Help command
	proxy.SendInput("1") // Invalid option
	proxy.SendInput("q") // Quit

	time.Sleep(10 * time.Millisecond)

	// Menu should be inactive after quit
	if proxy.terminalMenuManager.IsActive() {
		t.Error("Menu should be inactive after quit via routing")
	}
}

func TestMenuKeyCustomization(t *testing.T) {
	// Create mock TuiAPI
	mockAPI := &mockTuiAPI{
		dataReceived: make([]string, 0),
	}

	// Create proxy with menu manager
	proxy := New(mockAPI)

	// Change menu key to #
	proxy.terminalMenuManager.SetMenuKey('#')

	// Verify new key works
	if proxy.terminalMenuManager.GetMenuKey() != '#' {
		t.Error("Menu key should be changed to #")
	}

	// Test activation with new key
	proxy.SendInput("#")
	time.Sleep(10 * time.Millisecond)

	if !proxy.terminalMenuManager.IsActive() {
		t.Error("Menu should be active with custom key #")
	}

	// Test that old key doesn't work
	proxy.SendInput("q") // Quit first
	time.Sleep(10 * time.Millisecond)

	proxy.SendInput("$") // Try old key
	time.Sleep(10 * time.Millisecond)

	if proxy.terminalMenuManager.IsActive() {
		t.Error("Menu should not be active with old key $")
	}
}
