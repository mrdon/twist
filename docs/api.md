# Proxy-TUI API Architecture

## Project Overview

This document defines the high-level architecture and design principles for the API separation between the proxy and TUI modules in the TWIST application. This serves as the master architectural reference for all implementation phases.

**Goal**: Eliminate direct coupling between proxy and TUI modules through clean API interfaces that mirror proven architecture patterns from TWX (TradeWars eXtended).

## Architectural Principles

### Core Design Goals
1. **Zero Direct Coupling**: TUI must never import or access proxy internals directly
2. **Event-Driven Communication**: All proxy→TUI communication via callbacks
3. **Non-Blocking Operations**: All API methods return immediately using async patterns
4. **Thin Orchestration**: API implementations delegate to specialized business logic modules
5. **Performance Critical**: Support high-frequency data streaming without bottlenecks

### API Design Patterns
- **Static Connection Model**: `api.Connect()` function creates connected API instances
- **Callback-Based Events**: Status changes reported via TuiAPI callbacks, not polling
- **Symmetric Data Flow**: `SendData()` and `OnData()` for bidirectional communication
- **Fire-and-Forget**: Long operations report results via callbacks, not return values
- **Channel-Based Processing**: High-frequency callbacks use buffered channels

## Module Architecture

### Current Module Structure (Implemented)
```
internal/
├── api/                  # Core interface definitions
│   ├── api.go           # ProxyAPI, TuiAPI, shared types
│   └── connect.go       # api.Connect() function
├── proxy/               # Complete proxy package
│   ├── proxy.go         # Core proxy (accepts TuiAPI in constructor)
│   ├── proxy_api_impl.go # ProxyAPI implementation
│   ├── game_state_converters.go # API data converters
│   ├── database/        # Database management (moved from internal/)
│   ├── streaming/       # Data streaming (moved from internal/)
│   └── scripting/       # Script management (moved from internal/)
├── tui/
│   ├── api/             # TUI API integration
│   │   ├── proxy_client.go  # ProxyAPI client wrapper
│   │   └── tui_api_impl.go  # TuiAPI implementation
│   └── app.go           # Main TUI (API-only access)
└── [other UI modules]   # theme, ansi, terminal, components, etc.
```

### Data Flow Architecture (Current)
```
Game Server → Proxy.handleOutput() → Pipeline.Write() → 
Pipeline.batchProcessor() → tuiAPI.OnData() → TuiApiImpl.dataChan → 
TuiApiImpl.processDataLoop() → app.HandleTerminalData() → UI Update
```

## Core API Interfaces

### ProxyAPI - Commands from TUI to Proxy
```go
// internal/api/api.go
type ProxyAPI interface {
	// Connection Management
	Disconnect() error
	IsConnected() bool
	
	// Data Processing (symmetric with OnData)  
	SendData(data []byte) error
}
```

**Implementation**: `internal/proxy/proxy_api_impl.go`
- **Static Connection**: Created via `api.Connect(address, tuiAPI)` function
- **Async Operations**: All methods return immediately, use callbacks for results
- **Connection Lifecycle**: One ProxyAPI instance per connection

#### TuiAPI - Notifications from Proxy to TUI
```go
// internal/api/api.go
type TuiAPI interface {
	// Connection Events - single callback for all status changes
	OnConnectionStatusChanged(status ConnectionStatus, address string)
	OnConnectionError(err error)

	// Data Events - must return immediately (high frequency calls)
	OnData(data []byte)
}
```

**Implementation**: `internal/tui/api/tui_api_impl.go`
- **Channel-based Processing**: `OnData()` uses buffered channels for high-frequency calls
- **Async UI Updates**: All methods use goroutines and `QueueUpdateDraw()`
- **Performance Critical**: Methods return within microseconds

#### Connection Status Types
```go
type ConnectionStatus int

const (
	ConnectionStatusDisconnected ConnectionStatus = iota
	ConnectionStatusConnecting
	ConnectionStatusConnected
)
```

## Implementation Architecture (Current)

### Connection Flow
1. **TUI initiates connection**: `proxyClient.Connect(address, tuiAPI)`  
2. **ProxyClient calls static function**: `api.Connect(address, tuiAPI)` returns `ProxyAPI`
3. **Proxy creates instance**: `proxy.New(tuiAPI)` with direct TuiAPI reference
4. **Pipeline integration**: `streaming.NewPipelineWithScriptManager(tuiAPI, db, scriptManager)`
5. **Async connection attempt**: Connection runs in goroutine, status via callbacks
6. **Data streaming**: `Pipeline` → `tuiAPI.OnData()` → `TuiApiImpl` → channel processing → TUI

### Module Structure (Current Implementation)
```
internal/
├── api/                  # Core interface definitions
│   ├── api.go           # ProxyAPI, TuiAPI, ConnectionStatus
│   └── connect.go       # api.Connect() function
├── proxy/               # Complete proxy package
│   ├── proxy.go         # Core proxy (takes TuiAPI in constructor)
│   ├── proxy_api_impl.go # ProxyAPI implementation
│   ├── game_state_converters.go # API data converters
│   ├── database/        # Database management (moved)
│   ├── streaming/       # Data streaming (moved)
│   │   └── pipeline.go  # Calls tuiAPI.OnData() directly
│   └── scripting/       # Script management (moved)
└── tui/
    ├── api/             # TUI API integration
    │   ├── proxy_client.go  # ProxyAPI client wrapper
    │   └── tui_api_impl.go  # TuiAPI implementation with channels
    └── app.go           # Main TUI (uses only API, no direct proxy)
```

### Data Flow (Current Implementation)
```
Game Server → Proxy.handleOutput() → Pipeline.Write() → 
Pipeline.batchProcessor() → tuiAPI.OnData() → TuiApiImpl.dataChan → 
TuiApiImpl.processDataLoop() → app.HandleTerminalData() → 
TerminalComponent.Write() → UI Update
```

## API Extensions (Phases 3-4 Implemented)

The API has been extended with additional functionality through Phases 3-4:

### Current ProxyAPI Methods (Implemented)
```go
type ProxyAPI interface {
	// Connection Management (Phases 1-2)
	Disconnect() error
	IsConnected() bool
	SendData(data []byte) error
	
	// Phase 3: Script Management (implemented)
	LoadScript(filename string) error
	StopAllScripts() error
	GetScriptStatus() ScriptStatusInfo
	
	// Phase 4: Game State Access (implemented)
	GetCurrentSector() (int, error)
	GetSectorInfo(sectorNum int) (SectorInfo, error)
	GetPlayerInfo() (PlayerInfo, error)
}
```

### Current TuiAPI Methods (Implemented)
```go
type TuiAPI interface {
	// Connection & Data Events (Phases 1-2)
	OnConnectionStatusChanged(status ConnectionStatus, address string)
	OnConnectionError(err error)
	OnData(data []byte)
	
	// Phase 3: Script Events (implemented)
	OnScriptStatusChanged(status ScriptStatusInfo)
	OnScriptError(scriptName string, err error)
	
	// Phase 4: Game State Events (implemented)
	OnCurrentSectorChanged(sectorNumber int)
}
```

### Current Data Types (Implemented)
```go
// Phase 3: Script Management types (implemented)
type ScriptStatusInfo struct {
    ActiveCount int      `json:"active_count"`  // Number of running scripts
    TotalCount  int      `json:"total_count"`   // Total number of loaded scripts  
    ScriptNames []string `json:"script_names"`  // Names of loaded scripts
}

// Phase 4: Game State types (implemented)
type PlayerInfo struct {
    Name          string `json:"name"`           // Player name (if available)
    CurrentSector int    `json:"current_sector"` // Current sector location
}

type SectorInfo struct {
    Number        int    `json:"number"`         // Sector number
    NavHaz        int    `json:"nav_haz"`        // Navigation hazard level  
    HasTraders    int    `json:"has_traders"`    // Number of traders present
    Constellation string `json:"constellation"`  // Constellation name
    Beacon        string `json:"beacon"`         // Beacon text
}
```

## Complete API Design

### ProxyAPI - Full Interface Design
**Note**: This shows the complete architectural design. Each phase implements specific methods incrementally.

```go
type ProxyAPI interface {
    // Connection Management
    Disconnect() error
    IsConnected() bool
    SendData(data []byte) error
    
    // Basic Script Management
    LoadScript(filename string) error
    StopAllScripts() error
    GetScriptStatus() ScriptStatusInfo
    
    // Enhanced Script Management
    ExecuteScriptCommand(command string) error
    StopSpecificScript(scriptID string) error
    ListAllScripts() []ScriptInfo
    ListRunningScripts() []ScriptInfo
    GetScriptRuntime(scriptID string) (time.Duration, error)
    GetScriptVariables(scriptID string) (map[string]interface{}, error)
    SetScriptVariable(scriptID, name string, value interface{}) error
    
    // Database Integration
    SaveScriptVariable(name string, value interface{}) error
    LoadScriptVariable(name string) (interface{}, error)
    GetSectorFromDB(sectorNum int) (SectorInfo, error)
    SetSectorParameter(sector int, name, value string) error
    GetSectorParameter(sector int, name string) (string, error)
    
    // Game State Access
    GetGameState() (GameStateInfo, error)
    GetConnectionInfo() (ConnectionInfo, error)
    GetCurrentSector() (int, error)
    GetCurrentPrompt() (string, error)
    GetLastServerOutput() (string, error)
    GetTerminalBuffer() ([]string, error)
    GetSectorData(sectorNum int) (SectorInfo, error)
    GetCurrentPlayerInfo() (PlayerInfo, error)
    GetCurrentShipInfo() (ShipInfo, error)
    
    // Advanced Features (TWX-inspired)
    // Bot Management System
    SwitchBot(botName string) error
    GetActiveBotName() (string, error)
    GetActiveBotDirectory() (string, error)
    
    // Advanced Variable System
    SetGlobalVariable(name string, value interface{}) error
    GetGlobalVariable(name string) (interface{}, error)
    ListGlobalVariables() ([]VariableInfo, error)
    
    // Timer System
    CreateTimer(name string, interval time.Duration) error
    DeleteTimer(name string) error
    ListActiveTimers() ([]TimerInfo, error)
    
    // Event System
    TriggerProgramEvent(eventName, matchText string, exclusive bool) error
    RegisterEventHandler(eventName string, handler string) error
    UnregisterEventHandler(eventName string) error
}
```

### TuiAPI - Full Interface Design
**Note**: This shows the complete architectural design. Each phase implements specific methods incrementally.

```go
type TuiAPI interface {
    // Connection & Data Events
    OnConnectionStatusChanged(status ConnectionStatus, address string)  
    OnConnectionError(err error)
    OnData(data []byte)
    
    // Basic Script Events
    OnScriptStatusChanged(status ScriptStatusInfo)
    OnScriptError(scriptName string, err error)
    
    // Enhanced Script Events
    OnScriptLoaded(script ScriptInfo)
    OnScriptStarted(script ScriptInfo)
    OnScriptStopped(script ScriptInfo, reason string)
    OnScriptText(text string)
    OnScriptVariableChanged(scriptID, name string, value interface{})
    
    // Game State Events
    OnGameStateChanged(state GameStateInfo)
    OnCurrentSectorChanged(sector SectorInfo)
    OnPlayerMove(fromSector, toSector int)
    OnSectorScan(sector SectorInfo)
    OnPlayerInfoChanged(playerInfo PlayerInfo)
    OnCreditsChange(oldCredits, newCredits int)
    OnTurnsChange(oldTurns, newTurns int)
    
    // Advanced Events (TWX-inspired)
    // Bot Management Events
    OnBotSwitched(oldBot, newBot string)
    OnBotError(botName string, err error)
    
    // Timer Events  
    OnTimerFired(timerName string)
    OnTimerCreated(timer TimerInfo)
    
    // Variable Events
    OnGlobalVariableChanged(name string, oldValue, newValue interface{})
    OnSectorVariableChanged(sectorNum int, name string, value interface{})
    
    // System Events
    OnError(err error)
    OnStatusUpdate(message string)
}
```

## Implementation Guidelines

### Phase-Based Development
Each phase incrementally adds API methods while maintaining backward compatibility:
- **Phase 1-2**: Connection management, data streaming
- **Phase 3**: Basic script management
- **Phase 4**: Game state tracking
- **Phase 5+**: Advanced features, TWX compatibility

### Critical Implementation Requirements

#### Performance Requirements
- **TuiAPI Methods**: Must return within microseconds using goroutines for work
- **ProxyAPI Methods**: Return immediately, use callbacks for async results
- **High-Frequency Calls**: `OnData()` may be called hundreds of times per second
- **Channel Processing**: Use buffered channels for high-frequency data

#### Architecture Requirements
- **Zero Direct Coupling**: TUI must never import proxy internals
- **API-Only Communication**: All interaction through ProxyAPI/TuiAPI interfaces
- **Thin Orchestration**: API implementations delegate to business logic modules
- **Static Connection Pattern**: Use `api.Connect()` function, not instance methods

### Agent Implementation References

Each implementation phase has detailed instructions:
- `docs/api-phase-3.md` - Script Management API implementation
- `docs/api-phase-2.md` - Connection management and data streaming (reference)
- Future phases will have their own detailed implementation guides

### 2. Direct Method Call Architecture

The proxy calls TuiAPI methods directly - no event bus needed in the proxy. The TUI may optionally use an internal event bus for coordinating its own components, but this is internal to the TUI module.

#### Critical Performance Requirement: Non-Blocking API Methods

**All API methods in both interfaces MUST return immediately** to maintain system performance and stability:

##### TuiAPI Methods (Proxy → TUI):
- **MUST return within microseconds** - proxy processes network data in real-time
- **Use goroutines** for any actual work (UI updates, complex processing)
- **Queue UI updates** through tview's `QueueUpdateDraw()` mechanism
- **Never block** the calling proxy thread
- **High frequency calls**: `OnData()` may be called hundreds of times per second

##### ProxyAPI Methods (TUI → Proxy):
- **MUST return immediately** for user-initiated actions
- **Use goroutines** for operations that might take time (network operations, file I/O)
- **Return errors immediately** for invalid parameters, but not for async operation failures
- **Fire-and-forget pattern**: Long operations report results via TuiAPI callbacks

##### Why This Matters:
1. **Network Stability**: Proxy must process game server data without interruption
2. **UI Responsiveness**: User interactions must feel instant
3. **System Performance**: Blocking calls can cause cascading delays
4. **Concurrent Safety**: Goroutines prevent race conditions and deadlocks

##### Implementation Pattern:
```go
// TuiAPI - Return immediately, do work async
func (tui *TuiApiImpl) OnData(data []byte) {
    go func() {
        tui.app.QueueUpdateDraw(func() {
            // UI work happens here
        })
    }()
    // Returns immediately
}

// ProxyAPI - Return immediately, do work async  
func (p *ProxyApiImpl) Connect(address string, tuiAPI TuiAPI) error {
    // Validate parameters immediately
    if address == "" {
        return errors.New("address required")
    }
    
    // Do connection work async
    go func() {
        err := p.proxy.Connect(address)
        if err != nil {
            tuiAPI.OnConnectionError(err)
        } else {
            tuiAPI.OnConnected(ConnectionInfo{...})
        }
    }()
    
    return nil // Returns immediately
}
```

#### Thin Orchestration Layer Architecture

Both ProxyAPI and TuiAPI implementations **MUST** follow the thin orchestration layer pattern:

**All API methods should be one-liners** that delegate to specialized modules:
- **ProxyAPI methods**: Delegate to `ConnectionManager`, `DataManager`, `ScriptManager`, etc.
- **TuiAPI methods**: Delegate to `EventHandler`, `UIManager`, `StateManager`, etc.
- **Keep business logic** in dedicated, focused modules - not in API implementations
- **API classes are super thin** - just routing/orchestration with minimal logic

##### Example Pattern:
```go
// ProxyAPI - thin orchestration
func (p *ProxyApiImpl) Connect(address string, tuiAPI TuiAPI) error {
    return p.connectionManager.ConnectAsync(address, tuiAPI) // One-liner delegate
}

// TuiAPI - thin orchestration  
func (tui *TuiApiImpl) OnData(data []byte) {
    go tui.uiManager.ProcessTerminalData(data) // One-liner delegate (async)
}
```

This pattern ensures:
- **Clean separation** between API routing and business logic
- **Easy testing** of individual managers/handlers
- **Maintainable code** with focused responsibilities
- **Flexible architecture** for future enhancements

#### Proxy → TUI Communication (Simple Direct Calls)
```go
// internal/proxy/api/proxy_api.go
type ProxyApiImpl struct {
    core   *core.Proxy
    tuiAPI TuiAPI  // Single TUI instance per connection
}

func (p *ProxyApiImpl) Connect(address string, tuiAPI TuiAPI) error {
    p.tuiAPI = tuiAPI  // Store reference
    
    if err := p.core.connect(address); err != nil {
        tuiAPI.OnConnectionError(err)  // Direct call
        return err
    }
    
    connectionInfo := ConnectionInfo{
        Address:     address,
        ConnectedAt: time.Now(),
        Status:      "connected",
    }
    tuiAPI.OnConnected(connectionInfo)  // Direct call
    return nil
}

// When streaming pipeline receives data:
func (p *ProxyApiImpl) handleData(data []byte) {
    if p.tuiAPI != nil {
        p.tuiAPI.OnData(data)  // Simple direct call - returns immediately
    }
}
```

#### TUI Implementation (Must Return Immediately)
```go
// internal/tui/api/tui_api_impl.go
type TuiApiImpl struct {
    app      *TwistApp
    eventBus *EventBus  // Optional internal TUI event bus
}

func (tui *TuiApiImpl) OnData(data []byte) {
    // Return immediately by using goroutine
    go func() {
        tui.app.QueueUpdateDraw(func() {
            tui.app.updateTerminalView()
        })
    }()
    // Method returns immediately - proxy not blocked
}

func (tui *TuiApiImpl) OnCurrentSectorChanged(sector SectorInfo) {
    // Return immediately - complex UI work happens async
    go func() {
        tui.app.QueueUpdateDraw(func() {
            tui.app.updateSectorDisplay(sector)
            // Even if this triggers modal dialogs or complex interactions,
            // proxy continues processing immediately
        })
    }()
}
```

### 3. Implementation Architecture

#### Proxy Module Structure
```
internal/
├── proxy/
│   ├── api/              # New API module
│   │   ├── proxy_api.go     # ProxyAPI implementation
│   │   ├── tui_api.go       # TuiAPI interface definition
│   │   ├── types.go         # API data structures (Api* types)
│   │   └── converters.go    # Internal → API data conversion
│   ├── core/             # Renamed from proxy.go
│   │   ├── proxy.go         # Core proxy logic (internal)
│   │   ├── connection.go    # Connection management
│   │   └── state_manager.go # Internal state management
│   ├── streaming/        # Move from internal/streaming
│   ├── scripting/        # Move from internal/scripting  
│   └── database/         # Move from internal/database
```

#### TUI Module Structure
```
internal/
├── tui/
│   ├── api/              # New API integration
│   │   ├── tui_api_impl.go  # TuiAPI implementation
│   │   └── proxy_client.go  # ProxyAPI client wrapper
│   ├── app.go            # Updated main TUI app (no direct proxy access)
│   ├── components/       # UI components (use API data only)
│   └── handlers/         # Input handlers (use ProxyAPI only)
```

#### Module Import Restrictions (Enforced)
```go
// TUI Module - ONLY imports API
// internal/tui/app.go
import (
    "twist/internal/api"              // ✅ Only API types and interfaces
    // FORBIDDEN imports:
    // - twist/internal/proxy         ❌ No proxy internals
    // - twist/internal/proxy/database ❌ No direct database access
    // - twist/internal/proxy/streaming ❌ No streaming internals  
    // - twist/internal/proxy/scripting ❌ No scripting internals
)

// Proxy Module - Can import its internals
// internal/proxy/proxy_api_impl.go
import (
    "twist/internal/api"                     // ✅ Core API interfaces
    "twist/internal/proxy/database"          // ✅ Internal data access
    "twist/internal/proxy/streaming"         // ✅ Internal streaming
    "twist/internal/proxy/scripting"         // ✅ Internal scripting
    // Converts internal data to API types
)
```

## Implementation Status

**🎉 ALL PHASES COMPLETED - APPLICATION WORKING**

All 5 phases of the proxy-TUI API separation have been successfully implemented:

- ✅ **Phase 1-2**: Connection management and data streaming - **COMPLETED**
- ✅ **Phase 3**: Script management API - **COMPLETED**  
- ✅ **Phase 4**: Game state tracking - **COMPLETED**
- ✅ **Phase 5**: Module cleanup and separation - **COMPLETED**

The application is now fully functional with clean architectural separation.

## Implementation History

### Phase 1: Connection Management Foundation (COMPLETED)
**Goal**: Establish API infrastructure and implement connection/data streaming functionality.

**Scope**: Connection management only - Connect, Disconnect, SendData, and data streaming through OnData().

#### Key Deliverables:
- **ProxyAPI Interface**: `Connect()`, `Disconnect()`, `IsConnected()`, `SendData()`, `Shutdown()`
- **TuiAPI Interface**: `OnConnected()`, `OnDisconnected()`, `OnConnectionError()`, `OnData()`  
- **Data Types**: `ConnectionInfo` struct
- **Streaming Integration**: Replace `TerminalWriter` with `TuiAPI.OnData()` calls
- **Dual Path Support**: API layer works alongside existing proxy during transition

#### Architecture:
- Create API module structure (`internal/proxy/api/`, `internal/tui/api/`)
- Implement ProxyAPI as wrapper around existing proxy
- Implement TuiAPI with goroutine-based async UI updates
- Update streaming pipeline to call TuiAPI instead of direct terminal writes
- Add API layer to TUI app without breaking existing functionality

**Success Criteria**: Connect to game server and stream data through API layer while maintaining all existing functionality.

**Detailed implementation steps**: See `docs/api-phase-1.md`

### Phase 2: Connection Management Migration
**Goal**: Migrate connection functionality to use API exclusively.

**Approach**: Add only connection-related methods to the interfaces. No other methods exist yet.

#### Methods to Add to Interfaces:
**Add to ProxyAPI**: `Connect(address string, tuiAPI TuiAPI) error`, `Disconnect() error`, `IsConnected() bool`, `SendCommand(command string) error`  
**Add to TuiAPI**: `OnConnected(info ConnectionInfo)`, `OnDisconnected(reason string)`, `OnConnectionError(err error)`, `OnData(data []byte)`  
**Add to types.go**: `ConnectionInfo` struct only

#### Connection Proxy-Side (First):
1. **Extend ProxyAPI Interface and Implementation**
   - Add connection methods to `ProxyAPI` interface
   - Implement these methods in `ProxyApiImpl` as wrappers around existing proxy
   - Make proxy call `TuiAPI` methods directly when connection events happen
   - Add `ConnectionInfo` type and basic converter

2. **Update Streaming Pipeline**
   - Replace direct terminal writer injection with `TuiAPI.OnData()` calls
   - **Remove `streaming.TerminalWriter` interface completely** - this interface creates tight coupling and should be eliminated entirely
   - Update pipeline constructor to accept `TuiAPI` parameter instead of `TerminalWriter`
   - Replace `p.terminalWriter.Write(decoded)` with direct `p.tuiAPI.OnData(decoded)` calls
   - Handle scripting integration dependency on `TerminalInterface` (scripts need `GetLines()` access)

#### Connection TUI-Side (Second):
3. **Update TUI Connection Handling**
   - Add connection methods to `TuiAPI` interface
   - Implement these methods in `TuiApiImpl` with real functionality
   - Replace `ta.proxy.Connect()` with `proxyAPI.Connect()`
   - Remove TUI-side connection state (`ta.connected`, `ta.serverAddress`)

4. **Update Terminal Component**
   - Remove direct terminal callback mechanism
   - Receive data via `TuiAPI.OnData()` instead of direct writes
   - Keep existing terminal buffer handling (no API data structures needed yet)

**Files to Focus On**:
- `internal/proxy/proxy.go` (update constructor to accept TuiAPI, handle scripting integration)
- `internal/streaming/pipeline.go` (**remove TerminalWriter interface**, update constructors, replace Write calls)
- `internal/tui/app.go` (connection methods, remove connection state)
- `internal/tui/components/terminal.go` (receive data via API)

**Critical TerminalWriter Removal Tasks**:
- Remove `TerminalWriter` interface definition from `pipeline.go`
- Update `NewPipeline()` and `NewPipelineWithScriptManager()` signatures
- Replace `p.terminalWriter.Write(decoded)` with `p.tuiAPI.OnData(decoded)`
- Solve scripting integration: either extend `TuiAPI` with `GetLines()` or pass separate terminal reference
- Remove `GetTerminalWriter()` method (unused)

### Phase 3: Script Management Migration  
**Goal**: Migrate script functionality to use API exclusively.

**Approach**: Add only basic script management methods to the interfaces. Keep advanced features for later.

#### Methods to Add to Interfaces:
**Add to ProxyAPI**: `LoadScript(filename string) error`, `ExecuteScriptCommand(command string) error`, `StopAllScripts() error`, `GetScriptStatus() (ScriptStatusInfo, error)`  
**Add to TuiAPI**: `OnScriptLoaded(script ScriptInfo)`, `OnScriptStopped(script ScriptInfo, reason string)`, `OnScriptError(script ScriptInfo, err error)`, `OnScriptText(text string)`  
**Add to types.go**: `ScriptInfo`, `ScriptStatusInfo` (basic versions only)

#### Script Proxy-Side (First):
1. **Extend API Interfaces for Scripts**
   - Add script methods to `ProxyAPI` interface  
   - Add script types to `types.go` (basic `ScriptInfo` and `ScriptStatusInfo`)
   - Implement script methods in `ProxyApiImpl` as wrappers around existing script manager
   - Add basic converters for internal script data → API types

2. **Update Script Manager Integration**
   - Make script manager call `TuiAPI` methods directly when scripts load/stop/error
   - Remove direct proxy/terminal dependencies from scripting
   - Centralize script status tracking in proxy

#### Script TUI-Side (Second):
3. **Update TUI Script Handling**
   - Add script methods to `TuiAPI` interface
   - Implement these methods in `TuiApiImpl` with real functionality
   - Replace `ta.proxy.GetScriptManager()` with `proxyAPI.GetScriptStatus()`
   - Remove direct script manager access from status component

4. **Update Status Component**
   - Use TuiAPI callbacks instead of direct script manager access
   - Get script data via ProxyAPI calls only
   - Update to use basic API data structures (ScriptInfo, ScriptStatusInfo)

**Files to Focus On**:
- `internal/scripting/integration.go` (call TuiAPI methods directly)
- `internal/scripting/manager/manager.go` (call TuiAPI methods for lifecycle)
- `internal/tui/components/status.go` (use ProxyAPI and TuiAPI only)
- `internal/tui/app.go` (remove GetScriptManager usage)

### Phase 4: Game State Migration
**Goal**: Migrate game state and data access to use API exclusively.

**Approach**: Add only basic game state methods to the interfaces. Skip advanced tracking and analytics.

#### Methods to Add to Interfaces:
**Add to ProxyAPI**: `GetGameState() (GameStateInfo, error)`, `GetCurrentSector() (int, error)`, `GetCurrentPlayerInfo() (PlayerInfo, error)`  
**Add to TuiAPI**: `OnGameStateChanged(state GameStateInfo)`, `OnCurrentSectorChanged(sector SectorInfo)`, `OnPlayerInfoChanged(playerInfo PlayerInfo)`  
**Add to types.go**: `GameStateInfo`, `PlayerInfo`, `SectorInfo` (basic versions only)

#### Tasks:
1. **Extend API Interfaces for Game State**
   - Add game state methods to `ProxyAPI` interface
   - Add game state types to `types.go` (basic versions)
   - Add basic converters for internal game data → API types
   - Create `StateManager` for centralized game/connection/player state

2. **Update Game State Tracking**
   - Implement game state methods in `ProxyApiImpl` using new `StateManager`
   - Move basic state tracking from TUI to proxy
   - Make data parsing trigger `TuiAPI` methods directly when game state changes

3. **Update TUI State Access**
   - Add game state methods to `TuiAPI` interface  
   - Implement these methods in `TuiApiImpl` with real functionality
   - Replace local state management with ProxyAPI calls
   - Update UI components to use basic API data structures only

**Files to Focus On**:
- `internal/proxy/core/state_manager.go` (new file - centralized state)
- `internal/streaming/parser/` (call TuiAPI methods when parsing game data)
- `internal/tui/components/panels.go` (use ProxyAPI for state queries)
- All UI components (use API data structures only)

### Phase 5: Module Cleanup and Separation (COMPLETED)
**Goal**: Complete architectural separation and clean up legacy code.

#### Completed Tasks:
1. **✅ Moved Modules to Proxy Package**
   - ✅ Moved `internal/streaming` to `internal/proxy/streaming`
   - ✅ Moved `internal/scripting` to `internal/proxy/scripting`
   - ✅ Moved `internal/database` to `internal/proxy/database`
   - ✅ Updated import paths throughout codebase (63 imports across 39 files)

2. **✅ Removed Legacy Coupling**
   - ✅ Moved `Connect()` function from proxy package to API package
   - ✅ Added blank import in main.go to ensure proxy initialization
   - ✅ Enforced import restrictions with architecture tests
   - ✅ TUI now only imports `internal/api`

3. **✅ Testing and Verification**
   - ✅ Created architecture tests to enforce import restrictions
   - ✅ Verified all tests pass (unit and integration)
   - ✅ Confirmed application works correctly
   - ✅ No direct coupling remains between modules

### Future Phase 6: Advanced Features (Optional Enhancement)
**Goal**: Add advanced API features inspired by TWX architecture.

#### Potential Future Tasks:
1. **Plugin System**
   - Add plugin API for extending functionality
   - Implement plugin lifecycle management
   - Support for custom event handlers

2. **Remote API Support**
   - Add network API support (REST/WebSocket)
   - Enable remote TUI connections
   - Implement API authentication

3. **Advanced Scripting Integration**
   - Expose full API to scripting system
   - Add script-based event handlers
   - Implement custom trigger types

## Success Criteria ✅ ACHIEVED

1. **✅ Separation of Concerns**: TUI has zero direct access to proxy internals
2. **✅ API-Driven**: All communication uses direct TuiAPI calls or ProxyAPI methods
3. **✅ State Management**: Centralized state with read-only API access patterns
4. **✅ Data Isolation**: TUI only sees API data structures, never internal objects
5. **✅ Non-Blocking**: All TuiAPI methods return immediately via goroutines
6. **✅ Testability**: API interfaces enable comprehensive unit testing with architecture restrictions
7. **✅ Extensibility**: New features can be added without breaking changes
8. **✅ Performance**: No performance regression - application works correctly

## Risk Mitigation

1. **Incremental Implementation**: Each phase maintains functionality
2. **Extensive Testing**: Integration tests verify behavior at each phase
3. **Rollback Strategy**: Each phase can be reverted independently
4. **Performance Monitoring**: Benchmark key operations throughout migration (especially goroutine overhead)
5. **Documentation**: Clear API documentation prevents misuse

## Dependencies

- No external dependencies required
- All changes use existing Go standard library and current dependencies
- Maintains compatibility with current database and scripting systems

## Timeline (Completed)

- **✅ Phase 1-2**: Connection management and data streaming - **COMPLETED**
- **✅ Phase 3**: Script management API - **COMPLETED**  
- **✅ Phase 4**: Game state tracking - **COMPLETED**
- **✅ Phase 5**: Module cleanup and separation - **COMPLETED**

**🎉 TOTAL**: All phases completed successfully with working application

## Final Architecture Summary

The proxy-TUI API separation project has been successfully completed with the following achievements:

### ✅ **Clean Architecture Established**
- **Zero Coupling**: TUI has no direct access to proxy internals
- **API Boundary**: All communication flows through well-defined ProxyAPI/TuiAPI interfaces
- **Module Organization**: Proxy internals properly organized under `internal/proxy/`

### ✅ **Working Implementation** 
- **Connection Management**: Clean connection lifecycle via `api.Connect()`
- **Data Streaming**: High-performance data flow through TuiAPI callbacks
- **Script Management**: Full scripting functionality via API
- **Game State Tracking**: Real-time game state updates via API callbacks

### ✅ **Quality Assurance**
- **Architecture Tests**: Automated enforcement of import restrictions
- **Full Test Coverage**: All unit and integration tests passing
- **Performance Maintained**: No regressions, application works correctly

The application now has a robust, maintainable architecture that supports future enhancements while maintaining clean separation of concerns.