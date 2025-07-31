package api

import (
	"errors"
	"time"
	"twist/internal/proxy"
	"twist/internal/debug"
)

// ProxyApiImpl implements ProxyAPI as a thin orchestration layer
type ProxyApiImpl struct {
	proxy  *proxy.Proxy  // Active proxy instance
	tuiAPI TuiAPI        // TuiAPI reference for callbacks
}

// Connect creates a new proxy instance and returns a connected ProxyAPI
func Connect(address string, tuiAPI TuiAPI) (ProxyAPI, error) {
	defer debug.LogFunction("ProxyAPI.Connect")()
	debug.Log("Connect called with address: %s", address)
	
	if address == "" {
		debug.LogError(errors.New("address cannot be empty"), "Connect validation")
		return nil, errors.New("address cannot be empty")
	}
	
	if tuiAPI == nil {
		debug.LogError(errors.New("tuiAPI cannot be nil"), "Connect validation")
		return nil, errors.New("tuiAPI cannot be nil")
	}
	
	debug.Log("Creating ProxyAPI wrapper")
	// Create ProxyAPI wrapper first
	impl := &ProxyApiImpl{
		tuiAPI: tuiAPI,
	}
	
	debug.Log("Creating proxy instance with TuiAPI wrapper as terminal writer")
	// Create proxy instance with TuiAPI wrapper as terminal writer
	// The proxy constructor expects a TerminalWriter, so we implement that interface
	proxyInstance := proxy.New(impl)
	impl.proxy = proxyInstance
	
	debug.Log("Starting async connection attempt")
	// Attempt connection asynchronously to avoid blocking
	go func() {
		defer debug.LogFunction("Connect.goroutine")()
		debug.Log("Attempting proxy connection to %s", address)
		
		err := proxyInstance.Connect(address)
		if err != nil {
			debug.LogError(err, "proxy connection failed")
			// Connection failure -> call TuiAPI error callback
			debug.Log("Calling OnConnectionError callback")
			tuiAPI.OnConnectionError(err)
			return
		}
		
		debug.Log("Proxy connection successful")
		// Success -> call TuiAPI success callback
		connectionInfo := ConnectionInfo{
			Address:     address,
			ConnectedAt: time.Now(),
			Status:      ConnectionStatusConnected,
		}
		debug.Log("Calling OnConnected callback with info: %+v", connectionInfo)
		tuiAPI.OnConnected(connectionInfo)
	}()
	
	debug.Log("Returning ProxyAPI instance immediately")
	// Return connected ProxyAPI instance immediately
	return impl, nil
}

// Write implements streaming.TerminalWriter interface to bridge to TuiAPI
// This eliminates the direct coupling between pipeline and TUI terminal
func (p *ProxyApiImpl) Write(data []byte) {
	debug.Log("ProxyApiImpl.Write called with %d bytes: %q", len(data), string(data))
	if p.tuiAPI != nil {
		debug.Log("Calling tuiAPI.OnData()")
		p.tuiAPI.OnData(data)
	} else {
		debug.Log("ERROR: tuiAPI is nil in ProxyApiImpl.Write")
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