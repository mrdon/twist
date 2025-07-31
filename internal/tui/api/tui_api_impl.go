package api

import (
	"fmt"
	proxyapi "twist/internal/proxy/api"
)

// Forward declaration - will be defined when we update app.go
type TwistApp interface {
	HandleConnectionEstablished(info proxyapi.ConnectionInfo)
	HandleDisconnection(reason string)
	HandleConnectionError(err error)
	HandleTerminalData(data []byte)
}

// TuiApiImpl implements TuiAPI as a thin orchestration layer
type TuiApiImpl struct {
	app        TwistApp
	dataChan   chan []byte
	shutdownCh chan struct{}
}

// NewTuiAPI creates a new TuiAPI implementation
func NewTuiAPI(app TwistApp) proxyapi.TuiAPI {
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
func (tui *TuiApiImpl) OnConnected(info proxyapi.ConnectionInfo) {
	go tui.app.HandleConnectionEstablished(info)
}

func (tui *TuiApiImpl) OnDisconnected(reason string) {
	fmt.Printf("[DEBUG] TuiApiImpl.OnDisconnected called with reason: %s\n", reason)
	go tui.app.HandleDisconnection(reason)
}

func (tui *TuiApiImpl) OnConnectionError(err error) {
	go tui.app.HandleConnectionError(err)
}

func (tui *TuiApiImpl) OnData(data []byte) {
	
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

// processDataLoop runs in a single goroutine to process all terminal data sequentially
func (tui *TuiApiImpl) processDataLoop() {
	
	for {
		select {
		case data := <-tui.dataChan:
				// Process data sequentially - no race conditions possible
			func() {
				defer func() {
					if r := recover(); r != nil {
						}
				}()
				tui.app.HandleTerminalData(data)
			}()
			
		case <-tui.shutdownCh:
			return
		}
	}
}