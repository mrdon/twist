package proxy

import (
	"errors"
	"time"
	"twist/internal/api"
	"twist/internal/debug"
)

// ProxyApiImpl implements ProxyAPI as a thin orchestration layer
type ProxyApiImpl struct {
	proxy  *Proxy     // Active proxy instance
	tuiAPI api.TuiAPI // TuiAPI reference for callbacks
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
	debug.LogDataChunk(">>", data)

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
				// Connection was lost - clean up and notify
				p.proxy.Disconnect() // Ensure proper cleanup including game state
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

	// Phase 5: Use direct database API method (no converter needed)
	sectorInfo, err := p.proxy.db.GetSectorInfo(sectorNum)
	if err != nil {
		// Return empty sector info with error
		return api.SectorInfo{
			Number:        sectorNum,
			NavHaz:        0,
			HasTraders:    0,
			Constellation: "",
			Beacon:        "",
		}, err
	}

	return sectorInfo, nil
}

func (p *ProxyApiImpl) GetPlayerInfo() (api.PlayerInfo, error) {
	if p.proxy == nil {
		return api.PlayerInfo{}, errors.New("not connected")
	}
	// Phase 5: Create PlayerInfo directly (no converter needed)
	currentSector := p.proxy.GetCurrentSector()
	playerName := p.proxy.GetPlayerName()

	return api.PlayerInfo{
		Name:          playerName,
		CurrentSector: currentSector,
	}, nil
}

func (p *ProxyApiImpl) GetPortInfo(sectorNum int) (*api.PortInfo, error) {
	if p.proxy == nil {
		return nil, errors.New("not connected")
	}

	// Validate sector number range
	if sectorNum < 1 || sectorNum > 99999 {
		return nil, errors.New("invalid sector number")
	}

	// Phase 5: Use direct database API method (no converter needed)
	portInfo, err := p.proxy.db.GetPortInfo(sectorNum)
	if err != nil {
		return nil, err
	}

	// Return port info (nil if no port exists)
	return portInfo, nil
}

func (p *ProxyApiImpl) GetPlayerStats() (*api.PlayerStatsInfo, error) {
	if p.proxy == nil {
		return nil, errors.New("not connected")
	}

	// Phase 5: Use direct database API method (no converter needed)
	database := p.proxy.GetDatabase()
	if database == nil {
		return nil, errors.New("database not available")
	}

	// Get player stats directly in API format
	apiStats, err := database.GetPlayerStatsInfo()
	if err != nil {
		return nil, err
	}

	return &apiStats, nil
}
