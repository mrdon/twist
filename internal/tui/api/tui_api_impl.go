package api

import (
	proxyapi "twist/internal/proxy/api"
	"twist/internal/debug"
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
	app TwistApp
}

// NewTuiAPI creates a new TuiAPI implementation
func NewTuiAPI(app TwistApp) proxyapi.TuiAPI {
	return &TuiApiImpl{
		app: app,
	}
}

// Thin orchestration methods - all one-liners calling app directly
// All methods MUST return immediately using goroutines for async work
func (tui *TuiApiImpl) OnConnected(info proxyapi.ConnectionInfo) {
	debug.Log("TuiAPI.OnConnected called with info: %+v", info)
	go tui.app.HandleConnectionEstablished(info)
	debug.Log("TuiAPI.OnConnected dispatched to app handler")
}

func (tui *TuiApiImpl) OnDisconnected(reason string) {
	debug.Log("TuiAPI.OnDisconnected called with reason: %s", reason)
	go tui.app.HandleDisconnection(reason)
	debug.Log("TuiAPI.OnDisconnected dispatched to app handler")
}

func (tui *TuiApiImpl) OnConnectionError(err error) {
	debug.LogError(err, "TuiAPI.OnConnectionError")
	go tui.app.HandleConnectionError(err)
	debug.Log("TuiAPI.OnConnectionError dispatched to app handler")
}

func (tui *TuiApiImpl) OnData(data []byte) {
	debug.Log("TuiAPI.OnData called with %d bytes", len(data))
	go tui.app.HandleTerminalData(data)
}