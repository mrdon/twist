# API Phase 1: Minimal Foundation for Connection and Data Flow

## Goal

Establish the minimal API foundation needed to connect to the game server and stream data to the TUI, without breaking existing functionality. This focuses on the absolute bare minimum to get connection and data flow working through the API layer.

## Overview

Phase 1 creates the basic API infrastructure and implements **only the connection-related functionality**. The TUI will be able to connect to the game server and receive streaming data through the API layer instead of direct coupling.

## Architectural Requirements

All API methods must follow the non-blocking architecture requirements detailed in `docs/api.md`. This is critical for Phase 1 since we'll be handling high-frequency data streaming.

## Scope - Connection Essentials Only

### Methods to Implement
**ProxyAPI Interface:**
```go
type ProxyAPI interface {
    Connect(address string, tuiAPI TuiAPI) error
    Disconnect() error
    IsConnected() bool
    SendData(data []byte) error  // Matches OnData for symmetry
    Shutdown() error  // For clean interface
}
```

**TuiAPI Interface:**
```go
type TuiAPI interface {
    // All methods MUST return immediately (use goroutines for async work)
    OnConnected(info ConnectionInfo)
    OnDisconnected(reason string)
    OnConnectionError(err error)
    OnData(data []byte)
}
```

**Data Types:**
```go
type ConnectionStatus int

const (
    ConnectionStatusDisconnected ConnectionStatus = iota
    ConnectionStatusConnecting
    ConnectionStatusConnected
)

func (cs ConnectionStatus) String() string {
    switch cs {
    case ConnectionStatusDisconnected:
        return "disconnected"
    case ConnectionStatusConnecting:
        return "connecting"
    case ConnectionStatusConnected:
        return "connected"
    default:
        return "unknown"
    }
}

type ConnectionInfo struct {
    Address     string           `json:"address"`
    ConnectedAt time.Time        `json:"connected_at"`
    Status      ConnectionStatus `json:"status"`
}
```

### What's NOT in Phase 1
- Script management (comes in Phase 3)
- Game state tracking (comes in Phase 4) 
- Advanced connection metrics
- Database integration
- Any TWX-inspired advanced features

## Implementation Steps

### Step 1: Create API Module Structure

#### 1.1 Create API Directory Structure
```bash
mkdir -p internal/proxy/api
mkdir -p internal/tui/api
```

#### 1.2 Create `internal/proxy/api/types.go`
```go
package api

import "time"

// ConnectionStatus represents the current connection state
type ConnectionStatus int

const (
    ConnectionStatusDisconnected ConnectionStatus = iota
    ConnectionStatusConnecting
    ConnectionStatusConnected
)

func (cs ConnectionStatus) String() string {
    switch cs {
    case ConnectionStatusDisconnected:
        return "disconnected"
    case ConnectionStatusConnecting:
        return "connecting"
    case ConnectionStatusConnected:
        return "connected"
    default:
        return "unknown"
    }
}

// ConnectionInfo represents connection state information
type ConnectionInfo struct {
    Address     string           `json:"address"`
    ConnectedAt time.Time        `json:"connected_at"`
    Status      ConnectionStatus `json:"status"`
}
```

#### 1.3 Create `internal/proxy/api/tui_api.go`
```go
package api

// TuiAPI defines notifications from Proxy to TUI
// 
// CRITICAL: All methods must return immediately (within microseconds) to avoid
// blocking the proxy. Use goroutines for any actual work and queue UI updates
// through tview's QueueUpdateDraw mechanism.
type TuiAPI interface {
    // Connection Events - all must return immediately
    OnConnected(info ConnectionInfo)
    OnDisconnected(reason string)
    OnConnectionError(err error)
    
    // Data Events - must return immediately (high frequency calls)
    OnData(data []byte)
}
```

#### 1.4 Create `internal/proxy/api/proxy_api.go`
```go
package api

// ProxyAPI defines commands from TUI to Proxy
type ProxyAPI interface {
    // Connection Management
    Connect(address string, tuiAPI TuiAPI) error
    Disconnect() error
    IsConnected() bool
    
    // Data Processing
    SendData(data []byte) error
    
    // Lifecycle
    Shutdown() error
}
```

### Step 2: Implement ProxyAPI Wrapper

**Architecture Note**: ProxyAPI and TuiAPI implementations must follow the thin orchestration layer pattern detailed in `docs/api.md`. Each method should be a one-liner delegating to specialized modules.

#### 2.1 Create `internal/proxy/api/proxy_api_impl.go`
```go
package api

import (
    "errors"
    "time"
    "twist/internal/proxy"
)

// ProxyApiImpl implements ProxyAPI as a thin orchestration layer
type ProxyApiImpl struct {
    proxy *proxy.Proxy  // Active proxy with built-in TuiAPI
}

// Connect creates a new proxy instance and returns a connected ProxyAPI
func Connect(address string, tuiAPI TuiAPI) (ProxyAPI, error) {
    if address == "" {
        return nil, errors.New("address cannot be empty")
    }
    
    // Create fresh proxy instance with TuiAPI
    proxyInstance := proxy.New(tuiAPI)
    
    // Attempt connection
    err := proxyInstance.Connect(address)
    if err != nil {
        // Connection failure -> call TuiAPI error callback
        tuiAPI.OnConnectionError(err)
        return nil, err
    }
    
    // Success -> call TuiAPI success callback
    connectionInfo := ConnectionInfo{
        Address:     address,
        ConnectedAt: time.Now(),
        Status:      ConnectionStatusConnected,
    }
    tuiAPI.OnConnected(connectionInfo)
    
    // Return connected ProxyAPI instance
    return &ProxyApiImpl{
        proxy: proxyInstance,
    }, nil
}

// Thin orchestration methods - all one-liners delegating to proxy
func (p *ProxyApiImpl) Connect(address string, tuiAPI TuiAPI) error {
    // Not used - Connect is now a static function
    return errors.New("use api.Connect() function instead")
}

func (p *ProxyApiImpl) Disconnect() error {
    go func() {
        tuiAPI := p.proxy.GetTuiAPI()
        err := p.proxy.Disconnect()
        if err != nil {
            tuiAPI.OnConnectionError(err)
        } else {
            tuiAPI.OnDisconnected("user requested")
        }
    }()
    return nil
}

func (p *ProxyApiImpl) IsConnected() bool {
    return p.proxy.IsConnected()
}

func (p *ProxyApiImpl) SendData(data []byte) error {
    return p.proxy.SendInput(string(data))
}

func (p *ProxyApiImpl) Shutdown() error {
    return p.proxy.Disconnect()
}
```

### Step 3: Implement TuiAPI in TUI Module

#### 3.1 Create `internal/tui/api/tui_api_impl.go`
```go
package api

import (
    proxyapi "twist/internal/proxy/api"
)

// TuiApiImpl implements TuiAPI as a thin orchestration layer
type TuiApiImpl struct {
    app *TwistApp
}

// NewTuiAPI creates a new TuiAPI implementation
func NewTuiAPI(app *TwistApp) proxyapi.TuiAPI {
    return &TuiApiImpl{
        app: app,
    }
}

// Thin orchestration methods - all one-liners calling app directly
func (tui *TuiApiImpl) OnConnected(info proxyapi.ConnectionInfo) {
    go tui.app.handleConnectionEstablished(info)
}

func (tui *TuiApiImpl) OnDisconnected(reason string) {
    go tui.app.handleDisconnection(reason)
}

func (tui *TuiApiImpl) OnConnectionError(err error) {
    go tui.app.handleConnectionError(err)
}

func (tui *TuiApiImpl) OnData(data []byte) {
    go tui.app.handleTerminalData(data)
}
```

### Step 3.5: Keep It Simple

For Phase 1, we'll keep everything simple and add manager classes later if needed. The TuiAPI calls the app directly.

#### 3.2 Create `internal/tui/api/proxy_client.go`
```go
package api

import (
    proxyapi "twist/internal/proxy/api"
)

// ProxyClient manages ProxyAPI connections for TUI
type ProxyClient struct {
    currentAPI proxyapi.ProxyAPI  // Current active connection (nil if disconnected)
}

// NewProxyClient creates a new proxy client
func NewProxyClient() *ProxyClient {
    return &ProxyClient{
        currentAPI: nil,
    }
}

func (pc *ProxyClient) Connect(address string, tuiAPI proxyapi.TuiAPI) error {
    // Use static Connect function to create new ProxyAPI instance
    api, err := proxyapi.Connect(address, tuiAPI)
    if err != nil {
        return err
    }
    
    // Store the connected API instance
    pc.currentAPI = api
    return nil
}

func (pc *ProxyClient) Disconnect() error {
    if pc.currentAPI == nil {
        return nil
    }
    
    err := pc.currentAPI.Disconnect()
    pc.currentAPI = nil  // Clear reference after disconnect
    return err
}

func (pc *ProxyClient) IsConnected() bool {
    if pc.currentAPI == nil {
        return false
    }
    return pc.currentAPI.IsConnected()
}

func (pc *ProxyClient) SendData(data []byte) error {
    if pc.currentAPI == nil {
        return errors.New("not connected")
    }
    return pc.currentAPI.SendData(data)
}

func (pc *ProxyClient) Shutdown() error {
    if pc.currentAPI == nil {
        return nil
    }
    
    err := pc.currentAPI.Shutdown()
    pc.currentAPI = nil  // Clear reference after shutdown
    return err
}
```

### Step 4: Update Streaming Pipeline for TuiAPI Calls

#### 4.1 Modify `internal/streaming/pipeline.go`
Update the pipeline to receive TuiAPI in constructor and use it for data callbacks:

```go
// Add TuiAPI field to Pipeline struct
type Pipeline struct {
    // ... existing fields ...
    tuiAPI api.TuiAPI
}

// Update constructor to accept TuiAPI
func New(tuiAPI api.TuiAPI) *Pipeline {
    return &Pipeline{
        // ... existing field initialization ...
        tuiAPI: tuiAPI,
    }
}

// Find the existing data processing method and update it
// Look for where decoded data is written to terminalWriter
func (p *Pipeline) processIncomingData(data []byte) {
    // ... existing decoding logic ...
    
    // Replace: p.terminalWriter.Write(decoded)
    // With: 
    if p.tuiAPI != nil {
        p.tuiAPI.OnData(decoded)
    } else {
        // Fallback for transition period
        p.terminalWriter.Write(decoded)
    }
}
```

#### 4.2 Update Proxy Constructor
The proxy receives TuiAPI in its constructor and passes it to components:

```go
// In internal/proxy/proxy.go - update constructor signature:
func New(tuiAPI api.TuiAPI) *Proxy {
    p := &Proxy{
        // ... existing fields ...
        tuiAPI: tuiAPI,
    }
    
    // Pass TuiAPI to pipeline constructor
    p.pipeline = pipeline.New(tuiAPI)
    
    // ... rest of existing initialization ...
    
    return p
}

// Add field to Proxy struct
type Proxy struct {
    // ... existing fields ...
    tuiAPI api.TuiAPI
}

// Add getter for TuiAPI (used by ProxyApiImpl.Disconnect for callbacks)
func (p *Proxy) GetTuiAPI() api.TuiAPI {
    return p.tuiAPI
}
```

### Step 5: Update TUI App to Use API Layer

#### 5.1 Modify `internal/tui/app.go` - Add API Layer
```go
// Add to TwistApp struct
type TwistApp struct {
    // ... existing fields ...
    
    // API layer (alongside existing proxy for now)
    proxyClient *api.ProxyClient
    tuiAPI      proxyapi.TuiAPI
}

// In NewApplication() function:
func NewApplication() *TwistApp {
    // ... existing code ...
    
    twistApp := &TwistApp{
        // ... existing fields ...
    }
    
    // Create API layer - proxy instances created per connection via static Connect()
    twistApp.proxyClient = api.NewProxyClient()  // No ProxyAPI needed upfront
    twistApp.tuiAPI = api.NewTuiAPI(twistApp)     // TuiAPI calls app directly
    
    // ... rest of existing setup ...
    
    return twistApp
}
```

#### 5.2 Add TuiAPI Handler Methods to TwistApp
```go
// Add methods to handle TuiAPI callbacks directly
func (ta *TwistApp) handleConnectionEstablished(info api.ConnectionInfo) {
    ta.QueueUpdateDraw(func() {
        ta.connected = true
        ta.serverAddress = info.Address
        ta.menuComponent.SetConnectedMenu()
    })
}

func (ta *TwistApp) handleDisconnection(reason string) {
    ta.QueueUpdateDraw(func() {
        ta.connected = false
        ta.serverAddress = ""
        ta.menuComponent.SetDisconnectedMenu()
    })
}

func (ta *TwistApp) handleConnectionError(err error) {
    ta.QueueUpdateDraw(func() {
        ta.connected = false
        ta.serverAddress = ""
        ta.menuComponent.SetDisconnectedMenu()
        // Could show error modal: ta.showErrorModal(err)
    })
}

func (ta *TwistApp) handleTerminalData(data []byte) {
    // High-frequency data - could use channel if needed for performance
    ta.QueueUpdateDraw(func() {
        ta.terminal.Write(data)
        ta.terminal.Refresh()
    })
}
```

#### 5.3 Replace Connection Methods to Use API
```go
// Replace existing connect() method - complete migration
func (ta *TwistApp) connect(address string) {
    // Use API layer exclusively - connection state updated via callbacks
    if err := ta.proxyClient.Connect(address, ta.tuiAPI); err != nil {
        // Handle immediate validation errors
        ta.setConnectionState(false, "")
        ta.menuComponent.SetDisconnectedMenu()
        return
    }
    // Connection state will be updated via OnConnected/OnConnectionError callbacks
}

func (ta *TwistApp) disconnect() {
    // Use API layer exclusively
    ta.proxyClient.Disconnect()
    // Disconnection state will be updated via OnDisconnected callback
}

func (ta *TwistApp) sendCommand(command string) {
    if ta.proxyClient.IsConnected() {
        ta.proxyClient.SendData([]byte(command))
    }
}
```

## Testing Phase 1

### Manual Testing Steps
1. **Startup**: Verify app starts without errors
2. **API Creation**: Verify API objects are created successfully  
3. **Connection**: Test connecting to game server
4. **Data Flow**: Verify game data appears in terminal
5. **Performance**: Verify no lag or blocking during high-frequency data streams
6. **Disconnect**: Test disconnecting from server
7. **Shutdown**: Verify clean shutdown

### Performance Testing
- **Data Throughput**: Connect to busy game server and verify smooth data flow
- **UI Responsiveness**: Verify UI remains responsive during heavy network activity  
- **No Blocking**: Verify proxy doesn't pause/stutter during UI updates
- **Goroutine Management**: Check that goroutines are created/cleaned up properly

### Expected Behavior
- All existing functionality continues to work
- API layer exists alongside current implementation
- Connection can be established through either path
- Terminal data flows through API layer when configured
- No breaking changes to existing user experience

## Success Criteria

✅ **API Infrastructure Created**: All API files exist and compile  
✅ **Connection Works**: Can connect to game server through API  
✅ **Data Streaming**: Game data flows from proxy → TuiAPI → terminal  
✅ **No Regressions**: All existing functionality still works  
✅ **Clean Shutdown**: API layer cleans up properly  

## Next Steps (Phase 2)

Phase 2 will:
- Replace existing connection methods with API-only versions
- Remove `streaming.TerminalWriter` dependency completely  
- Remove dual connection paths (use API exclusively)
- Add connection state management in proxy

## Key Architectural Improvements

### Thin Orchestration Layer Pattern
Both proxy and TUI sides now use manager interfaces:

**Proxy Side Managers:**
- `ConnectionManager` - handles connection logic and stores TuiAPI reference
- `DataManager` - handles data transmission

**TUI Side Managers:**
- `ConnectionEventManager` - handles connection state changes
- `TerminalDataManager` - handles high-frequency terminal data with channels

### Performance Optimizations
- **Channel-based terminal data processing** - `TerminalDataManager` uses buffered channel for high-frequency data
- **Non-blocking TuiAPI calls** - all methods return immediately via goroutines
- **Efficient data flow** - TuiAPI reference flows from ConnectionManager through proxy components

### Complete Migration Strategy
- **Replace existing connection methods entirely** - no backwards compatibility needed
- **Connection state managed via callbacks** - TUI receives updates through TuiAPI methods
- **Error handling through callbacks** - manager failures trigger appropriate TuiAPI error methods

## Files Created/Modified Summary

### New Files:
- `internal/proxy/api/types.go` - Data types (ConnectionInfo, ConnectionStatus enum)
- `internal/proxy/api/tui_api.go` - TuiAPI interface definition
- `internal/proxy/api/proxy_api.go` - ProxyAPI interface definition  
- `internal/proxy/api/proxy_api_impl.go` - ProxyAPI implementation with static Connect function
- `internal/tui/api/tui_api_impl.go` - TuiAPI implementation that calls app directly
- `internal/tui/api/proxy_client.go` - ProxyClient wrapper for TUI usage

### Modified Files:
- `internal/streaming/pipeline.go` - Update constructor to accept TuiAPI, replace terminalWriter.Write() with tuiAPI.OnData()
- `internal/proxy/proxy.go` - Update constructor to accept TuiAPI, add tuiAPI field, add GetTuiAPI() method
- `internal/tui/app.go` - Add API layer creation, add TuiAPI handler methods, replace connection methods

### Key Implementation Notes:
1. **Static Connect Pattern**: Use `api.Connect(address, tuiAPI)` function instead of method on instance
2. **No Manager Classes**: Keep proxy and app methods simple, add managers later if needed
3. **No Null Checks**: ProxyApiImpl always has valid proxy, TuiApiImpl always has valid app
4. **Direct Delegation**: Thin orchestration layers that directly call proxy/app methods
5. **Pipeline Integration**: Find existing data processing method and replace terminalWriter.Write() calls

### Import Changes:
- TUI can import `internal/proxy/api` (API types only)
- No changes to forbidden imports yet (comes in later phases)

### Testing Approach:
- Manual testing: startup → connect → data flow → disconnect → shutdown
- Focus on performance: verify no lag during high-frequency data streams
- Verify existing functionality unchanged during transition

This phase establishes clean separation with minimal complexity while providing complete connection functionality through the API layer.

## Implementation Notes for New Developer

### Critical Points:
1. **Find the Right Pipeline Method**: Look for the method in `internal/streaming/pipeline.go` that processes incoming data and writes to `terminalWriter`. This might be called `processData`, `handleIncoming`, `processIncomingData`, or similar.

2. **Pipeline Constructor**: The existing pipeline constructor likely takes different parameters. You'll need to update both the signature and all callers.

3. **Proxy Constructor**: Similarly, the proxy constructor will need signature changes and all existing callers updated.

4. **App Method Naming**: The TuiAPI handler methods (like `handleConnectionEstablished`) are new - you'll need to add these to the `TwistApp` struct.

5. **Import Statements**: Make sure to add proper imports:
   ```go
   // In proxy files
   import "twist/internal/proxy/api"
   
   // In TUI files  
   import proxyapi "twist/internal/proxy/api"
   ```

### Common Issues:
- **Circular imports**: Keep the import direction clean - TUI imports proxy/api, not the reverse
- **Method signatures**: Update all callers when changing constructor signatures
- **Missing error handling**: The static `Connect` can return errors that need handling
- **Goroutine cleanup**: Make sure the async operations in Connect/Disconnect don't leak goroutines

### Verification:
1. **Compile check**: All files should compile after changes
2. **Connection flow**: Verify the complete flow: `ProxyClient.Connect()` → `api.Connect()` → `proxy.New()` → `proxy.Connect()` → `tuiAPI.OnConnected()`
3. **Data flow**: Verify: game server data → pipeline → `tuiAPI.OnData()` → app handler → UI update