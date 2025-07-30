# Proxy-TUI API Separation Plan

## Overview

This document outlines the design and implementation plan for creating a clean API separation between the proxy and TUI modules in the TWIST application. The goal is to eliminate direct coupling and establish clear interfaces that mirror the proven architecture patterns from TWX.

## Current Architecture Analysis

### Current TUI-Proxy Interactions Identified:

1. **Direct Proxy Access in TUI (`app.go`)**:
   - `ta.proxy.Connect(address)` - Connection management
   - `ta.proxy.Disconnect()` - Disconnection
   - `ta.proxy.SendInput(command)` - Command sending
   - `ta.proxy.GetScriptManager()` - Script manager access

2. **Terminal Writer Injection**:
   - Proxy is initialized with `terminal` as writer
   - Direct terminal buffer access from proxy via streaming pipeline
   - Terminal update callbacks trigger TUI updates

3. **Script Manager Coupling**:
   - Status component directly accesses script manager
   - Script manager needs both proxy and terminal interfaces
   - Direct method calls for script status updates

4. **State Management Issues**:
   - No centralized state management
   - TUI maintains connection state separately from proxy
   - No event system for state synchronization

## API Design

### 1. Core Interfaces

#### API Data Transfer Objects
```go
// internal/proxy/api/types.go - API-specific data structures
type ApiSectorInfo struct {
    Number      int           `json:"number"`
    Name        string        `json:"name"`
    PlayerCount int           `json:"player_count"`
    Ports       []ApiPortInfo `json:"ports,omitempty"`
}

type ApiPortInfo struct {
    Type        string `json:"type"`
    ProductType string `json:"product_type"`
    Amount      int    `json:"amount"`
}

type ApiScriptInfo struct {
    Name        string `json:"name"`
    Status      string `json:"status"`  // "running", "stopped", "error"
    Runtime     string `json:"runtime"`
    LastError   string `json:"last_error,omitempty"`
}

type ApiGameState struct {
    CurrentSector   int    `json:"current_sector"`
    CurrentTurns    int    `json:"current_turns"`
    CurrentCredits  int    `json:"current_credits"`
    PlayerName      string `json:"player_name"`
    ShipType        string `json:"ship_type"`
}

type ApiConnectionInfo struct {
    Address     string    `json:"address"`
    ConnectedAt time.Time `json:"connected_at"`
    Status      string    `json:"status"` // "connected", "connecting", "disconnected"
}

type ApiScriptStatus struct {
    ActiveCount    int             `json:"active_count"`
    TotalCount     int             `json:"total_count"`
    RunningScripts []ApiScriptInfo `json:"running_scripts"`
    LoadedScripts  []ApiScriptInfo `json:"loaded_scripts"`
}
```

#### ProxyAPI - Commands from TUI to Proxy
```go
type ProxyAPI interface {
    // Connection Management
    Connect(address string, tuiAPI TuiAPI) error
    Disconnect() error
    IsConnected() bool
    
    // Command Processing (returns immediately)
    SendCommand(command string) error
    
    // Script Management (returns immediately)
    LoadScript(filename string) error
    ExecuteScriptCommand(command string) error
    StopAllScripts() error
    GetScriptStatus() (ApiScriptStatus, error)
    
    // State Access (read-only)
    GetGameState() (ApiGameState, error)
    GetConnectionInfo() (ApiConnectionInfo, error)
    
    // Lifecycle
    Shutdown() error
}
```

#### TuiAPI - Notifications from Proxy to TUI (All methods return immediately)
```go
type TuiAPI interface {
    // Data events (async)
    OnData(data []byte)
    OnScriptText(text string)
    
    // Connection events (async)
    OnConnected(info ApiConnectionInfo)
    OnDisconnected(reason string)
    OnConnectionError(err error)
    
    // Script events (async)
    OnScriptLoaded(script ApiScriptInfo)
    OnScriptStopped(script ApiScriptInfo, reason string)
    OnScriptError(script ApiScriptInfo, err error)
    
    // Game state events (async)
    OnGameStateChanged(state ApiGameState)
    OnCurrentSectorChanged(sector ApiSectorInfo)
    
    // System events (async)
    OnError(err error)
    OnStatusUpdate(message string)
}
```

### 2. Direct Method Call Architecture

The proxy calls TuiAPI methods directly - no event bus needed in the proxy. The TUI may optionally use an internal event bus for coordinating its own components, but this is internal to the TUI module.

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
    
    connectionInfo := ApiConnectionInfo{
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

func (tui *TuiApiImpl) OnCurrentSectorChanged(sector ApiSectorInfo) {
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

#### Module Import Restrictions
```go
// TUI Module - ONLY imports API
// internal/tui/app.go
import (
    "twist/internal/proxy/api"  // Only API types and interfaces
    // FORBIDDEN imports:
    // - internal/database       ❌
    // - internal/streaming      ❌  
    // - internal/scripting      ❌
    // - internal/proxy/core     ❌
)

// Proxy Module - Can import its internals
// internal/proxy/api/proxy_api.go
import (
    "twist/internal/proxy/core"      // ✅ Internal proxy logic
    "twist/internal/database"        // ✅ Internal data access
    "twist/internal/streaming"       // ✅ Internal streaming
    // Converts internal data to API types
)
```

## Implementation Plan

### Phase 1: API Foundation (No Breaking Changes)
**Goal**: Establish API interfaces and event system without changing existing functionality.

#### Tasks:
1. **Create API Module Structure**
   - Create `internal/proxy/api/` directory
   - Define `ProxyAPI` and `TuiAPI` interfaces
   - Implement basic event system (`EventBus`, `Event`, event types)

2. **Implement TuiAPI in TUI Module**
   - Create `TuiApiImpl` struct that wraps existing TUI functionality
   - Implement all `TuiAPI` methods to call existing TUI methods
   - Wire event handlers to existing update mechanisms

3. **Add ProxyAPI Layer to Proxy**
   - Create `ProxyApiImpl` struct that wraps existing `Proxy`
   - Implement all `ProxyAPI` methods as passthroughs to existing proxy methods
   - Add event bus integration for notifications

4. **State Management Infrastructure**
   - Create `StateManager` for centralized game/connection state
   - Move state tracking from TUI to proxy
   - Implement read-only state access via API

**Files to Create**:
- `internal/proxy/api/proxy_api.go` (ProxyAPI implementation)
- `internal/proxy/api/tui_api.go` (TuiAPI interface definition)
- `internal/proxy/api/types.go` (Api* data structures)
- `internal/proxy/api/converters.go` (internal → API data conversion)
- `internal/tui/api/tui_api_impl.go` (TuiAPI implementation)
- `internal/tui/api/proxy_client.go` (ProxyAPI client wrapper)

**Files to Modify**:
- `internal/proxy/proxy.go` (add API wrapper, emit API calls)
- `internal/tui/app.go` (minimal - add API layer alongside existing proxy)

### Phase 2: Proxy-Side Integration (Minimal TUI Changes)
**Goal**: Complete proxy-side API implementation and event emission.

#### Tasks:
1. **Direct API Call Integration**
   - Wire proxy to call TuiAPI methods directly (connection, script, data events)
   - Update streaming pipeline to call TuiAPI.OnData() instead of direct terminal writes
   - Update script manager to call TuiAPI script methods

2. **State Management Completion**
   - Move all game state tracking from various modules to StateManager
   - Implement API data conversion (internal → Api* types)
   - Add state persistence and recovery mechanisms

3. **Script Manager API Integration**
   - Update script manager to work through API layer
   - Remove direct proxy/terminal dependencies from scripting
   - Call TuiAPI methods for script lifecycle notifications

**Files to Modify**:
- `internal/proxy/proxy.go` (call TuiAPI methods directly)
- `internal/streaming/pipeline.go` (call TuiAPI.OnData(), not direct writes)
- `internal/scripting/integration.go` (use API interfaces)
- `internal/scripting/manager/manager.go` (call TuiAPI script methods)

### Phase 3: TUI-Side Migration (Breaking Changes Start)
**Goal**: Update TUI to use API exclusively, removing direct proxy access.

#### Tasks:
1. **Remove Direct Proxy Access from TUI**
   - Replace `ta.proxy.Connect()` calls with `proxyAPI.Connect()`
   - Replace `ta.proxy.SendInput()` calls with `proxyAPI.SendCommand()`
   - Replace `ta.proxy.GetScriptManager()` with `proxyAPI.GetScriptStatus()`

2. **API-Driven Updates**
   - Replace terminal callback with TuiAPI.OnData() method
   - Update status component to use TuiAPI.OnScriptLoaded() instead of direct script manager access
   - Implement all TuiAPI methods with async goroutines

3. **State Access Migration**
   - Remove TUI-side connection state management
   - Use ProxyAPI for all state queries (GetGameState(), GetConnectionInfo())
   - Update UI components to use Api* data structures only

**Files to Modify**:
- `internal/tui/app.go` (major refactor - remove proxy field, add ProxyAPI usage)
- `internal/tui/components/status.go` (use TuiAPI callbacks instead of direct access)
- `internal/tui/components/terminal.go` (receive data via TuiAPI.OnData())

### Phase 4: Module Separation (Final Cleanup)
**Goal**: Complete architectural separation and code organization.

#### Tasks:
1. **Move Modules to Proxy Package**
   - Move `internal/streaming` to `internal/proxy/streaming`
   - Move `internal/scripting` to `internal/proxy/scripting`
   - Move `internal/database` to `internal/proxy/database`
   - Update import paths throughout codebase

2. **Remove Legacy Interfaces**
   - Remove `streaming.TerminalWriter` interface
   - Remove direct terminal injection into proxy
   - Clean up unused proxy methods

3. **API Refinement**
   - Add missing Api* types discovered during implementation
   - Optimize direct method call performance
   - Add API versioning support for future changes

4. **Documentation and Testing**
   - Add comprehensive API documentation
   - Create integration tests for API interactions
   - Update existing tests to use API interfaces

**Files to Modify**:
- All import statements throughout codebase
- Remove `internal/terminal` injection from proxy
- Clean up legacy interfaces and unused code

### Phase 5: Advanced Features (Future Enhancement)
**Goal**: Add advanced API features inspired by TWX architecture.

#### Tasks:
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

## Success Criteria

1. **Separation of Concerns**: TUI has zero direct access to proxy internals
2. **API-Driven**: All communication uses direct TuiAPI calls or ProxyAPI methods
3. **State Management**: Centralized state with read-only API access patterns
4. **Data Isolation**: TUI only sees Api* data structures, never internal objects
5. **Non-Blocking**: All TuiAPI methods return immediately via goroutines
6. **Testability**: API interfaces enable comprehensive unit testing
7. **Extensibility**: New features can be added without breaking changes
8. **Performance**: No performance regression from current implementation

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

## Timeline Estimate

- **Phase 1**: 3-4 days (API foundation)
- **Phase 2**: 4-5 days (Proxy integration)  
- **Phase 3**: 3-4 days (TUI migration)
- **Phase 4**: 2-3 days (Module separation)
- **Phase 5**: 5-7 days (Advanced features - optional)

**Total**: 12-16 days for core separation (Phases 1-4)

## Agent Assignment Strategy

Each phase should be handled by separate agent invocations with this document as context:

1. **Agent 1**: "Implement Phase 1 of proxy-TUI API separation per docs/api.md"
2. **Agent 2**: "Implement Phase 2 of proxy-TUI API separation per docs/api.md"  
3. **Agent 3**: "Implement Phase 3 of proxy-TUI API separation per docs/api.md"
4. **Agent 4**: "Implement Phase 4 of proxy-TUI API separation per docs/api.md"

This approach ensures each agent has complete context while maintaining manageable scope per agent session.