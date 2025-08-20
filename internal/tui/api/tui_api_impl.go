package api

import (
	coreapi "twist/internal/api"
	"twist/internal/debug"
)

// Forward declaration - will be defined when we update app.go
type TwistApp interface {
	HandleConnectionStatusChanged(status coreapi.ConnectionStatus, address string)
	HandleConnectionError(err error)
	HandleTerminalData(data []byte)
	HandleScriptStatusChanged(status coreapi.ScriptStatusInfo)
	HandleScriptError(scriptName string, err error)
	HandleDatabaseStateChanged(info coreapi.DatabaseStateInfo)
	HandleCurrentSectorChanged(sectorInfo coreapi.SectorInfo)
	HandlePortUpdated(portInfo coreapi.PortInfo)
	HandleTraderDataUpdated(sectorNumber int, traders []coreapi.TraderInfo)
	HandlePlayerStatsUpdated(stats coreapi.PlayerStatsInfo)
	HandleSectorUpdated(sectorInfo coreapi.SectorInfo)
}

// TuiApiImpl implements TuiAPI as a thin orchestration layer
type TuiApiImpl struct {
	app        TwistApp
	dataChan   chan []byte
	shutdownCh chan struct{}
}

// NewTuiAPI creates a new TuiAPI implementation
func NewTuiAPI(app TwistApp) coreapi.TuiAPI {
	impl := &TuiApiImpl{
		app:        app,
		dataChan:   make(chan []byte, 100), // Buffered channel for data
		shutdownCh: make(chan struct{}),
	}

	// Start single processing goroutine
	go impl.processDataLoop()

	return impl
}

// Thin orchestration methods - all one-liners calling app directly
// All methods MUST return immediately using goroutines for async work
func (tui *TuiApiImpl) OnConnectionStatusChanged(status coreapi.ConnectionStatus, address string) {
	go tui.app.HandleConnectionStatusChanged(status, address)
}

func (tui *TuiApiImpl) OnConnectionError(err error) {
	go tui.app.HandleConnectionError(err)
}

func (tui *TuiApiImpl) OnData(data []byte) {
	// Log raw data chunks for debugging
	debug.LogDataChunk("<", data)

	// Copy data and send to processing channel
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	// Non-blocking send to avoid blocking network thread
	select {
	case tui.dataChan <- dataCopy:
	default:
		// Channel full - could log warning or handle differently
	}
}

// Script event methods - all one-liners calling app directly
func (tui *TuiApiImpl) OnScriptStatusChanged(status coreapi.ScriptStatusInfo) {
	go tui.app.HandleScriptStatusChanged(status)
}

func (tui *TuiApiImpl) OnScriptError(scriptName string, err error) {
	go tui.app.HandleScriptError(scriptName, err)
}

// Database event methods - database loading/unloading handler
func (tui *TuiApiImpl) OnDatabaseStateChanged(info coreapi.DatabaseStateInfo) {
	go tui.app.HandleDatabaseStateChanged(info)
}

// Game state event methods - simple sector change handler
func (tui *TuiApiImpl) OnCurrentSectorChanged(sectorInfo coreapi.SectorInfo) {
	go tui.app.HandleCurrentSectorChanged(sectorInfo)
}

// Port update event handler - detailed port information updates
func (tui *TuiApiImpl) OnPortUpdated(portInfo coreapi.PortInfo) {
	go tui.app.HandlePortUpdated(portInfo)
}

// Trader data event handler - called when trader information is captured
func (tui *TuiApiImpl) OnTraderDataUpdated(sectorNumber int, traders []coreapi.TraderInfo) {
	go tui.app.HandleTraderDataUpdated(sectorNumber, traders)
}

// Player statistics event handler - called when player stats are updated
func (tui *TuiApiImpl) OnPlayerStatsUpdated(stats coreapi.PlayerStatsInfo) {
	go tui.app.HandlePlayerStatsUpdated(stats)
}

// Sector update event handler - called when sector data is updated (e.g. from etherprobe)
func (tui *TuiApiImpl) OnSectorUpdated(sectorInfo coreapi.SectorInfo) {
	go tui.app.HandleSectorUpdated(sectorInfo)
}

// processDataLoop runs in a single goroutine to process all terminal data sequentially
func (tui *TuiApiImpl) processDataLoop() {
	for {
		select {
		case data := <-tui.dataChan:
			// Process data sequentially - no race conditions possible
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Panic in data processing - recovered
					}
				}()
				tui.app.HandleTerminalData(data)
			}()

		case <-tui.shutdownCh:
			return
		}
	}
}
