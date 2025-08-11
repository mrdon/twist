package factory

import (
	"errors"
	"time"
	"twist/internal/api"
	"twist/internal/proxy"
)

// Connect creates a new proxy instance and returns a connected ProxyAPI
func Connect(address string, tuiAPI api.TuiAPI, options ...*api.ConnectOptions) api.ProxyAPI {
	// Never return errors - all failures go via callbacks
	// Even nil tuiAPI or empty address should be handled gracefully via callbacks
	if tuiAPI == nil {
		// This is a programming error, but handle gracefully
		return &proxy.ProxyApiImpl{} // Will fail safely when used
	}
	
	// Create ProxyAPI wrapper first
	impl := &proxy.ProxyApiImpl{}
	
	// Create proxy instance with TuiAPI directly - no adapter needed
	proxyInstance := proxy.New(tuiAPI)
	impl.SetProxy(proxyInstance)
	impl.SetTuiAPI(tuiAPI)
	
	// Notify TUI that connection is starting
	tuiAPI.OnConnectionStatusChanged(api.ConnectionStatusConnecting, address)
	
	// Attempt connection asynchronously to avoid blocking with 5-second timeout
	go func() {
		// Create a channel for the connection result
		resultChan := make(chan error, 1)
		
		// Start connection attempt in another goroutine
		go func() {
			var err error
			if len(options) > 0 {
				err = proxyInstance.Connect(address, options[0])
			} else {
				err = proxyInstance.Connect(address)
			}
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
			impl.StartMonitoring()
			
		case <-time.After(5 * time.Second):
			// Timeout -> call TuiAPI error callback
			tuiAPI.OnConnectionError(errors.New("connection timeout after 5 seconds"))
		}
	}()
	
	// Return connected ProxyAPI instance immediately
	return impl
}