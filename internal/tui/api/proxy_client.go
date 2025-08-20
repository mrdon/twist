package api

import (
	"errors"
	coreapi "twist/internal/api"
	"twist/internal/api/factory"
)

// ProxyClient manages ProxyAPI connections for TUI
type ProxyClient struct {
	currentAPI coreapi.ProxyAPI // Current active connection (nil if disconnected)
}

// NewProxyClient creates a new proxy client
func NewProxyClient() *ProxyClient {
	return &ProxyClient{
		currentAPI: nil,
	}
}

func (pc *ProxyClient) Connect(address string, tuiAPI coreapi.TuiAPI) error {
	// Use static Connect function to create new ProxyAPI instance
	// Static function never returns errors - all failures go via callbacks
	proxyAPI := factory.Connect(address, tuiAPI)

	// Store the connected API instance
	pc.currentAPI = proxyAPI
	return nil
}

func (pc *ProxyClient) ConnectWithScript(address string, tuiAPI coreapi.TuiAPI, scriptName string) error {
	// Use Connect function with ConnectOptions to load initial script
	connectOpts := &coreapi.ConnectOptions{ScriptName: scriptName}
	proxyAPI := factory.Connect(address, tuiAPI, connectOpts)

	// Store the connected API instance
	pc.currentAPI = proxyAPI
	return nil
}

func (pc *ProxyClient) Disconnect() error {
	if pc.currentAPI == nil {
		return nil
	}

	err := pc.currentAPI.Disconnect()
	pc.currentAPI = nil // Clear reference after disconnect
	return err
}

func (pc *ProxyClient) IsConnected() bool {
	if pc.currentAPI == nil {
		return false
	}
	return pc.currentAPI.IsConnected()
}

func (pc *ProxyClient) SendData(data []byte) error {
	if pc.currentAPI == nil {
		return errors.New("not connected")
	}
	return pc.currentAPI.SendData(data)
}

func (pc *ProxyClient) GetCurrentAPI() coreapi.ProxyAPI {
	return pc.currentAPI
}
