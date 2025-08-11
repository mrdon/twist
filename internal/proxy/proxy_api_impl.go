package proxy

import (
	"errors"
	"time"
	"twist/internal/api"
	"twist/internal/debug"
	"twist/internal/proxy/converter"
)


// ProxyApiImpl implements ProxyAPI as a thin orchestration layer
type ProxyApiImpl struct {
	proxy  *Proxy  // Active proxy instance
	tuiAPI api.TuiAPI        // TuiAPI reference for callbacks
}

// SetProxy sets the proxy instance (for factory package)
func (p *ProxyApiImpl) SetProxy(proxy *Proxy) {
	p.proxy = proxy
}

// SetTuiAPI sets the TuiAPI instance (for factory package) 
func (p *ProxyApiImpl) SetTuiAPI(tuiAPI api.TuiAPI) {
	p.tuiAPI = tuiAPI
}

// StartMonitoring starts connection monitoring (for factory package)
func (p *ProxyApiImpl) StartMonitoring() {
	go p.monitorConnection()
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
	
	// Log raw data chunks for debugging
	debug.LogDataChunk("SendData", data)
	
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
	activeCount := 0
	totalCount := 0
	
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
	sectorInfo := convertDatabaseSectorToAPI(sectorNum, dbSector)
	
	// Phase 2: Set HasPort flag by checking if port exists in ports table
	if portData, err := p.proxy.db.LoadPort(sectorNum); err == nil && portData.ClassIndex > 0 {
		sectorInfo.HasPort = true
	}
	
	return sectorInfo, nil
}

func (p *ProxyApiImpl) GetPlayerInfo() (api.PlayerInfo, error) {
	if p.proxy == nil {
		return api.PlayerInfo{}, errors.New("not connected")
	}
	currentSector := p.proxy.GetCurrentSector()
	playerName := p.proxy.GetPlayerName()
	return convertDatabasePlayerToAPI(currentSector, playerName), nil
}

func (p *ProxyApiImpl) GetPortInfo(sectorNum int) (*api.PortInfo, error) {
	if p.proxy == nil {
		return nil, errors.New("not connected")
	}
	
	// Validate sector number range
	if sectorNum < 1 || sectorNum > 99999 {
		return nil, errors.New("invalid sector number")
	}
	
	// Phase 2: Load port data from separate ports table
	portData, err := p.proxy.db.LoadPort(sectorNum)
	if err != nil {
		return nil, err
	}
	
	// Check if sector has a port
	if portData.ClassIndex == 0 {
		return nil, nil // No port in this sector
	}
	
	// Convert port data to API format
	portInfo, err := converter.ConvertTPortToPortInfo(sectorNum, portData)
	if err != nil {
		return nil, err
	}
	
	// Return nil if no port exists in this sector (not an error)
	return portInfo, nil
}


func (p *ProxyApiImpl) GetPlayerStats() (*api.PlayerStatsInfo, error) {
	if p.proxy == nil {
		return nil, errors.New("not connected")
	}
	
	// Get database from proxy - this is the single source of truth
	database := p.proxy.GetDatabase()
	if database == nil {
		return nil, errors.New("database not available")
	}
	
	// Load player stats from database
	playerStats, err := database.LoadPlayerStats()
	if err != nil {
		return nil, err
	}
	
	// Convert TPlayerStats to API format using converter
	apiStats := converter.ConvertTPlayerStatsToPlayerStatsInfo(playerStats)
	
	return &apiStats, nil
}

