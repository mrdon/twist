package api

// ProxyAPI defines commands from TUI to Proxy
type ProxyAPI interface {
	// Connection Management
	Disconnect() error
	IsConnected() bool
	
	// Data Processing (symmetric with OnData)  
	SendData(data []byte) error
	
	// Script Management (Phase 3)
	LoadScript(filename string) error
	StopAllScripts() error
	GetScriptStatus() ScriptStatusInfo
	
	// Game State Management (Phase 4)
	GetCurrentSector() (int, error)
	GetSectorInfo(sectorNum int) (SectorInfo, error)
	GetPlayerInfo() (PlayerInfo, error)
}

// TuiAPI defines notifications from Proxy to TUI
//
// CRITICAL: All methods must return immediately (within microseconds) to avoid
// blocking the proxy. Use goroutines for any actual work and queue UI updates
// through tview's QueueUpdateDraw mechanism.
type TuiAPI interface {
	// Connection Events - single callback for all status changes
	OnConnectionStatusChanged(status ConnectionStatus, address string)
	OnConnectionError(err error)

	// Data Events - must return immediately (high frequency calls)
	OnData(data []byte)
	
	// Script Events (Phase 3)
	OnScriptStatusChanged(status ScriptStatusInfo)
	OnScriptError(scriptName string, err error)
	
	// Game State Events (Phase 4.3 - MINIMAL)
	OnCurrentSectorChanged(sectorNumber int) // Simple sector change callback
}

// ConnectionStatus represents the current connection state
type ConnectionStatus int

const (
	ConnectionStatusDisconnected ConnectionStatus = iota
	ConnectionStatusConnecting
	ConnectionStatusConnected
)

func (cs ConnectionStatus) String() string {
	switch cs {
	case ConnectionStatusDisconnected:
		return "disconnected"
	case ConnectionStatusConnecting:
		return "connecting"
	case ConnectionStatusConnected:
		return "connected"
	default:
		return "unknown"
	}
}

// ScriptStatusInfo provides basic script information for Phase 3
type ScriptStatusInfo struct {
	ActiveCount int      `json:"active_count"` // Number of running scripts
	TotalCount  int      `json:"total_count"`  // Total number of loaded scripts  
	ScriptNames []string `json:"script_names"` // Names of loaded scripts
}

// PlayerInfo provides basic player information for Phase 4
type PlayerInfo struct {
	Name          string `json:"name"`           // Player name (if available)
	CurrentSector int    `json:"current_sector"` // Current sector location
}

// SectorInfo provides basic sector information for panel display
type SectorInfo struct {
	Number        int    `json:"number"`         // Sector number
	NavHaz        int    `json:"nav_haz"`        // Navigation hazard level  
	HasTraders    int    `json:"has_traders"`    // Number of traders present
	Constellation string `json:"constellation"`  // Constellation name
	Beacon        string `json:"beacon"`         // Beacon text
}

