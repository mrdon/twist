# API Phase 3: Script Management Migration (Updated Based on Current Implementation)

## Goal

Migrate script functionality to use API exclusively, eliminating direct script manager access from TUI and establishing clean script lifecycle management through the API layer.

## Overview

Phase 3 completes the script management migration with **minimal scope** focused on API separation:
1. **Adding basic script methods to ProxyAPI** - LoadScript, StopAllScripts, GetScriptStatus
2. **Adding basic script events to TuiAPI** - OnScriptStatusChanged, OnScriptError  
3. **Removing direct script manager access from TUI** - status component uses API only
4. **Simple script data conversion** - minimal API data structures for basic info
5. **Clean API separation** - TUI cannot access script manager directly

**Out of Scope for Phase 3** (saved for future phases):
- Complex trigger system implementation
- Individual script stopping by ID
- Script variable management
- Bot management features
- Advanced TWX compatibility features

## Current State Analysis

### ✅ What Was Successfully Implemented in Phase 2:

**Clean API Communication**:
- TerminalWriter interface completely eliminated
- Proxy constructor takes TuiAPI directly: `func New(tuiAPI api.TuiAPI) *Proxy`
- Pipeline calls `tuiAPI.OnData(decoded)` directly
- No coupling interfaces between proxy and TUI

**Connection Management**:
- Single callback: `OnConnectionStatusChanged(status ConnectionStatus, address string)`
- Static connection model: `api.Connect()` function creates connected instances
- Symmetric data flow: `SendData()` and `OnData()` methods

### ❌ Script Management Problems to Fix:

**Direct Script Manager Access**: TUI still bypasses API:
- Status component directly accesses script manager via ScriptManagerInterface (status.go:15)
- `SetScriptManager(sm ScriptManagerInterface)` method bypasses API layer (status.go:54)
- No API data structures for script information
- Script lifecycle events not implemented

**Missing Script API Methods**:
- ProxyAPI methods exist in proxy.go but not exposed through API layer
- No script lifecycle callbacks in TuiAPI (OnScriptLoaded, OnScriptStopped, etc.)
- No script data conversion (internal ScriptManager data → API types)

**Current Implementation Analysis**:
- API package is at `/internal/api/api.go` (not `/internal/proxy/api/` as originally planned)
- Script manager already initialized with database: `scriptManager := scripting.NewScriptManager(db)` (proxy.go:49)
- SetupConnections pattern requires TerminalInterface: `scriptManager.SetupConnections(p, nil)` (proxy.go:65)
- Script methods already exist on proxy: LoadScript, ExecuteScriptCommand, GetScriptStatus, StopAllScripts (proxy.go:269-287)

## Scope - Script Management API Design (Corrected)

### Package Structure (Fixed):

**Actual API Package**: `/internal/api/api.go` (verified from current implementation)
**Import Statement**: `"twist/internal/api"` 
**Type References**: `api.ScriptStatusInfo` (not `proxyapi.ScriptStatusInfo`)

### Script API Interface Design:

**ProxyAPI Interface - Minimal Script Methods (extend existing interface in `/internal/api/api.go`):**
```go
type ProxyAPI interface {
    // Connection Management (from Phase 2)
    Disconnect() error
    IsConnected() bool
    SendData(data []byte) error
    
    // Script Management (Phase 3 - Minimal) 
    LoadScript(filename string) error
    StopAllScripts() error
    GetScriptStatus() ScriptStatusInfo  // No error return - always succeeds
}
```

**TuiAPI Interface - Minimal Script Events:**
```go
type TuiAPI interface {
    // Connection Events (from Phase 2)
    OnConnectionStatusChanged(status ConnectionStatus, address string)
    OnConnectionError(err error)
    OnData(data []byte)
    
    // Script Events (Phase 3 - Minimal)
    OnScriptStatusChanged(status ScriptStatusInfo)
    OnScriptError(scriptName string, err error)
}
```

**Minimal Script Data Types (Phase 3 scope only):**
```go
// Simple script status information - just what status component needs
type ScriptStatusInfo struct {
    ActiveCount int      `json:"active_count"`  // Number of running scripts
    TotalCount  int      `json:"total_count"`   // Total number of loaded scripts  
    ScriptNames []string `json:"script_names"`  // Names of loaded scripts
}
```

**Future Data Types (Phase 4+):**
```go
// Full ScriptInfo structure for individual script details (Phase 4+)
type ScriptInfo struct {
    ID             string    `json:"id"`
    Name           string    `json:"name"`
    Filename       string    `json:"filename"`
    Status         string    `json:"status"`  // "running", "stopped", "error"
    System         bool      `json:"is_system"`
    LoadTime       time.Time `json:"loaded_at"`
    // ... additional fields in future phases
}
```

### Script Management Architecture:

**Centralized Script State**: Proxy maintains authoritative script state
- Internal script manager remains in proxy module
- API layer provides read-only access to script data
- All script mutations go through ProxyAPI methods
- Script lifecycle events broadcast via TuiAPI callbacks

**Data Flow Pattern**:
1. **TUI → ProxyAPI**: `LoadScript(filename)`, `StopScript(id)`, etc.
2. **ProxyAPI → ScriptManager**: Delegate to internal script manager
3. **ScriptManager → TuiAPI**: Script events via callbacks (OnScriptLoaded, etc.)
4. **TuiAPI → TUI**: Update UI components asynchronously

**Non-Blocking Script Operations**: All script operations return immediately
- Script loading/stopping happens asynchronously
- Results reported via TuiAPI callbacks
- Errors handled through OnScriptError callback
- Status updates through script lifecycle events

## TWX Compatibility Reference (Future Phases)

**Note**: Full TWX Script.pas compatibility analysis has been documented but is **out of scope for Phase 3**. The current phase focuses only on basic API separation.

**TWX Features for Future Implementation:**
- Complete trigger system (6 trigger types)
- Bot management and switching
- Script variable system
- Advanced event processing
- Auto-run script management

**Phase 3 Approach**: Design the minimal API with TWX patterns in mind, but implement only what's needed to eliminate direct script manager access from TUI.

## Implementation Steps (Minimal Scope)

### Step 1: Add Minimal Script Types to API

#### 1.1 Add Script Types to API

Update `internal/api/api.go` (add types at the end of the file):

```go
// ADD minimal script types for Phase 3 only

// Simple script status information for status component
type ScriptStatusInfo struct {
    ActiveCount int      `json:"active_count"`  // Number of running scripts
    TotalCount  int      `json:"total_count"`   // Total number of loaded scripts  
    ScriptNames []string `json:"script_names"`  // Names of loaded scripts
}
```

### Step 2: Update ProxyAPI Interface

#### 2.1 Add Script Methods to ProxyAPI Interface

Update `internal/api/api.go` - add methods to ProxyAPI interface:

```go
type ProxyAPI interface {
    // Connection Management (from Phase 2)
    Disconnect() error
    IsConnected() bool
    SendData(data []byte) error
    
    // Script Management (Phase 3)
    LoadScript(filename string) error
    StopAllScripts() error
    GetScriptStatus() ScriptStatusInfo
}
```

#### 2.2 Add Script Event Methods to TuiAPI Interface  

Update `internal/api/api.go` - add methods to TuiAPI interface:

```go
type TuiAPI interface {
    // Connection Events (from Phase 2)
    OnConnectionStatusChanged(status ConnectionStatus, address string)
    OnConnectionError(err error)
    OnData(data []byte)
    
    // Script Events (Phase 3)
    OnScriptStatusChanged(status ScriptStatusInfo)
    OnScriptError(scriptName string, err error)
}
```

#### 2.3 Add Script Status Converter Function

Add to `internal/proxy/script_manager_api.go` (conversion from existing proxy data):

```go
// convertScriptStatus converts proxy script manager status to API format
func convertScriptStatus(statusMap map[string]interface{}, scriptManager interface{}) api.ScriptStatusInfo {
    // Extract counts from existing GetStatus() return value
    totalCount := 0
    activeCount := 0
    
    if total, ok := statusMap["total_scripts"].(int); ok {
        totalCount = total
    }
    if running, ok := statusMap["running_scripts"].(int); ok {
        activeCount = running
    }
    
    // Get script names - this will require accessing the script manager
    scriptNames := []string{}
    // TODO: Add method to get script names from engine.ListScripts()
    
    return api.ScriptStatusInfo{
        ActiveCount: activeCount,
        TotalCount:  totalCount,
        ScriptNames: scriptNames,
    }
}
```

### Step 3: Implement ProxyAPI Script Methods  

#### 3.1 Add Script Methods to ProxyApiImpl

**Decision**: Add methods to existing `internal/proxy/proxy_api_impl.go` (keep in one file for now)

Add these methods to `ProxyApiImpl` struct:

```go
// Script Management Methods - Thin orchestration layer (one-liners)

func (p *ProxyApiImpl) LoadScript(filename string) error {
    return p.scriptManager.LoadScriptAsync(filename, p.tuiAPI) // One-liner delegate
}

func (p *ProxyApiImpl) StopAllScripts() error {
    return p.scriptManager.StopAllScriptsAsync(p.tuiAPI) // One-liner delegate
}

func (p *ProxyApiImpl) GetScriptStatus() ScriptStatusInfo {
    return p.scriptManager.GetScriptStatusInfo() // One-liner delegate
}
```

#### 3.2 Add ScriptManager Field to ProxyApiImpl

Update `internal/proxy/proxy_api_impl.go` - add field and connect to actual implementation:

```go
// UPDATE ProxyApiImpl struct to include script manager
type ProxyApiImpl struct {
    proxy         *proxy.Proxy
    tuiAPI        api.TuiAPI
    scriptManager *ScriptManagerAPI  // ADD this field for delegation
}

// UPDATE Connect function to initialize script manager API
func Connect(address string, tuiAPI api.TuiAPI) api.ProxyAPI {
    // ... existing connection logic ...
    
    impl := &ProxyApiImpl{
        proxy:  proxyInstance,
        tuiAPI: tuiAPI,
        scriptManager: NewScriptManagerAPI(proxyInstance, tuiAPI), // Initialize script manager API
    }
    
    // ... rest of connection logic ...
    
    return impl
}
```

#### 3.3 Create ScriptManagerAPI (Business Logic Layer)

Create `internal/proxy/script_manager_api.go` (new file for script business logic):

```go
// ScriptManagerAPI - Contains actual script management business logic
// Note: Non-blocking requirements only apply to ProxyAPI/TuiAPI methods

package proxy

import (
    "errors"
    "twist/internal/api"
)

type ScriptManagerAPI struct {
    proxy  *Proxy  // Fixed reference
    tuiAPI api.TuiAPI
}

func NewScriptManagerAPI(proxy *Proxy, tuiAPI api.TuiAPI) *ScriptManagerAPI {
    return &ScriptManagerAPI{
        proxy:  proxy,
        tuiAPI: tuiAPI,
    }
}

// LoadScriptAsync - Returns immediately, does work in goroutine
func (sm *ScriptManagerAPI) LoadScriptAsync(filename string, tuiAPI api.TuiAPI) error {
    if sm.proxy == nil {
        return errors.New("not connected")
    }
    
    // Do work asynchronously - return immediately
    go func() {
        err := sm.proxy.LoadScript(filename)
        if err != nil {
            tuiAPI.OnScriptError(filename, err)
        } else {
            // Report status change
            status := sm.GetScriptStatusInfo()
            tuiAPI.OnScriptStatusChanged(status)
        }
    }()
    
    return nil // Returns immediately
}

// StopAllScriptsAsync - Returns immediately, does work in goroutine  
func (sm *ScriptManagerAPI) StopAllScriptsAsync(tuiAPI api.TuiAPI) error {
    if sm.proxy == nil {
        return errors.New("not connected")
    }
    
    // Do work asynchronously - return immediately
    go func() {
        err := sm.proxy.StopAllScripts()
        if err != nil {
            tuiAPI.OnScriptError("all scripts", err)
        } else {
            // Report status change
            status := sm.GetScriptStatusInfo()
            tuiAPI.OnScriptStatusChanged(status)
        }
    }()
    
    return nil // Returns immediately
}

// GetScriptStatusInfo - Returns immediately with current status
func (sm *ScriptManagerAPI) GetScriptStatusInfo() api.ScriptStatusInfo {
    if sm.proxy == nil {
        return api.ScriptStatusInfo{
            ActiveCount: 0,
            TotalCount:  0,
            ScriptNames: []string{},
        }
    }
    
    // Get status from proxy and convert
    statusMap := sm.proxy.GetScriptStatus()
    return convertScriptStatus(statusMap, sm.proxy.GetScriptManager())
}
```

### Step 4: Update Script Manager Constructor

#### 4.1 Update ScriptManager to Accept TuiAPI

**Problem**: Current `SetupConnections(p, nil)` expects TerminalInterface  
**Solution**: Pass TuiAPI to script manager so it can call callback methods

Update `internal/proxy/proxy.go` - modify script manager initialization:

```go
// UPDATE the proxy constructor - pass TuiAPI to script manager
func New(tuiAPI api.TuiAPI) *Proxy {
    // ... existing database initialization ...
    
    // Create script manager with TuiAPI reference  
    scriptManager := scripting.NewScriptManager(db, tuiAPI)
    
    p := &Proxy{
        // ... existing fields ...
        scriptManager: scriptManager,
        tuiAPI:        tuiAPI,
    }
    
    // Initialize streaming pipeline with TuiAPI directly
    p.pipeline = streaming.NewPipelineWithScriptManager(tuiAPI, db, scriptManager)
    
    // Setup script manager connections - pass TuiAPI instead of nil
    scriptManager.SetupConnections(p, tuiAPI)
    
    return p
}
```

### Step 5: Remove Direct Script Manager Access from TUI

#### 5.1 Update Status Component to Use ProxyAPI

**Current Problem**: `SetScriptManager(sm ScriptManagerInterface)` bypasses API  
**Solution**: Replace with ProxyAPI access

Update `internal/tui/components/status.go`:

```go
// UPDATE Status component to use ProxyAPI only

type Status struct {
    // ... existing fields ...
    // REMOVE: scriptManager field completely
    proxyAPI api.ProxyAPI  // ADD this field
}

// REMOVE SetScriptManager method completely:
// func (s *Status) SetScriptManager(sm ScriptManagerInterface) {
//     // Remove this method entirely
// }

// ADD SetProxyAPI method
func (s *Status) SetProxyAPI(proxyAPI api.ProxyAPI) {
    s.proxyAPI = proxyAPI
}

// UPDATE display method to use API
func (s *Status) GetInfo() string {
    info := ""
    
    // ... existing connection status logic unchanged ...
    
    // UPDATE script status to use API
    if s.proxyAPI != nil {
        scriptStatus := s.proxyAPI.GetScriptStatus()
        info += fmt.Sprintf("Scripts: %d active, %d total\n", 
            scriptStatus.ActiveCount, scriptStatus.TotalCount)
        
        // Show script names if any loaded
        if len(scriptStatus.ScriptNames) > 0 {
            info += "Loaded: " + strings.Join(scriptStatus.ScriptNames, ", ") + "\n"
        }
    } else {
        info += "Scripts: not connected\n"
    }
    
    return info
}
```

#### 5.2 Update TUI App to Set ProxyAPI on Status Component

Update `internal/tui/app.go` - modify connection handler:

```go
// UPDATE HandleConnectionStatusChanged to set ProxyAPI on status component
func (ta *TwistApp) HandleConnectionStatusChanged(status api.ConnectionStatus, address string) {
    ta.app.QueueUpdateDraw(func() {
        switch status {
        case api.ConnectionStatusConnected:
            ta.connected = true
            ta.serverAddress = address
            ta.menuComponent.SetConnectedMenu()
            ta.statusComponent.SetConnectionStatus(true, address)
            
            // SET ProxyAPI on status component after connection established
            if ta.proxyClient.IsConnected() {
                ta.statusComponent.SetProxyAPI(ta.proxyClient.GetCurrentAPI())
            }
            
            // ... rest of connection logic unchanged ...
            
        // ... other status cases unchanged ...
        }
    })
}

// ADD script event handlers
func (ta *TwistApp) HandleScriptStatusChanged(status api.ScriptStatusInfo) {
    // Status component will be updated automatically via SetProxyAPI
    // For now, just output to terminal
    msg := fmt.Sprintf("Script status: %d active, %d total\n", 
        status.ActiveCount, status.TotalCount)
    ta.terminalComponent.Write([]byte(msg))
}

func (ta *TwistApp) HandleScriptError(scriptName string, err error) {
    // Output script error to terminal
    msg := fmt.Sprintf("Script error in %s: %s\n", scriptName, err.Error())
    ta.terminalComponent.Write([]byte(msg))
}
```
## Testing Phase 3

### Manual Testing Steps

1. **Script Loading**: Test loading scripts through TUI menu system
2. **Script Status Display**: Verify status component shows script information via API
3. **Script Lifecycle**: Test start/stop/error scenarios with proper callbacks
4. **Connection Cycle**: Test script manager availability after reconnection
5. **Error Handling**: Test script loading failures and error display

### Key Validation Points

- [ ] **No direct script manager access** in TUI components
- [ ] **Script lifecycle events** work through TuiAPI callbacks
- [ ] **Script status display** uses ProxyAPI.GetScriptStatus() only
- [ ] **Script operations** (load/stop) work through ProxyAPI methods
- [ ] **Error handling** works via OnScriptError callback
- [ ] **Status component** updates automatically on script events

### Expected Behavior

- Status component gets script information via API calls only
- Script loading/stopping triggers appropriate TuiAPI callbacks
- Script errors display in terminal and update status component
- No direct access to internal ScriptManager from TUI code
- Script manager lifecycle properly integrated with connection events

## Success Criteria (Minimal Scope)

✅ **Direct Script Manager Access Eliminated**: TUI has no direct script manager access  
✅ **Basic Script API Methods**: LoadScript, StopAllScripts, GetScriptStatus work via API  
✅ **Minimal Script Events**: OnScriptStatusChanged, OnScriptError implemented  
✅ **Simple Script Data**: ScriptStatusInfo provides basic count/name information  
✅ **Status Component Migration**: Uses ProxyAPI exclusively for script information  
✅ **Clean API Separation**: No TUI imports of internal scripting packages  

## Files to Modify Summary

### Core API Files:
- `internal/api/api.go` - Add ScriptStatusInfo type and extend ProxyAPI/TuiAPI interfaces

### Script Manager Integration:
- `internal/proxy/proxy.go` - Update constructor to pass TuiAPI to script manager
- `internal/proxy/script_manager_api.go` - New file for script business logic layer
- `internal/scripting/` - Add TuiAPI integration, call callbacks on events

### TUI Integration:
- `internal/tui/api/tui_api_impl.go` - Implement script event handlers
- `internal/tui/app.go` - Add script event handlers, update connection handler
- `internal/tui/components/status.go` - Remove direct script manager, use ProxyAPI only

## Key Architectural Changes

1. **Script Data Conversion**: Clean mapping between internal script data and API types
2. **Event-Driven Script Updates**: Script lifecycle changes trigger TuiAPI callbacks
3. **Centralized Script State**: Proxy maintains authoritative script state, TUI gets read-only access
4. **Non-Blocking Script Operations**: All script operations asynchronous with callback results
5. **API-Only Script Access**: TUI components use ProxyAPI exclusively for script data

## Implementation Notes

### Critical Integration Points:

1. **Script Manager Constructor Update**: Must pass TuiAPI to script manager for callback integration
2. **Status Component Migration**: Complete removal of direct script manager access
3. **Event Handler Integration**: TUI event handlers must update UI components appropriately
4. **Error Flow**: Script errors must flow through OnScriptError to maintain clean separation

### Performance Considerations:
- Script status queries should be lightweight (cached data preferred)
- Script event callbacks must return immediately (use goroutines for UI work)

## Risk Mitigation

1. **Incremental Implementation**: Implement API methods before removing direct access
2. **Fallback Handling**: Graceful degradation when script manager unavailable
3. **Event Callback Safety**: All TuiAPI callbacks must handle nil checks and errors
4. **UI Update Coordination**: Use QueueUpdateDraw for all UI updates from callbacks
5. **Testing at Each Step**: Verify script operations work before removing legacy code

## Next Steps (Phase 4)

Phase 4 will add game state management API methods:
- Add game state tracking methods to ProxyAPI
- Add game state events to TuiAPI (OnSectorChanged, OnPlayerStatsChanged, etc.)
- Migrate game data parsing to trigger API callbacks
- Remove any remaining direct coupling between proxy internals and TUI
- Implement centralized state management in proxy

This phase establishes **complete script management through API** with no direct coupling between TUI and internal script manager.