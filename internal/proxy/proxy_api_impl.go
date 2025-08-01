package proxy

import (
	"errors"
	"time"
	// "twist/internal/debug" // Keep for future debugging
	"twist/internal/api"
)

// ProxyApiImpl implements ProxyAPI as a thin orchestration layer
type ProxyApiImpl struct {
	proxy  *Proxy  // Active proxy instance
	tuiAPI api.TuiAPI        // TuiAPI reference for callbacks
}

// Connect creates a new proxy instance and returns a connected ProxyAPI
func Connect(address string, tuiAPI api.TuiAPI) api.ProxyAPI {
	// Never return errors - all failures go via callbacks
	// Even nil tuiAPI or empty address should be handled gracefully via callbacks
	if tuiAPI == nil {
		// This is a programming error, but handle gracefully
		return &ProxyApiImpl{} // Will fail safely when used
	}
	
	// Create ProxyAPI wrapper first
	impl := &ProxyApiImpl{
		tuiAPI: tuiAPI,
	}
	
	// Create proxy instance with TuiAPI directly - no adapter needed
	proxyInstance := New(tuiAPI)
	impl.proxy = proxyInstance
	
	// Notify TUI that connection is starting
	tuiAPI.OnConnectionStatusChanged(api.ConnectionStatusConnecting, address)
	
	// Attempt connection asynchronously to avoid blocking with 5-second timeout
	go func() {
		// Create a channel for the connection result
		resultChan := make(chan error, 1)
		
		// Start connection attempt in another goroutine
		go func() {
			err := proxyInstance.Connect(address)
			resultChan <- err
		}()
		
		// Wait for either connection result or timeout
		select {
		case err := <-resultChan:
			if err != nil {
				// Connection failure -> call TuiAPI error callback
				tuiAPI.OnConnectionError(err)
				return
			}
			
			// Success -> call TuiAPI success callback
			tuiAPI.OnConnectionStatusChanged(api.ConnectionStatusConnected, address)
			
			// Start monitoring for network disconnections
			go impl.monitorConnection()
			
		case <-time.After(5 * time.Second):
			// Timeout -> call TuiAPI error callback
			tuiAPI.OnConnectionError(errors.New("connection timeout after 5 seconds"))
		}
	}()
	
	// Return connected ProxyAPI instance immediately
	return impl
}

// Thin orchestration methods - all one-liners delegating to proxy

func (p *ProxyApiImpl) Disconnect() error {
	if p.proxy == nil {
		return nil
	}
	
	go func() {
		err := p.proxy.Disconnect()
		if err != nil {
			p.tuiAPI.OnConnectionError(err)
		} else {
			p.tuiAPI.OnConnectionStatusChanged(api.ConnectionStatusDisconnected, "")
		}
	}()
	return nil
}

func (p *ProxyApiImpl) IsConnected() bool {
	if p.proxy == nil {
		return false
	}
	return p.proxy.IsConnected()
}

func (p *ProxyApiImpl) SendData(data []byte) error {
	if p.proxy == nil {
		return errors.New("not connected")
	}
	p.proxy.SendInput(string(data))
	return nil
}

// monitorConnection monitors the proxy connection and calls appropriate callbacks
func (p *ProxyApiImpl) monitorConnection() {
	// monitorConnection started
	if p.proxy == nil {
		// proxy is nil, returning
		return
	}
	
	// Use a ticker to periodically check connection status
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case _, ok := <-p.proxy.GetErrorChan():
			if !ok {
				// Error channel closed
				// Channel closed, check connection status
				if !p.proxy.IsConnected() {
					// Proxy not connected after channel close, calling OnConnectionStatusChanged
					p.tuiAPI.OnConnectionStatusChanged(api.ConnectionStatusDisconnected, "")
				}
				return
			}
			// Got error from channel
			// Check if proxy is still connected after the error
			if !p.proxy.IsConnected() {
				// Proxy not connected, calling OnConnectionStatusChanged
				// Connection was lost - call disconnection callback
				p.tuiAPI.OnConnectionStatusChanged(api.ConnectionStatusDisconnected, "")
				return
			}
			// Proxy still connected, continuing to monitor
			
		case <-ticker.C:
			// Periodically check if connection is still alive
			if !p.proxy.IsConnected() {
				// Periodic check: proxy not connected, calling OnConnectionStatusChanged
				p.tuiAPI.OnConnectionStatusChanged(api.ConnectionStatusDisconnected, "")
				return
			}
		}
	}
}