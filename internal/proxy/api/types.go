package api

import "time"

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

// ConnectionInfo represents connection state information
type ConnectionInfo struct {
	Address     string           `json:"address"`
	ConnectedAt time.Time        `json:"connected_at"`
	Status      ConnectionStatus `json:"status"`
}