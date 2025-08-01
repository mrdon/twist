# API Phase 2: Eliminate TerminalWriter and Enforce Pure API Communication

## Goal

Eliminate the TerminalWriter interface completely and establish pure API-only communication between proxy and TUI. No backwards compatibility - clean architectural separation is the priority.

## Overview

Phase 2 completes the architectural migration by:
1. **Eliminating TerminalWriter interface completely** - no coupling through interfaces
2. **Making proxy constructor TuiAPI-native** - proxy receives TuiAPI directly  
3. **Updating streaming pipeline to use TuiAPI directly** - no intermediate interfaces
4. **Removing all direct proxy-TUI coupling** - API is the exclusive interface
5. **Clean proxy-TUI separation** - proxy only knows about TuiAPI, never TUI internals

## Current State Analysis (Actual Implementation)

### ✅ What Was Successfully Implemented:

**TuiAPI Interface with Clean Methods**:
- `OnConnectionStatusChanged(status ConnectionStatus, address string)` - single callback for all connection state changes
- `OnConnectionError(err error)` - connection errors
- `OnData(data []byte)` - data streaming from proxy to TUI
- Channel-based data processing with dedicated goroutine
- Performance-optimized with buffered channels

**Symmetric API Design**:
- `TuiAPI.OnData(data []byte)` - proxy sends data to TUI
- `ProxyAPI.SendData(data []byte)` - TUI sends data to proxy  
- Clean bidirectional communication with consistent method signatures

**ProxyAPI Lifecycle**:
- One ProxyAPI instance per connection
- TUI disposes ProxyAPI when connection ends
- Create new ProxyAPI instance for new connections

**API-First Connection Flow**: TUI uses API layer for connections:
- `ta.proxyClient.Connect(address, ta.tuiAPI)` 
- Full callback system: `HandleConnecting`, `HandleConnectionStatusChanged`, etc.
- Async connection with timeout handling

### ❌ Major Architectural Problems to Fix:

**TerminalWriter Interface Still Exists**: 
- `streaming.TerminalWriter` interface at `internal/streaming/pipeline.go:50`
- Pipeline still calls `p.terminalWriter.Write(decoded)` at line 157
- `ProxyApiImpl` implements TerminalWriter as a bridge (lines 82-88)
- **This creates unnecessary coupling - proxy should call TuiAPI directly**

**Dual Proxy Architecture**: TUI maintains both patterns:
- Direct proxy: `proxyInstance := proxy.New(nil)` (line 56 in app.go)  
- API layer: `proxyClient` and `tuiAPI` (lines 86-87 in app.go)
- TUI struct has both `proxy *proxy.Proxy` field AND API fields

**Legacy Proxy Constructor**: Constructor expects TerminalWriter:
- `func New(terminalWriter streaming.TerminalWriter) *Proxy`
- This forces either `nil` (broken) or bridge pattern (unnecessary coupling)

**Script Manager Direct Access**: Bypasses API layer:
- `statusComp.SetScriptManager(proxyInstance.script management())` (line 98)

## Scope - Clean Interface Design

### Final Interface Design for Phase 2:

**ProxyAPI Interface (clean and minimal):**
```go
type ProxyAPI interface {
    // Connection Management
    Disconnect() error
    IsConnected() bool
    
    // Data Processing (symmetric with OnData)  
    SendData(data []byte) error
}
```

**Note**: No `Connect()` method - connection handled by static `api.Connect()` function only.

**TuiAPI Interface (simplified - no redundant callbacks):**
```go
type TuiAPI interface {
    // Connection Events - single callback for all status changes
    OnConnectionStatusChanged(status ConnectionStatus, address string)
    OnConnectionError(err error)
    
    // Data Events - high frequency
    OnData(data []byte)
}
```

**Data Types (minimal - no unnecessary metrics):**
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
```

### ProxyAPI Lifecycle Model:
- **One ProxyAPI instance per connection** - created by `api.Connect()` static function
- **No Connect() method**: Interface has no Connect method - only static function creates connected instances
- **Disposable**: TUI discards ProxyAPI when connection ends
- **Fresh instances**: Create new ProxyAPI for each new connection  
- **No connection pooling**: ProxyClient simply holds current ProxyAPI reference
- **Simple disposal**: Call `Disconnect()` then set `ProxyClient.currentAPI = nil`
- **No Shutdown() method**: Redundant with `Disconnect()` - eliminated for simplicity

### Key Implementation Clarifications:
- **Non-blocking Connection**: `proxy.Connect()` returns immediately, reports status via callbacks
- **Single Pipeline Constructor**: No script manager version - deferred to Phase 3
- **Address Tracking**: Proxy stores `currentAddress` field for callback context
- **No Immediate Failures**: `api.Connect()` never returns errors - even validation errors go via `OnConnectionError()` callback
- **Channel-based Data**: Current `TuiApiImpl` channel approach sufficient for high-frequency `OnData()`
- **Method Naming Consistency**: Use `HandleConnectionStatusChanged()` in TUI for clean, consistent naming
- **Remove Unused Types**: Delete `ConnectionInfo` struct completely - no longer used
- **Import Structure**: Both sides can import `internal/proxy/api` - watch for circular imports

### What's NOT in Phase 2:
- Script management (comes in Phase 3)
- Game state tracking (comes in Phase 4)  
- Connection metrics/statistics (unnecessary complexity)
- Advanced TWX-inspired features

### What Will Be Temporarily Broken:
- **Script status display**: Status component won't show script information
- **Direct script manager access**: `proxyInstance.GetScriptManager()` calls removed
- **Script lifecycle UI**: Script loading/stopping through TUI will not work
- **Acceptable breakage**: Clean API separation takes priority over temporary functionality

## Callback Interface Migration Tasks

### Current vs New Callback Interface

**Current Interface (needs changing):**
```go
type TuiAPI interface {
    OnConnecting(address string)
    OnConnected(info ConnectionInfo)
    OnDisconnected(reason string)
    OnConnectionError(err error)
    OnData(data []byte)
}
```

**New Interface (simplified):**
```go
type TuiAPI interface {
    OnConnectionStatusChanged(status ConnectionStatus, address string)
    OnConnectionError(err error)
    OnData(data []byte)
}
```

### Files That Need Updates

#### 1. **internal/proxy/api/tui_api.go**
- **Change**: Update interface definition
- **Replace**: `OnConnecting()`, `OnConnected()`, `OnDisconnected()` methods
- **With**: Single `OnConnectionStatusChanged(status ConnectionStatus, address string)` method
- **Keep**: `OnConnectionError()` and `OnData()` unchanged

#### 2. **internal/proxy/api/proxy_api_impl.go** 
- **Change**: Update 8 TuiAPI method calls
- **Line 39**: `tuiAPI.OnConnecting(address)` → `tuiAPI.OnConnectionStatusChanged(ConnectionStatusConnecting, address)`
- **Line 67**: `tuiAPI.OnConnected(connectionInfo)` → `tuiAPI.OnConnectionStatusChanged(ConnectionStatusConnected, address)`
- **Line 106**: `tuiAPI.OnDisconnected("user requested")` → `tuiAPI.OnConnectionStatusChanged(ConnectionStatusDisconnected, address)`
- **Line 137**: `tuiAPI.OnDisconnected("shutdown")` → `tuiAPI.OnConnectionStatusChanged(ConnectionStatusDisconnected, address)`
- **Line 163**: `tuiAPI.OnDisconnected("connection closed")` → `tuiAPI.OnConnectionStatusChanged(ConnectionStatusDisconnected, address)`
- **Line 172**: `tuiAPI.OnDisconnected("connection lost: " + err.Error())` → `tuiAPI.OnConnectionStatusChanged(ConnectionStatusDisconnected, address)`
- **Line 181**: `tuiAPI.OnDisconnected("connection lost")` → `tuiAPI.OnConnectionStatusChanged(ConnectionStatusDisconnected, address)`
- **Keep**: `OnConnectionError()` calls unchanged (lines 57, 74, 104, 135)

#### 3. **internal/tui/api/tui_api_impl.go**
- **Change**: Update TuiApiImpl method implementations  
- **Remove**: `OnConnecting()`, `OnConnected()`, `OnDisconnected()` methods (lines 41-53)
- **Add**: Single `OnConnectionStatusChanged(status proxyapi.ConnectionStatus, address string)` method
- **Update**: Forward declaration interface for TwistApp (lines 10-16)
- **Keep**: `OnConnectionError()` and `OnData()` unchanged

#### 4. **internal/tui/app.go**
- **Change**: Update TwistApp handler methods
- **Remove**: `HandleConnecting()` (line 255), `HandleConnectionEstablished()` (line 263), `HandleDisconnection()` (line 277)
- **Add**: Single `HandleConnectionStatusChanged(status proxyapi.ConnectionStatus, address string)` method
- **Logic**: Combine all three handlers into switch statement on status
- **Keep**: `HandleConnectionError()` and `HandleTerminalData()` unchanged

#### 5. **internal/proxy/api/types.go**
- **Remove**: `ConnectionInfo` struct entirely (lines 27-32) - unused after callback changes
- **Keep**: `ConnectionStatus` enum and String() method (lines 5-25)

### Implementation Strategy

**Order of Changes:**
1. **Update interface definition** in `tui_api.go` first
2. **Update proxy calls** in `proxy_api_impl.go` to use new callback
3. **Update TUI implementation** in `tui_api_impl.go` to implement new interface
4. **Update TUI handlers** in `app.go` to handle consolidated callback
5. **Clean up unused types** in `types.go` if desired

**Combined Handler Logic:**
```go
func (ta *TwistApp) HandleConnectionStatusChanged(status proxyapi.ConnectionStatus, address string) {
    ta.app.QueueUpdateDraw(func() {
        switch status {
        case proxyapi.ConnectionStatusConnecting:
            // Logic from old HandleConnecting()
            ta.statusComponent.SetConnectionStatus(false, "Connecting to "+address+"...")
            
        case proxyapi.ConnectionStatusConnected:
            // Logic from old HandleConnectionEstablished()
            ta.connected = true
            ta.serverAddress = address
            ta.menuComponent.SetConnectedMenu()
            ta.statusComponent.SetConnectionStatus(true, address)
            if ta.modalVisible {
                ta.closeModal()
            }
            
        case proxyapi.ConnectionStatusDisconnected:
            // Logic from old HandleDisconnection()
            ta.connected = false
            ta.serverAddress = ""
            ta.menuComponent.SetDisconnectedMenu()
            ta.statusComponent.SetConnectionStatus(false, "")
            // Show disconnect message in terminal
            disconnectMsg := "\r\x1b[K\x1b[31;1mDISCONNECTED\x1b[0m\n"
            ta.terminalComponent.Write([]byte(disconnectMsg))
        }
    })
}
```

### Testing Requirements

After implementing callback changes:
- **Connection flow**: Verify connecting/connected/disconnected states work
- **UI updates**: Verify menu and status components update correctly  
- **Error handling**: Verify `OnConnectionError()` still works independently
- **Terminal display**: Verify disconnect message still appears
- **Modal management**: Verify connection modal closes properly

## Implementation Steps

### Step 1: Eliminate TerminalWriter Interface Completely

**Core Principle**: Proxy should only know about TuiAPI - no intermediate interfaces.

#### 1.1 Remove TerminalWriter Interface from Pipeline

Update `internal/streaming/pipeline.go`:

```go
// REMOVE this interface completely:
// type TerminalWriter interface {
//     Write(data []byte) error  
// }

// UPDATE Pipeline struct - replace TerminalWriter with TuiAPI
type Pipeline struct {
    // Input
    rawDataChan chan []byte
    
    // Processing layers
    telnetHandler  *telnet.Handler
    // REMOVE: terminalWriter TerminalWriter
    tuiAPI         api.TuiAPI  // Direct TuiAPI reference
    decoder        *encoding.Decoder
    sectorParser   *parser.SectorParser
    scriptManager  ScriptManager
    
    // ... rest of fields unchanged ...
}
```

#### 1.2 Update Pipeline Constructor

Replace TerminalWriter parameter with TuiAPI and use single constructor:

```go
// UPDATE to single constructor (script manager handled in Phase 3)
func NewPipeline(tuiAPI api.TuiAPI, db database.Database) *Pipeline {
    return &Pipeline{
        rawDataChan:   make(chan []byte, 100),
        tuiAPI:        tuiAPI,  // Direct TuiAPI reference
        decoder:       charmap.CodePage850.NewDecoder(),
        sectorParser:  parser.NewSectorParser(),
        // ... rest of initialization without script manager ...
        batchSize:     1024,
        batchTimeout:  10 * time.Millisecond,
        stopChan:      make(chan struct{}),
    }
}

// REMOVE: NewPipelineWithScriptManager - script manager in Phase 3
// REMOVE: writer func([]byte) error parameter - replaced by tuiAPI.OnData()
```

#### 1.3 Replace TerminalWriter.Write() with Direct TuiAPI Calls

Find the data processing method (around line 157) and update:

```go
// FIND this line:
// p.terminalWriter.Write(decoded)

// REPLACE with direct TuiAPI call:
if p.tuiAPI != nil {
    p.tuiAPI.OnData(decoded)
}
```

### Step 2: Update Proxy Constructor to be TuiAPI-Native

#### 2.1 Update Proxy Constructor Signature

Update `internal/proxy/proxy.go`:

```go
// UPDATE constructor signature - remove TerminalWriter completely
func New(tuiAPI api.TuiAPI) *Proxy {
    // ... existing database and script manager initialization ...
    
    p := &Proxy{
        outputChan:    make(chan string, 100),
        inputChan:     make(chan string, 100),
        errorChan:     make(chan error, 100),
        db:            db,
        scriptManager: scriptManager,
        tuiAPI:        tuiAPI,  // Store TuiAPI reference
    }
    
    // UPDATE pipeline creation - simplified constructor
    p.pipeline = streaming.NewPipeline(tuiAPI, db)
    
    return p
}

// ADD TuiAPI field to Proxy struct
type Proxy struct {
    conn     net.Conn
    reader   *bufio.Reader
    writer   *bufio.Writer
    mu       sync.RWMutex
    connected bool
    
    // Channels for communication
    outputChan chan string
    inputChan  chan string
    errorChan  chan error
    
    // Core components
    pipeline      *streaming.Pipeline
    scriptManager *scripting.ScriptManager
    db            database.Database
    
    // NEW: Direct TuiAPI reference
    tuiAPI api.TuiAPI
    
    // NEW: Connection tracking for callbacks
    currentAddress string  // Track address for OnConnectionStatusChanged callbacks
}
```

### Step 3: Update ProxyApiImpl to Remove TerminalWriter Bridge

#### 3.1 Remove TerminalWriter Implementation from ProxyApiImpl

Update `internal/proxy/api/proxy_api_impl.go`:

```go
// REMOVE the Write method completely:
// func (p *ProxyApiImpl) Write(data []byte) {
//     if p.tuiAPI != nil {
//         p.tuiAPI.OnData(data)
//     }
// }
// ProxyApiImpl should NOT implement TerminalWriter!

// UPDATE Connect function - proxy constructor now takes TuiAPI
func Connect(address string, tuiAPI TuiAPI) ProxyAPI {
    // Never return errors - all failures go via callbacks
    // Even nil tuiAPI or empty address should be handled gracefully via callbacks
    if tuiAPI == nil {
        // This is a programming error, but handle gracefully
        return &ProxyApiImpl{} // Will fail safely when used
    }
    
    // Create ProxyAPI wrapper
    impl := &ProxyApiImpl{
        tuiAPI: tuiAPI,
    }
    
    // Create proxy instance with TuiAPI directly - no bridge needed
    proxyInstance := proxy.New(tuiAPI)  // Proxy constructor now takes TuiAPI
    impl.proxy = proxyInstance
    
    // Start connection attempt (always async - never fails immediately)
    // Even address validation errors go via callbacks
    tuiAPI.OnConnectionStatusChanged(ConnectionStatusConnecting, address)
    go func() {
        err := proxyInstance.Connect(address)
        if err != nil {
            tuiAPI.OnConnectionError(err)
            tuiAPI.OnConnectionStatusChanged(ConnectionStatusDisconnected, address)
        } else {
            tuiAPI.OnConnectionStatusChanged(ConnectionStatusConnected, address)
        }
    }()
    
    return impl
}
```

#### 3.2 Add Enhanced ProxyAPI Methods

Add missing methods to complete the API:

```go
// ADD enhanced methods to ProxyApiImpl  
func (p *ProxyApiImpl) SendData(data []byte) error {
    if p.proxy == nil {
        return errors.New("not connected")
    }
    p.proxy.SendInput(string(data))  // Convert to string for existing proxy method
    return nil
}

// REMOVE Shutdown() method - redundant with Disconnect()
// Current implementation shows both methods do identical work except for reason string

func (p *ProxyApiImpl) script management() interface{} {
    if p.proxy == nil {
        return nil
    }
    return p.proxy.script management()
}

// connection status management removed - status tracked via callbacks
```

### Step 4: Remove Direct Proxy Instance from TUI

Complete the clean separation by removing dual proxy architecture.

#### 4.1 Update `internal/tui/app.go` - Remove Direct Proxy

```go
// UPDATE TwistApp struct - remove direct proxy field completely
type TwistApp struct {
    app    *tview.Application
    // REMOVE: proxy  *proxy.Proxy
    
    // API layer (now exclusive)
    proxyClient *api.ProxyClient
    tuiAPI      proxyapi.TuiAPI
    
    // ... rest of existing fields unchanged ...
}

// UPDATE NewApplication() - remove direct proxy creation
func NewApplication() *TwistApp {
    // REMOVE these lines completely:
    // proxyInstance := proxy.New(nil)
    
    // Create the main application
    app := tview.NewApplication()
    
    // ... existing UI component creation unchanged ...
    
    twistApp := &TwistApp{
        app:                app,
        // REMOVE: proxy:      proxyInstance,
        terminal:           nil,
        connected:          false,
        serverAddress:      "twgs.geekm0nkey.com:23",
        terminalUpdateChan: make(chan struct{}, 100),
        menuComponent:      menuComp,
        terminalComponent:  terminalComp,
        panelComponent:     panelComp,
        statusComponent:    statusComp,
        inputHandler:       inputHandler,
        globalShortcuts:    twistComponents.NewGlobalShortcutManager(),
    }
    
    // API layer creation (unchanged)
    twistApp.proxyClient = api.NewProxyClient()
    twistApp.tuiAPI = api.NewTuiAPI(twistApp)
    
    // REMOVE direct script manager access:
    // statusComp.SetScriptManager(proxyInstance.script management())
    // Script manager will be set via API after connection established
    
    // ... rest of existing setup unchanged ...
    
    return twistApp
}
```

#### 4.2 Update Connection Handler to Set Script Manager via API

Update the `HandleConnectionStatusChanged` method:

```go
func (ta *TwistApp) HandleConnectionStatusChanged(info proxyapi.ConnectionInfo) {
    ta.QueueUpdateDraw(func() {
        ta.connected = true
        ta.serverAddress = info.Address
        ta.menuComponent.SetConnectedMenu()
        ta.statusComponent.SetConnectionStatus(true, "Connected to "+info.Address)
        
        // Script manager setup removed - will be handled in Phase 3
    })
}
```

### Step 5: Update Interface Definitions and ProxyClient

#### 5.1 Update ProxyAPI Interface Definition

Update `internal/proxy/api/proxy_api.go`:

```go
type ProxyAPI interface {
    // Connection Management
    Connect(address string, tuiAPI TuiAPI) error
    Disconnect() error
    IsConnected() bool
    
    // Data Processing (symmetric with OnData)
    SendData(data []byte) error        // Matches OnData callback
    
    // Script Manager Access
    script management() interface{}     // Bridge to script manager
    
    // Connection Status
    connection status management() ConnectionStatus
    
    // Lifecycle
    Shutdown() error
}
```

#### 5.2 Update ProxyClient Wrapper Methods

Add missing methods to `internal/tui/api/proxy_client.go`:

```go
// ADD new methods to ProxyClient
func (pc *ProxyClient) SendData(command string) error {
    if pc.currentAPI == nil {
        return errors.New("not connected")
    }
    return pc.currentAPI.SendData(command)
}

// script management removed - will be added in Phase 3

// connection status management removed - status tracked via callbacks
```

#### 5.3 Update TUI Command Sending

Update `internal/tui/app.go` sendCommand method:

```go
// UPDATE sendCommand method to use clean API
func (ta *TwistApp) sendCommand(command string) {
    if ta.proxyClient.IsConnected() {
        ta.proxyClient.SendData([]byte(command))  // Convert string to []byte for API
    }
}
```

### Step 3: Update Status Component for API-Based Script Manager

#### 3.1 Update Status Component Initialization

The status component needs to get the script manager through the API when connected:

```go
// FIND this line in NewApplication():
// statusComp.SetScriptManager(proxyInstance.script management())

// REPLACE with delayed initialization:
// We'll set the script manager after connection is established

// Remove this line for now - status component will get script manager
// via connection callback in HandleConnectionStatusChanged
```

#### 3.2 Update Connection Handler to Set Script Manager

Add to `internal/tui/app.go` in the `HandleConnectionStatusChanged` method:

```go
func (ta *TwistApp) HandleConnectionStatusChanged(info proxyapi.ConnectionInfo) {
    ta.QueueUpdateDraw(func() {
        ta.connected = true
        ta.serverAddress = info.Address
        ta.menuComponent.SetConnectedMenu()
        ta.statusComponent.SetConnectionStatus(true, "Connected to "+info.Address)
        
        // Script manager setup removed - will be handled in Phase 3
    })
}
```

### Step 4: Add Missing Interface Methods and Update ProxyClient

#### 4.1 Update ProxyClient to Support New Methods

Add missing methods to `internal/tui/api/proxy_client.go`:

```go
// ADD new methods to ProxyClient
func (pc *ProxyClient) SendData(command string) error {
    if pc.currentAPI == nil {
        return errors.New("not connected")
    }
    return pc.currentAPI.SendData(command)
}

// script management removed - will be added in Phase 3

// connection status management removed - status tracked via callbacks
```

#### 4.2 Update Interface Definitions

Add new methods to `internal/proxy/api/proxy_api.go`:

```go
type ProxyAPI interface {
    // Connection Management  
    Connect(address string, tuiAPI TuiAPI) error
    Disconnect() error
    IsConnected() bool
    
    // Command Processing
    SendData(command string) error  // ADD this for clarity
    SendData(data []byte) error        // Keep existing for compatibility
    
    // Script Manager Bridge (temporary)
    script management() interface{}     // ADD this
    
    // Connection Status
    connection status management() ConnectionStatus  // ADD this
    
    // Lifecycle
    Shutdown() error
}
```

### Step 5: Update TUI to Use SendData

#### 5.1 Update Command Sending Method

In `internal/tui/app.go`, update the `sendCommand` method:

```go
// UPDATE sendCommand method to use clearer API
func (ta *TwistApp) sendCommand(command string) {
    if ta.proxyClient.IsConnected() {
        // REPLACE: ta.proxyClient.SendData([]byte(command))
        // WITH:
        ta.proxyClient.SendData(command)
    }
}
```

## Testing Phase 2

### Manual Testing Steps
1. **Startup**: Verify app starts without errors after removing TerminalWriter
2. **Connection Flow**: Test connection works through pure API layer
3. **Data Streaming**: Verify data flows directly (Pipeline → TuiAPI.OnData() → TUI)
4. **Script Status**: Verify script manager status display works after connection
5. **Data Sending**: Test SendData method works for user input (convert strings to []byte)
6. **Multiple Connections**: Test connecting/disconnecting multiple times
7. **Interface Elimination**: Verify TerminalWriter interface is completely gone

### Key Validation Points
- [ ] **TerminalWriter interface removed** completely from codebase
- [ ] **No direct proxy field** in TwistApp struct  
- [ ] **Pure API communication** - Pipeline calls TuiAPI.OnData() directly
- [ ] **Script manager** accessed via API after connection established
- [ ] **API methods complete** - SendData, script management, connection status management implemented
- [ ] **No bridge pattern** - direct TuiAPI calls throughout
- [ ] **Clean proxy constructor** - takes TuiAPI parameter only

### Expected Behavior
- TUI creates no direct proxy instances
- Pipeline calls TuiAPI.OnData() directly (no intermediate interfaces)
- Script manager status updates after successful connection via API
- Data flows: Game Server → Pipeline → TuiAPI.OnData() → TUI (no bridges)
- All user commands sent via SendData API method (strings converted to []byte)
- Proxy constructor signature: `func New(tuiAPI api.TuiAPI) *Proxy`

## Success Criteria

✅ **TerminalWriter Eliminated**: Interface completely removed from codebase  
✅ **Direct Proxy Removed**: TUI has no direct proxy field or instance creation  
✅ **Pure API Communication**: Pipeline → TuiAPI.OnData() with no intermediate interfaces  
✅ **Clean API Methods**: SendData implemented with simplified callbacks  
✅ **Clean Proxy Constructor**: Takes only TuiAPI parameter  
✅ **No Coupling Interfaces**: Zero interfaces between proxy and TUI except TuiAPI  
✅ **Performance Maintained**: Direct calls more efficient than bridge pattern  

## Files to Modify Summary

### Critical Changes Required:
- `internal/streaming/pipeline.go` - **REMOVE TerminalWriter interface**, update struct and constructors, replace `terminalWriter.Write()` with `tuiAPI.OnData()`
- `internal/proxy/proxy.go` - **UPDATE constructor signature** to `func New(tuiAPI api.TuiAPI)`, add tuiAPI field
- `internal/proxy/api/proxy_api_impl.go` - **REMOVE Write() method**, **REMOVE Shutdown() method**, **REMOVE Connect() method**, update static Connect() function
- `internal/tui/app.go` - **REMOVE direct proxy field**, remove proxy creation, update script manager initialization
- `internal/proxy/api/proxy_api.go` - Remove Connect() method, Add SendData method only (script management in Phase 3)
- `internal/tui/api/proxy_client.go` - Add wrapper methods for new API functionality

### Key Architectural Changes:
1. **TerminalWriter interface elimination** - completely removed from codebase
2. **Direct TuiAPI integration** - Pipeline calls TuiAPI.OnData() directly  
3. **Clean proxy constructor** - takes only TuiAPI parameter
4. **Pure API communication** - no intermediate coupling interfaces
5. **Static connection model** - only `api.Connect()` function, no instance Connect() method
6. **Single pipeline constructor** - script manager complexity deferred to Phase 3
7. **Non-blocking connection** - all connection state via callbacks, no blocking calls

## Implementation Notes

### Critical Implementation Points:

1. **Complete TerminalWriter Removal**: 
   - Remove interface definition from `internal/streaming/pipeline.go`
   - Update Pipeline struct to have `tuiAPI api.TuiAPI` field instead
   - Update both pipeline constructors to take TuiAPI parameter
   - Replace `p.terminalWriter.Write(decoded)` with `p.tuiAPI.OnData(decoded)`

2. **Proxy Constructor Migration**:
   - Change signature from `func New(terminalWriter streaming.TerminalWriter)` to `func New(tuiAPI api.TuiAPI)`
   - Add `tuiAPI api.TuiAPI` field to Proxy struct
   - Pass TuiAPI to pipeline constructor instead of TerminalWriter

3. **ProxyApiImpl Cleanup**:
   - Remove `Write(data []byte)` method completely
   - Update `Connect()` function to call `proxy.New(tuiAPI)` directly
   - No more bridge pattern - direct API communication

4. **Import Updates**: Add `"twist/internal/proxy/api"` imports where needed

### What MUST be eliminated:

- ❌ **TerminalWriter interface** - completely removed
- ❌ **ProxyApiImpl.Write() method** - bridge pattern eliminated  
- ❌ **Direct proxy field in TUI** - API-only communication
- ❌ **proxy.New(nil) calls** - constructor now requires TuiAPI

### Clean Architecture Verification:

```bash
# Verify TerminalWriter is completely eliminated
grep -r "TerminalWriter" internal/
# Should return NO results

# Verify no direct proxy in TUI
grep -n "proxy \*proxy.Proxy" internal/tui/app.go
# Should return NO results

# Verify clean proxy constructor
grep -n "func New.*TuiAPI" internal/proxy/proxy.go
# Should find the new signature

# Verify direct TuiAPI calls in pipeline
grep -n "tuiAPI.OnData" internal/streaming/pipeline.go
# Should find the direct call
```

## Risk Mitigation

1. **Breaking Changes Accepted**: No backwards compatibility needed - clean separation is the goal
2. **Systematic Approach**: Update pipeline → proxy → API → TUI in sequence  
3. **Import Management**: Careful handling of circular import prevention
4. **Testing at Each Step**: Verify compilation after each major component update
5. **Rollback Plan**: Keep Phase 1 branch as fallback if issues arise

## Next Steps (Phase 3)

Phase 3 will add proper script management API methods:
- Add script lifecycle events (OnScriptLoaded, OnScriptStopped, OnScriptError) to TuiAPI
- Add script management methods (LoadScript, StopScript, GetScriptStatus) to ProxyAPI  
- Remove temporary script management() bridge method
- Implement full script variable management through API

This phase establishes **pure API-only communication** with zero coupling interfaces between proxy and TUI.

## Final Implementation Notes for Agent

### Critical Design Decisions Made:
1. **Static Connect Pattern**: `proxyapi.Connect()` function never returns errors - signature is `func Connect(address string, tuiAPI TuiAPI) ProxyAPI`
2. **No Immediate Failures**: Even validation errors (empty address, nil tuiAPI) handled via async callbacks
3. **Method Naming**: Use `HandleConnectionStatusChanged()` in TUI for consistency with callback name
4. **Type Cleanup**: Remove `ConnectionInfo` struct completely - not used in simplified callback interface  
5. **ProxyClient.Connect()**: Can keep same name since it's just a passthrough wrapper
6. **Import Safety**: Both sides import `internal/proxy/api` - watch for circular imports during implementation

### Error Handling Strategy:
- **No synchronous errors**: `api.Connect()` always succeeds and returns ProxyAPI instance
- **All failures async**: Connection, validation, DNS errors all go via `OnConnectionError()` callback
- **Graceful degradation**: Invalid parameters create safe-to-use but non-functional ProxyAPI instances

### Implementation Order Recommendation:
1. Update interface definitions first (`tui_api.go`, `proxy_api.go`)
2. Update streaming pipeline (remove TerminalWriter completely)  
3. Update proxy constructor and Connect logic
4. Update TUI callback handling (consolidate 3 handlers into 1)
5. Clean up unused types and methods
6. Test connection flow end-to-end

This ensures compilation at each step and minimal debugging of integration issues.