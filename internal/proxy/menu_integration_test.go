package proxy

import (
	"net"
	"testing"
	"time"
	"twist/internal/api"
)

// MockTuiAPI for testing
type mockTuiAPI struct {
	dataReceived []string
}

func (m *mockTuiAPI) OnData(data []byte)                                         { m.dataReceived = append(m.dataReceived, string(data)) }
func (m *mockTuiAPI) OnConnectionStatusChanged(status api.ConnectionStatus, address string) {}
func (m *mockTuiAPI) OnConnectionError(err error)                                {}
func (m *mockTuiAPI) OnScriptStatusChanged(status api.ScriptStatusInfo)         {}
func (m *mockTuiAPI) OnScriptError(scriptName string, err error)                {}
func (m *mockTuiAPI) OnDatabaseStateChanged(info api.DatabaseStateInfo)         {}
func (m *mockTuiAPI) OnCurrentSectorChanged(sector api.SectorInfo)              {}
func (m *mockTuiAPI) OnTraderDataUpdated(sectorNumber int, traders []api.TraderInfo) {}
func (m *mockTuiAPI) OnPlayerStatsUpdated(stats api.PlayerStatsInfo)            {}
func (m *mockTuiAPI) OnPortUpdated(portInfo api.PortInfo)                       {}
func (m *mockTuiAPI) OnSectorUpdated(sectorInfo api.SectorInfo)                 {}

func TestTerminalMenuIntegration(t *testing.T) {
	t.Skip("Terminal menu test - needs telnet mocking for fast execution")
	
	mockAPI := &mockTuiAPI{dataReceived: make([]string, 0)}
	
	// Create mock connection with telnet echo
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	
	// Handle telnet negotiation to prevent blocking
	go func() {
		buffer := make([]byte, 1024)
		for {
			n, err := server.Read(buffer)
			if err != nil {
				return
			}
			server.Write(buffer[:n]) // Echo back
		}
	}()
	
	// Create proxy (this will do telnet negotiation)
	proxy := New(client, "test:23", mockAPI, &api.ConnectOptions{})
	
	// Wait a bit for initialization
	time.Sleep(50 * time.Millisecond)
	
	// Verify terminal menu manager is initialized
	if proxy.terminalMenuManager == nil {
		t.Fatal("Terminal menu manager should be initialized")
	}
	
	// Test menu activation with $ character
	proxy.SendInput("$")
	time.Sleep(10 * time.Millisecond)
	
	// Basic test - just verify menu manager exists and doesn't crash
	if proxy.terminalMenuManager.GetMenuKey() != '$' {
		t.Error("Default menu key should be $")
	}
}