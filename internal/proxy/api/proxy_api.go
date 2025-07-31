package api

// ProxyAPI defines commands from TUI to Proxy
type ProxyAPI interface {
	// Connection Management
	Connect(address string, tuiAPI TuiAPI) error
	Disconnect() error
	IsConnected() bool

	// Data Processing
	SendData(data []byte) error

	// Lifecycle
	Shutdown() error
}