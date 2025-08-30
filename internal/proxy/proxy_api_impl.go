package proxy

import (
	"errors"
	"time"
	"twist/internal/api"
	"twist/internal/log"
)

// ProxyApiImpl implements ProxyAPI as a thin orchestration layer
type ProxyApiImpl struct {
	proxy  *Proxy     // Active proxy instance
	tuiAPI api.TuiAPI // TuiAPI reference for callbacks
}

// NewProxyApiImpl creates a new ProxyApiImpl with required non-nullable instances
func NewProxyApiImpl(proxy *Proxy, tuiAPI api.TuiAPI) *ProxyApiImpl {
	if proxy == nil {
		log.Error("ProxyApiImpl creation failed", "error", "proxy cannot be nil")
		panic("proxy cannot be nil")
	}
	if tuiAPI == nil {
		log.Error("ProxyApiImpl creation failed", "error", "tuiAPI cannot be nil")
		panic("tuiAPI cannot be nil")
	}

	log.Info("Creating ProxyApiImpl", "proxy", "valid", "tuiAPI", "valid")
	return &ProxyApiImpl{
		proxy:  proxy,
		tuiAPI: tuiAPI,
	}
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
	log.LogDataChunk(">>", data)

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
	return p.proxy.GetScriptStatus()
}

// Game State Management Methods - Simple direct delegation (one-liners)

func (p *ProxyApiImpl) GetCurrentSector() (int, error) {
	if p.proxy == nil {
		return 0, errors.New("not connected")
	}
	return p.proxy.GetCurrentSector() // Direct delegation
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
		// Never return empty sector data - return zero value and let caller handle error
		return api.SectorInfo{}, err
	}

	return sectorInfo, nil
}

func (p *ProxyApiImpl) GetPlayerInfo() (api.PlayerInfo, error) {
	if p.proxy == nil {
		return api.PlayerInfo{}, errors.New("not connected")
	}
	// Phase 5: Create PlayerInfo directly (no converter needed)
	currentSector, err := p.proxy.GetCurrentSector()
	if err != nil {
		return api.PlayerInfo{}, err
	}
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

// Script Menu Operations - Direct delegation to proxy script manager

func (p *ProxyApiImpl) GetScriptList() ([]api.ScriptInfo, error) {
	if p.proxy == nil {
		return nil, errors.New("not connected")
	}

	scriptManager := p.proxy.GetScriptManager()
	if scriptManager == nil {
		return []api.ScriptInfo{}, nil // Return empty list if no script manager
	}

	engine := scriptManager.GetEngine()
	if engine == nil {
		return []api.ScriptInfo{}, nil // Return empty list if no engine
	}

	// Get all scripts from the engine
	allScripts := engine.GetAllScripts()
	runningScripts := engine.GetRunningScripts()

	// Create a map of running script IDs for quick lookup
	runningMap := make(map[string]bool)
	for _, runningScript := range runningScripts {
		runningMap[runningScript.GetID()] = true
	}

	// Convert to API format
	apiScripts := make([]api.ScriptInfo, len(allScripts))
	for i, script := range allScripts {
		apiScripts[i] = api.ScriptInfo{
			ID:       script.GetID(),
			Name:     script.GetName(),
			Filename: script.GetFilename(),
			IsActive: runningMap[script.GetID()],
		}
	}

	return apiScripts, nil
}

func (p *ProxyApiImpl) SendBurstCommand(burstText string) error {
	if p.proxy == nil {
		return errors.New("not connected")
	}

	// Delegate to the proxy's burst command implementation
	// This reuses the exact same logic as the terminal menu system
	return p.proxy.SendBurstCommand(burstText)
}
