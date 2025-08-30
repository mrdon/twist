package factory

import (
	"fmt"
	"net"
	"strings"
	"twist/internal/api"
	"twist/internal/proxy"
)

// Connect creates a new proxy instance and returns a connected ProxyAPI
func Connect(address string, tuiAPI api.TuiAPI, options ...*api.ConnectOptions) api.ProxyAPI {
	// Validate input parameters
	if tuiAPI == nil {
		panic("tuiAPI cannot be nil")
	}
	if address == "" {
		panic("address cannot be empty")
	}

	// Parse options
	var opts *api.ConnectOptions
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	} else {
		opts = &api.ConnectOptions{}
	}

	// Parse address (default to telnet port if not specified)
	if !strings.Contains(address, ":") {
		address = address + ":23"
	}

	// Establish network connection (blocking)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		// Connection failed - notify TUI and panic
		tuiAPI.OnConnectionError(fmt.Errorf("failed to connect to %s: %w", address, err))
		panic(fmt.Errorf("failed to connect to %s: %w", address, err))
	}

	// Create proxy with established connection
	proxyInstance := proxy.New(conn, address, tuiAPI, opts)
	
	// Notify TUI of successful connection
	tuiAPI.OnConnectionStatusChanged(api.ConnectionStatusConnected, address)
	
	// Return connected proxy instance
	return proxyInstance
}
