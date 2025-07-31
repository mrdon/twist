package api

// TuiAPI defines notifications from Proxy to TUI
//
// CRITICAL: All methods must return immediately (within microseconds) to avoid
// blocking the proxy. Use goroutines for any actual work and queue UI updates
// through tview's QueueUpdateDraw mechanism.
type TuiAPI interface {
	// Connection Events - all must return immediately
	OnConnecting(address string)
	OnConnected(info ConnectionInfo)
	OnDisconnected(reason string)
	OnConnectionError(err error)

	// Data Events - must return immediately (high frequency calls)
	OnData(data []byte)
}