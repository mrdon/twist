package api

import (
	"twist/internal/debug"
	coreapi "twist/internal/api"
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
	debug.Log("TuiApiImpl: OnConnectionStatusChanged called - status: %v, address: %s", status, address)
	go tui.app.HandleConnectionStatusChanged(status, address)
}

func (tui *TuiApiImpl) OnConnectionError(err error) {
	debug.Log("TuiApiImpl: OnConnectionError called - error: %v", err)
	go tui.app.HandleConnectionError(err)
}

func (tui *TuiApiImpl) OnData(data []byte) {
	// Log raw data chunks for debugging
	debug.LogDataChunk("OnData", data)
	
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
	debug.Log("TuiApiImpl: OnScriptStatusChanged called - status: %+v", status)
	go tui.app.HandleScriptStatusChanged(status)
}

func (tui *TuiApiImpl) OnScriptError(scriptName string, err error) {
	debug.Log("TuiApiImpl: OnScriptError called - scriptName: %s, error: %v", scriptName, err)
	go tui.app.HandleScriptError(scriptName, err)
}

// Database event methods - database loading/unloading handler
func (tui *TuiApiImpl) OnDatabaseStateChanged(info coreapi.DatabaseStateInfo) {
	debug.Log("TuiApiImpl: OnDatabaseStateChanged called - info: %+v", info)
	go tui.app.HandleDatabaseStateChanged(info)
}

// Game state event methods - simple sector change handler
func (tui *TuiApiImpl) OnCurrentSectorChanged(sectorInfo coreapi.SectorInfo) {
	debug.Log("TuiApiImpl: OnCurrentSectorChanged called - sectorInfo: %+v", sectorInfo)
	go tui.app.HandleCurrentSectorChanged(sectorInfo)
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