package api

// ProxyAPI defines commands from TUI to Proxy
type ProxyAPI interface {
	// Connection Management
	Disconnect() error
	IsConnected() bool
	
	// Data Processing (symmetric with OnData)  
	SendData(data []byte) error
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