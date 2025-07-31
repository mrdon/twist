package api

import (
	"errors"
	proxyapi "twist/internal/proxy/api"
	"twist/internal/debug"
)

// ProxyClient manages ProxyAPI connections for TUI
type ProxyClient struct {
	currentAPI proxyapi.ProxyAPI // Current active connection (nil if disconnected)
}

// NewProxyClient creates a new proxy client
func NewProxyClient() *ProxyClient {
	return &ProxyClient{
		currentAPI: nil,
	}
}

func (pc *ProxyClient) Connect(address string, tuiAPI proxyapi.TuiAPI) error {
	defer debug.LogFunction("ProxyClient.Connect")()
	debug.Log("ProxyClient.Connect called with address: %s", address)
	
	// Use static Connect function to create new ProxyAPI instance
	api, err := proxyapi.Connect(address, tuiAPI)
	if err != nil {
		debug.LogError(err, "proxyapi.Connect failed")
		return err
	}

	debug.Log("ProxyAPI instance created successfully")
	// Store the connected API instance
	pc.currentAPI = api
	debug.Log("ProxyClient.Connect completed successfully")
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

func (pc *ProxyClient) Shutdown() error {
	if pc.currentAPI == nil {
		return nil
	}

	err := pc.currentAPI.Shutdown()
	pc.currentAPI = nil // Clear reference after shutdown
	return err
}