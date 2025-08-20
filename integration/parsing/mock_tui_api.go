package parsing

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"twist/internal/api"
)

// MockTuiAPI captures all API calls for validation and implements the real TuiAPI interface
type MockTuiAPI struct {
	calls []string
	t     *testing.T
}

// NewMockTuiAPI creates a new mock TUI API
func NewMockTuiAPI(t *testing.T) *MockTuiAPI {
	return &MockTuiAPI{
		calls: make([]string, 0),
		t:     t,
	}
}

// TuiAPI interface implementation - captures all calls as strings

// OnConnectionStatusChanged implements TuiAPI interface
func (m *MockTuiAPI) OnConnectionStatusChanged(status api.ConnectionStatus, address string) {
	call := fmt.Sprintf("OnConnectionStatusChanged(status=%s, address=%s)", status, address)
	m.calls = append(m.calls, call)
	if m.t != nil {
		m.t.Logf("MockTuiAPI: %s", call)
	}
}

// OnConnectionError implements TuiAPI interface
func (m *MockTuiAPI) OnConnectionError(err error) {
	call := fmt.Sprintf("OnConnectionError(err=%v)", err)
	m.calls = append(m.calls, call)
	if m.t != nil {
		m.t.Logf("MockTuiAPI: %s", call)
	}
}

// OnData implements TuiAPI interface - no-op for transcript tests
func (m *MockTuiAPI) OnData(data []byte) {
	// No-op - don't capture OnData calls to focus on other API calls
}

// OnScriptStatusChanged implements TuiAPI interface
func (m *MockTuiAPI) OnScriptStatusChanged(status api.ScriptStatusInfo) {
	call := fmt.Sprintf("OnScriptStatusChanged(active=%d, total=%d, names=%v)",
		status.ActiveCount, status.TotalCount, status.ScriptNames)
	m.calls = append(m.calls, call)
	if m.t != nil {
		m.t.Logf("MockTuiAPI: %s", call)
	}
}

// OnScriptError implements TuiAPI interface
func (m *MockTuiAPI) OnScriptError(scriptName string, err error) {
	call := fmt.Sprintf("OnScriptError(scriptName=%s, err=%v)", scriptName, err)
	m.calls = append(m.calls, call)
	if m.t != nil {
		m.t.Logf("MockTuiAPI: %s", call)
	}
}

// OnDatabaseStateChanged implements TuiAPI interface
func (m *MockTuiAPI) OnDatabaseStateChanged(info api.DatabaseStateInfo) {
	call := fmt.Sprintf("OnDatabaseStateChanged(game=%s, host=%s, port=%s, db=%s, loaded=%t)",
		info.GameName, info.ServerHost, info.ServerPort, info.DatabaseName, info.IsLoaded)
	m.calls = append(m.calls, call)
	if m.t != nil {
		m.t.Logf("MockTuiAPI: %s", call)
	}
}

// OnCurrentSectorChanged implements TuiAPI interface
func (m *MockTuiAPI) OnCurrentSectorChanged(sectorInfo api.SectorInfo) {
	// Serialize sector info as JSON for robust comparison
	jsonData, err := json.Marshal(sectorInfo)
	if err != nil {
		// Fallback to basic info if JSON fails
		call := fmt.Sprintf("OnCurrentSectorChanged(sector=%d)", sectorInfo.Number)
		m.calls = append(m.calls, call)
	} else {
		call := fmt.Sprintf("OnCurrentSectorChanged(%s)", string(jsonData))
		m.calls = append(m.calls, call)
	}

	if m.t != nil {
		m.t.Logf("MockTuiAPI: %s", m.calls[len(m.calls)-1])
	}
}

// OnPlayerStatsUpdated implements TuiAPI interface
func (m *MockTuiAPI) OnPlayerStatsUpdated(stats api.PlayerStatsInfo) {
	call := fmt.Sprintf("OnPlayerStatsUpdated(turns=%d, credits=%d, fighters=%d, shields=%d)",
		stats.Turns, stats.Credits, stats.Fighters, stats.Shields)
	m.calls = append(m.calls, call)
	if m.t != nil {
		m.t.Logf("MockTuiAPI: %s", call)
	}
}

// OnTraderDataUpdated implements TuiAPI interface
func (m *MockTuiAPI) OnTraderDataUpdated(sectorNumber int, traders []api.TraderInfo) {
	call := fmt.Sprintf("OnTraderDataUpdated(sector=%d, traders_count=%d)",
		sectorNumber, len(traders))
	m.calls = append(m.calls, call)
	if m.t != nil {
		m.t.Logf("MockTuiAPI: %s", call)
	}
}

// OnPortUpdated implements TuiAPI interface
func (m *MockTuiAPI) OnPortUpdated(portInfo api.PortInfo) {
	call := fmt.Sprintf("OnPortUpdated(sector=%d, name=%s, class=%s)",
		portInfo.SectorID, portInfo.Name, portInfo.ClassType.String())
	m.calls = append(m.calls, call)
	if m.t != nil {
		m.t.Logf("MockTuiAPI: %s", call)
	}
}

// OnSectorUpdated implements TuiAPI interface
func (m *MockTuiAPI) OnSectorUpdated(sectorInfo api.SectorInfo) {
	call := fmt.Sprintf("OnSectorUpdated(sector=%d, visited=%t)",
		sectorInfo.Number, sectorInfo.Visited)
	m.calls = append(m.calls, call)
	if m.t != nil {
		m.t.Logf("MockTuiAPI: %s", call)
	}
}

// GetCallsAsString returns all calls as a single string for easy validation
func (m *MockTuiAPI) GetCallsAsString() string {
	return strings.Join(m.calls, "\n")
}

// GetCalls returns all recorded calls
func (m *MockTuiAPI) GetCalls() []string {
	return m.calls
}

// Reset clears all recorded calls
func (m *MockTuiAPI) Reset() {
	m.calls = make([]string, 0)
}
