package proxy

import (
	"errors"
	"time"
	// "twist/internal/debug" // Keep for future debugging
	"twist/internal/api"
)

func init() {
	// Register the Connect implementation with the API package
	api.SetConnectImpl(connect)
}

// ProxyApiImpl implements ProxyAPI as a thin orchestration layer
type ProxyApiImpl struct {
	proxy  *Proxy  // Active proxy instance
	tuiAPI api.TuiAPI        // TuiAPI reference for callbacks
}

// connect creates a new proxy instance and returns a connected ProxyAPI
// This is the internal implementation registered with the API package
func connect(address string, tuiAPI api.TuiAPI) api.ProxyAPI {
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

// Script Management Methods - Thin orchestration layer (one-liners)

func (p *ProxyApiImpl) LoadScript(filename string) error {
	if p.proxy == nil {
		return errors.New("not connected")
	}
	
	// Do work asynchronously - return immediately
	go func() {
		err := p.proxy.LoadScript(filename)
		if err != nil {
			p.tuiAPI.OnScriptError(filename, err)
		} else {
			// Report status change
			status := p.convertScriptStatus()
			p.tuiAPI.OnScriptStatusChanged(status)
		}
	}()
	
	return nil // Returns immediately
}

func (p *ProxyApiImpl) StopAllScripts() error {
	if p.proxy == nil {
		return errors.New("not connected")
	}
	
	// Do work asynchronously - return immediately
	go func() {
		err := p.proxy.StopAllScripts()
		if err != nil {
			p.tuiAPI.OnScriptError("all scripts", err)
		} else {
			// Report status change
			status := p.convertScriptStatus()
			p.tuiAPI.OnScriptStatusChanged(status)
		}
	}()
	
	return nil // Returns immediately
}

func (p *ProxyApiImpl) GetScriptStatus() api.ScriptStatusInfo {
	if p.proxy == nil {
		return api.ScriptStatusInfo{
			ActiveCount: 0,
			TotalCount:  0,
			ScriptNames: []string{},
		}
	}
	
	return p.convertScriptStatus()
}

// convertScriptStatus converts proxy script manager status to API format
func (p *ProxyApiImpl) convertScriptStatus() api.ScriptStatusInfo {
	statusMap := p.proxy.GetScriptStatus()
	
	// Extract counts from existing GetStatus() return value
	totalCount := 0
	activeCount := 0
	
	if total, ok := statusMap["total_scripts"].(int); ok {
		totalCount = total
	}
	if running, ok := statusMap["running_scripts"].(int); ok {
		activeCount = running
	}
	
	// Get script names - will need to be extended when proxy supports this
	scriptNames := []string{}
	if names, ok := statusMap["script_names"].([]string); ok {
		scriptNames = names
	}
	
	return api.ScriptStatusInfo{
		ActiveCount: activeCount,
		TotalCount:  totalCount,
		ScriptNames: scriptNames,
	}
}

// Game State Management Methods - Simple direct delegation (one-liners)

func (p *ProxyApiImpl) GetCurrentSector() (int, error) {
	if p.proxy == nil {
		return 0, errors.New("not connected")
	}
	return p.proxy.GetCurrentSector(), nil // Simple delegation
}

func (p *ProxyApiImpl) GetSectorInfo(sectorNum int) (api.SectorInfo, error) {
	if p.proxy == nil {
		return api.SectorInfo{Number: sectorNum}, errors.New("not connected")
	}
	
	// Validate sector number range
	if sectorNum < 1 || sectorNum > 99999 {
		return api.SectorInfo{Number: sectorNum}, errors.New("invalid sector number")
	}
	
	// Direct delegation using new GetSector wrapper method
	dbSector, err := p.proxy.GetSector(sectorNum)
	if err != nil {
		// Return empty sector info with error rather than potentially corrupted data
		return api.SectorInfo{
			Number:        sectorNum,
			NavHaz:        0,
			HasTraders:    0,
			Constellation: "",
			Beacon:        "",
		}, err
	}
	
	// Simple conversion using converter function
	return convertDatabaseSectorToAPI(sectorNum, dbSector), nil
}

func (p *ProxyApiImpl) GetPlayerInfo() (api.PlayerInfo, error) {
	if p.proxy == nil {
		return api.PlayerInfo{}, errors.New("not connected")
	}
	currentSector := p.proxy.GetCurrentSector()
	playerName := p.proxy.GetPlayerName()
	return convertDatabasePlayerToAPI(currentSector, playerName), nil
}

