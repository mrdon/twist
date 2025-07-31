package api

import (
	"errors"
	"fmt"
	"time"
	"twist/internal/proxy"
)

// ProxyApiImpl implements ProxyAPI as a thin orchestration layer
type ProxyApiImpl struct {
	proxy  *proxy.Proxy  // Active proxy instance
	tuiAPI TuiAPI        // TuiAPI reference for callbacks
}

// Connect creates a new proxy instance and returns a connected ProxyAPI
func Connect(address string, tuiAPI TuiAPI) (ProxyAPI, error) {
	if address == "" {
		return nil, errors.New("address cannot be empty")
	}
	
	if tuiAPI == nil {
		return nil, errors.New("tuiAPI cannot be nil")
	}
	
	// Create ProxyAPI wrapper first
	impl := &ProxyApiImpl{
		tuiAPI: tuiAPI,
	}
	
	// Create proxy instance with TuiAPI wrapper as terminal writer
	// The proxy constructor expects a TerminalWriter, so we implement that interface
	proxyInstance := proxy.New(impl)
	impl.proxy = proxyInstance
	
	// Attempt connection asynchronously to avoid blocking
	go func() {
		err := proxyInstance.Connect(address)
		if err != nil {
			// Connection failure -> call TuiAPI error callback
			tuiAPI.OnConnectionError(err)
			return
		}
		
		// Success -> call TuiAPI success callback
		connectionInfo := ConnectionInfo{
			Address:     address,
			ConnectedAt: time.Now(),
			Status:      ConnectionStatusConnected,
		}
		tuiAPI.OnConnected(connectionInfo)
		
		// Start monitoring for network disconnections
		go impl.monitorConnection()
	}()
	
	// Return connected ProxyAPI instance immediately
	return impl, nil
}

// Write implements streaming.TerminalWriter interface to bridge to TuiAPI
// This eliminates the direct coupling between pipeline and TUI terminal
func (p *ProxyApiImpl) Write(data []byte) {
	if p.tuiAPI != nil {
		p.tuiAPI.OnData(data)
	}
}

// Thin orchestration methods - all one-liners delegating to proxy
func (p *ProxyApiImpl) Connect(address string, tuiAPI TuiAPI) error {
	// Not used - Connect is now a static function
	return errors.New("use api.Connect() function instead")
}

func (p *ProxyApiImpl) Disconnect() error {
	if p.proxy == nil {
		return nil
	}
	
	go func() {
		err := p.proxy.Disconnect()
		if err != nil {
			p.tuiAPI.OnConnectionError(err)
		} else {
			p.tuiAPI.OnDisconnected("user requested")
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

func (p *ProxyApiImpl) Shutdown() error {
	if p.proxy == nil {
		return nil
	}
	
	go func() {
		err := p.proxy.Disconnect()
		if err != nil {
			p.tuiAPI.OnConnectionError(err)
		} else {
			p.tuiAPI.OnDisconnected("shutdown")
		}
	}()
	return nil
}

// monitorConnection monitors the proxy connection and calls appropriate callbacks
func (p *ProxyApiImpl) monitorConnection() {
	fmt.Printf("[DEBUG] monitorConnection started\n")
	if p.proxy == nil {
		fmt.Printf("[DEBUG] proxy is nil, returning\n")
		return
	}
	
	// Use a ticker to periodically check connection status
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case err, ok := <-p.proxy.GetErrorChan():
			if !ok {
				fmt.Printf("[DEBUG] Error channel closed\n")
				// Channel closed, check connection status
				if !p.proxy.IsConnected() {
					fmt.Printf("[DEBUG] Proxy not connected after channel close, calling OnDisconnected\n")
					p.tuiAPI.OnDisconnected("connection closed")
				}
				return
			}
			fmt.Printf("[DEBUG] Got error from channel: %v\n", err)
			// Check if proxy is still connected after the error
			if !p.proxy.IsConnected() {
				fmt.Printf("[DEBUG] Proxy not connected, calling OnDisconnected\n")
				// Connection was lost - call disconnection callback
				p.tuiAPI.OnDisconnected("connection lost: " + err.Error())
				return
			}
			fmt.Printf("[DEBUG] Proxy still connected, continuing to monitor\n")
			
		case <-ticker.C:
			// Periodically check if connection is still alive
			if !p.proxy.IsConnected() {
				fmt.Printf("[DEBUG] Periodic check: proxy not connected, calling OnDisconnected\n")
				p.tuiAPI.OnDisconnected("connection lost")
				return
			}
		}
	}
}