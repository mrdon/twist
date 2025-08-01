# API Phase 4: Game State Management Migration (SIMPLIFIED)

## Goal

Migrate game state and data access to use API exclusively, using **simple direct delegation** following the successful patterns from Phases 2-3, eliminating remaining direct coupling between TUI and proxy internals.

## Overview

Phase 4 completes the game state migration with **simplified incremental approach** broken into small sub-phases:

### **Phase 4.1: Fix Critical Dependency Violation** ⚡ **IMMEDIATE**
1. **Remove database import from TUI** - fix `internal/tui/components/panels.go` 
2. **Create basic API data types** - simple `PlayerInfo`, `SectorInfo` for TUI display
3. **Add placeholder API methods** - basic structure without complex logic

### **Phase 4.2: Add Basic Game State API** 
1. **Add simple ProxyAPI methods** - `GetCurrentSector()`, `GetSectorInfo()` using direct delegation
2. **Add basic database access** - `GetDatabase()` method to proxy for API access
3. **Simple data conversion** - lightweight converter functions

### **Phase 4.3: Add Basic State Tracking**
1. **Add current sector tracking** - simple `currentSector int` field in proxy
2. **Basic TuiAPI events** - `OnCurrentSectorChanged()` for sector updates
3. **Simple parser callbacks** - minimal integration points

### **Phase 4.4: Complete TUI Migration**
1. **Update panel components** - use API methods instead of database direct access
2. **Add TUI event handlers** - basic state update handling
3. **Integration testing** - ensure all functionality works

**ARCHITECTURAL PRINCIPLE**: **Simple Direct Delegation** - following the thin orchestration pattern that worked successfully in Phases 2-3, avoiding complex StateManager architecture.

**Out of Scope for Phase 4** (saved for future phases):
- Complex centralized state management
- Advanced caching and performance optimization
- Advanced analytics and statistics
- Historical state tracking
- Complex trigger system integration

## Current State Analysis

### ✅ What Was Successfully Implemented in Previous Phases:

**Clean API Communication (Phase 2)**:
- TuiAPI callbacks with immediate return pattern
- Static connection model via `api.Connect()` function
- Symmetric data flow with `SendData()` and `OnData()`

**Script Management API (Phase 3)**:
- Script lifecycle management through API: `LoadScript()`, `StopAllScripts()`, `GetScriptStatus()`
- Script events via callbacks: `OnScriptStatusChanged()`, `OnScriptError()`
- Dependency violation fixed: TUI no longer imports `internal/scripting`

### ❌ Game State Problems to Fix:

**🚨 CRITICAL DEPENDENCY VIOLATION**: TUI components access game data directly:
- `/internal/tui/components/panels.go` imports `"twist/internal/database"` (line 6) - **VIOLATES API SEPARATION**
- Panel component uses `*database.TSector` and `*database.TTrader` directly (lines 59, 77)
- This is the **primary architectural violation** that Phase 4 must fix immediately

**Missing Basic Game State API**:
- No game state methods in ProxyAPI interface  
- No basic API data structures for TUI display
- No simple data conversion from database to API types
- TUI has no API-based way to get current sector or basic game info

**Current Game Data Flow (Problematic)**:
```
Database → TUI Components (DIRECT ACCESS - VIOLATES API)
```

**Target Game Data Flow (Simple API-Based)**:
```
Database → ProxyAPI → API Data Types → TUI Components
```

**Key Implementation Requirements**:
- **Simple direct delegation** following successful Phase 2-3 patterns
- **Prioritize dependency violation fix** as immediate critical issue
- **Break into smaller sub-phases** to reduce implementation risk
- **Avoid complex StateManager architecture** - use thin orchestration only

## Scope - Simplified Game State API Design

### Package Structure (Unchanged):
**API Package**: `/internal/api/api.go` (established in previous phases)  
**Import Pattern**: `"twist/internal/api"`
**Type References**: `api.PlayerInfo`, `api.SectorInfo` (simplified types)

### **SIMPLIFIED** Game State API Interface Design:

**ProxyAPI Interface - Add ONLY Essential Methods (extend existing interface in `/internal/api/api.go`):**
```go
type ProxyAPI interface {
    // Connection Management (Phase 2)
    Disconnect() error
    IsConnected() bool
    SendData(data []byte) error
    
    // Script Management (Phase 3)
    LoadScript(filename string) error
    StopAllScripts() error
    GetScriptStatus() ScriptStatusInfo
    
    // Game State Management (Phase 4 - SIMPLIFIED)
    GetCurrentSector() (int, error)               // Phase 4.2
    GetSectorInfo(sectorNum int) (SectorInfo, error)  // Phase 4.2
    GetPlayerInfo() (PlayerInfo, error)           // Phase 4.2
}
```

**TuiAPI Interface - Add ONLY Basic Events:**
```go
type TuiAPI interface {
    // Connection Events (Phase 2)
    OnConnectionStatusChanged(status ConnectionStatus, address string)
    OnConnectionError(err error)
    OnData(data []byte)
    
    // Script Events (Phase 3)
    OnScriptStatusChanged(status ScriptStatusInfo)
    OnScriptError(scriptName string, err error)
    
    // Game State Events (Phase 4 - MINIMAL)
    OnCurrentSectorChanged(sectorNumber int)      // Phase 4.3 - Simple callback
}
```

**KEY SIMPLIFICATIONS**:
- **Removed `GameStateInfo`** - too complex for Phase 4, defer to Phase 5
- **Simplified sector callback** - just sector number, not full info object
- **Removed `OnPlayerInfoChanged`** - defer to Phase 5
- **Consistent error returns** - all methods return `(Type, error)` pattern

### **SIMPLIFIED** Game State Data Types (Phase 4 scope only):

**MINIMAL API Data Types for TUI Display:**
```go
// Basic player information (Phase 4.1) - SIMPLIFIED based on current capability
type PlayerInfo struct {
    CurrentSector int `json:"current_sector"` // Current sector location only
    // Name field REMOVED - no current player name storage in system
}

// Basic sector information for panel display (Phase 4.1)
type SectorInfo struct {
    Number        int    `json:"number"`         // Sector number
    NavHaz        int    `json:"nav_haz"`        // Navigation hazard level  
    HasTraders    int    `json:"has_traders"`    // Number of traders present
    Constellation string `json:"constellation"`  // Constellation name
    Beacon        string `json:"beacon"`         // Beacon text
}
```

**Key Simplifications vs Original Spec**:
- **Removed `GameStateInfo`** - too complex, defer to Phase 5
- **Removed `PlayerInfo.Name`** - no current player name storage exists in system
- **Removed `HasPort`, `HasShips`, `Warps[]`, `LastUpdate`** - not essential for basic panel display
- **Simplified `SectorInfo`** - only fields actually used by current panel component
- **No complex arrays or time formatting** - keep data conversion simple

**Future Data Types (Phase 5+):**
```go
// Detailed types will be added in Phase 5 when we need:
// - Full game state tracking
// - Historical data
// - Complex sector information with ships, planets, etc.
// - Advanced player statistics
```

## **SIMPLIFIED** Game State Management Architecture:

### **Simple Direct Delegation** (Following Phase 2-3 Success Pattern):
**No Complex StateManager** - Use the same thin orchestration pattern that worked successfully:
- ProxyAPI methods delegate directly to proxy internal methods
- Simple `currentSector int` field in proxy for basic state tracking
- Direct database access through existing proxy database field
- Minimal parser integration with simple callback hooks

**Simplified Data Flow Pattern**:
1. **TUI → ProxyAPI**: Call `GetSectorInfo(sectorNum)` 
2. **ProxyAPI → Proxy**: Direct delegation `p.proxy.GetDatabase().GetSector(sectorNum)`
3. **ProxyAPI → Conversion**: Simple `convertDatabaseSectorToAPI(dbSector)`
4. **ProxyAPI → TUI**: Return converted API data immediately

**Key Architectural Decisions**:
- **No Caching** - direct database access, let database handle performance
- **No Complex State Tracking** - just basic `currentSector int` field  
- **No Goroutines for State** - only for TuiAPI callbacks (same as Phase 2-3)
- **Simple Conversion Functions** - lightweight mapping of database fields to API fields

## Implementation Steps (Simplified Incremental Approach)

## **Phase 4.1: Fix Critical Dependency Violation** ⚡ **IMMEDIATE**

### Step 1.1: Add Minimal API Types

Update `internal/api/api.go` (add types at the end of the file):

```go
// ADD MINIMAL game state types for Phase 4.1

// Basic player information 
type PlayerInfo struct {
    Name          string `json:"name"`           // Player name (if available)  
    CurrentSector int    `json:"current_sector"` // Current sector location
}

// Basic sector information for panel display
type SectorInfo struct {
    Number        int    `json:"number"`         // Sector number
    NavHaz        int    `json:"nav_haz"`        // Navigation hazard level  
    HasTraders    int    `json:"has_traders"`    // Number of traders present
    Constellation string `json:"constellation"`  // Constellation name
    Beacon        string `json:"beacon"`         // Beacon text
}
```

**Note**: These are minimal types - only fields actually needed by panel components.

### Step 1.2: Fix Panel Component Database Dependency

**🚨 CRITICAL**: Remove database import and create placeholder API usage

Update `internal/tui/components/panels.go`:

```go
// REMOVE database dependency - fix critical violation
package components

import (
    "fmt"
    "strings"
    // REMOVE: "twist/internal/database"  ← DEPENDENCY VIOLATION FIXED
    "twist/internal/api"                   // ← Use core API only
    "twist/internal/theme"
    
    "github.com/rivo/tview"
)

// PanelComponent manages the side panel components
type PanelComponent struct {
    leftView     *tview.TextView
    leftWrapper  *tview.Flex
    rightView    *tview.TextView
    rightWrapper *tview.Flex
    proxyAPI     api.ProxyAPI  // ADD: API access for game data
}

// ... NewPanelComponent unchanged ...

// ADD SetProxyAPI method
func (pc *PanelComponent) SetProxyAPI(proxyAPI api.ProxyAPI) {
    pc.proxyAPI = proxyAPI
}

// UPDATE UpdateTraderInfo to use API PlayerInfo data (keeps existing method name per TWX)
func (pc *PanelComponent) UpdateTraderInfo(playerInfo api.PlayerInfo) {
    var info strings.Builder
    info.WriteString(fmt.Sprintf("[yellow]Player Info[-]\n"))
    info.WriteString(fmt.Sprintf("Current Sector: %d\n", playerInfo.CurrentSector))
    
    pc.leftView.SetText(info.String())
}

// UPDATE UpdateSectorInfo to use API data structures (placeholder)
func (pc *PanelComponent) UpdateSectorInfo(sector api.SectorInfo) {
    var info strings.Builder
    info.WriteString(fmt.Sprintf("[cyan]Sector %d Info[-]\n", sector.Number))
    info.WriteString(fmt.Sprintf("Nav Hazard: %d\n", sector.NavHaz))
    
    if sector.HasTraders > 0 {
        info.WriteString(fmt.Sprintf("Traders: %d\n", sector.HasTraders))
    }
    
    if sector.Constellation != "" {
        info.WriteString(fmt.Sprintf("Constellation: %s\n", sector.Constellation))
    }
    
    if sector.Beacon != "" {
        info.WriteString(fmt.Sprintf("Beacon: %s\n", sector.Beacon))
    }
    
    pc.rightView.SetText(info.String())
}

// ADD SetPlaceholderText methods for testing Phase 4.1
func (pc *PanelComponent) SetPlaceholderPlayerText() {
    pc.leftView.SetText("[yellow]Player Info[-]\nAPI data not yet available")
}

func (pc *PanelComponent) SetPlaceholderSectorText() {
    pc.rightView.SetText("[cyan]Sector Info[-]\nAPI data not yet available")
}

// REMOVE: Old methods that used database types
// func (pc *PanelComponent) UpdateTraderInfo(trader *database.TTrader) { ... }
// func (pc *PanelComponent) UpdateSectorInfo(sector *database.TSector) { ... }
```

**Result**: Database dependency violation is **FIXED** - TUI no longer imports internal database.

**Phase 4.1 Complete**: Critical dependency violation fixed, basic API types added, panel component ready for API integration.

---

## **Phase 4.2: Add Basic Game State API** 

### Step 2.1: Add Database Access Methods to Proxy

**Simple Database Access** - Add methods for API layer to access database:

Update `internal/proxy/proxy.go` - add database access methods:

```go
// ADD GetDatabase method for API access (follows GetScriptManager pattern line 274)
func (p *Proxy) GetDatabase() database.Database {
    return p.db
}

// ADD GetSector wrapper method for cleaner API (database has LoadSector, API uses GetSector)
func (p *Proxy) GetSector(sectorNum int) (database.TSector, error) {
    return p.db.LoadSector(sectorNum)
}
```

**Note**: Database interface uses `LoadSector(index int) (TSector, error)` (line 21 in database.go), but `GetSector()` is more natural for API usage.

### Step 2.2: Add ProxyAPI Interface Methods

Update `internal/api/api.go` - add methods to ProxyAPI interface:

```go
type ProxyAPI interface {
    // Connection Management (Phase 2)
    Disconnect() error
    IsConnected() bool
    SendData(data []byte) error
    
    // Script Management (Phase 3)
    LoadScript(filename string) error
    StopAllScripts() error
    GetScriptStatus() ScriptStatusInfo
    
    // Game State Management (Phase 4.2 - SIMPLIFIED)
    GetCurrentSector() (int, error)                    // Basic current sector
    GetSectorInfo(sectorNum int) (SectorInfo, error)   // Basic sector info
    GetPlayerInfo() (PlayerInfo, error)                // Basic player info
}
```

### Step 2.3: Add Simple Data Conversion Functions

Create `internal/proxy/game_state_converters.go` (new file for lightweight data conversion):

```go
package proxy

import (
    "twist/internal/api"
    "twist/internal/database"
)

// convertDatabaseSectorToAPI converts database TSector to API SectorInfo
func convertDatabaseSectorToAPI(sectorNum int, dbSector database.TSector) api.SectorInfo {
    return api.SectorInfo{
        Number:        sectorNum,
        NavHaz:        dbSector.NavHaz,  
        HasTraders:    len(dbSector.Traders),
        Constellation: dbSector.Constellation,
        Beacon:        dbSector.Beacon,
    }
}

// convertDatabasePlayerToAPI converts current sector to API PlayerInfo
func convertDatabasePlayerToAPI(currentSector int) api.PlayerInfo {
    return api.PlayerInfo{
        CurrentSector: currentSector,
    }
}
```

### Step 2.4: Implement ProxyAPI Game State Methods

**Simple Direct Delegation** - Add methods to `internal/proxy/proxy_api_impl.go`:

```go
// Game State Management Methods - Simple direct delegation (one-liners)

func (p *ProxyApiImpl) GetCurrentSector() (int, error) {
    // TODO Phase 4.3: Add currentSector field to proxy, for now return 0
    return 0, nil
}

func (p *ProxyApiImpl) GetSectorInfo(sectorNum int) (api.SectorInfo, error) {
    // Direct delegation using new GetSector wrapper method
    dbSector, err := p.proxy.GetSector(sectorNum)
    if err != nil {
        return api.SectorInfo{Number: sectorNum}, err
    }
    
    // Simple conversion using converter function
    return convertDatabaseSectorToAPI(sectorNum, dbSector), nil
}

func (p *ProxyApiImpl) GetPlayerInfo() (api.PlayerInfo, error) {
    // TODO Phase 4.3: Add current sector tracking, for now return placeholder
    return api.PlayerInfo{CurrentSector: 0}, nil
}
```

**Phase 4.2 Complete**: Basic API methods implemented with simple direct delegation pattern.

---

## **Phase 4.3: Add Basic State Tracking**

### Step 3.1: Add Simple Current Sector Tracking

**Add Basic State Fields** - Update `internal/proxy/proxy.go`:

```go
type Proxy struct {
    conn     net.Conn
    reader   *bufio.Reader
    writer   *bufio.Writer
    mu       sync.RWMutex
    connected bool
    
    // ... existing channels and components ...
    
    // ADD: Basic game state tracking (Phase 4.3) - based on parser CurrentSectorIndex
    currentSector int    // Track current sector number (from parser)
    
    // ... rest of existing fields ...
}

// ADD: Simple getter/setter methods
func (p *Proxy) GetCurrentSector() int {
    p.mu.RLock()
    defer p.mu.RUnlock()
    return p.currentSector
}

func (p *Proxy) SetCurrentSector(sectorNum int) {
    p.mu.Lock()
    defer p.mu.Unlock()
    oldSector := p.currentSector
    p.currentSector = sectorNum
    
    // Trigger callback if sector changed and TuiAPI is available
    if oldSector != sectorNum && p.tuiAPI != nil {
        go p.tuiAPI.OnCurrentSectorChanged(sectorNum)
    }
}
```

**Implementation Notes**:
- **Parser Integration**: Parser already tracks `sp.ctx.State.CurrentSectorIndex` (line 66 in parser/types.go)
- **Callback Trigger**: `SetCurrentSector()` automatically calls TuiAPI when sector changes
- **Thread Safety**: Uses existing `mu sync.RWMutex` for state protection

### Step 3.2: Update ProxyAPI Methods to Use State Tracking

**Update ProxyAPI Methods** - Update `internal/proxy/proxy_api_impl.go`:

```go
// UPDATE GetCurrentSector to use proxy state
func (p *ProxyApiImpl) GetCurrentSector() (int, error) {
    return p.proxy.GetCurrentSector(), nil // Simple delegation
}

// UPDATE GetPlayerInfo to use proxy state  
func (p *ProxyApiImpl) GetPlayerInfo() (api.PlayerInfo, error) {
    currentSector := p.proxy.GetCurrentSector()
    return convertDatabasePlayerToAPI(currentSector), nil
}
```

### Step 3.3: Add Simple TuiAPI Event

**Add Basic Event** - Update `internal/api/api.go`:

```go
type TuiAPI interface {
    // Connection Events (Phase 2)
    OnConnectionStatusChanged(status ConnectionStatus, address string)
    OnConnectionError(err error)
    OnData(data []byte)
    
    // Script Events (Phase 3)
    OnScriptStatusChanged(status ScriptStatusInfo)
    OnScriptError(scriptName string, err error)
    
    // Game State Events (Phase 4.3 - MINIMAL)
    OnCurrentSectorChanged(sectorNumber int)      // Simple sector change callback
}
```

**Phase 4.3 Complete**: Basic state tracking added with simple getter/setter methods.

---

## **Phase 4.4: Complete TUI Migration**

### Step 4.1: Implement TuiAPI Event Handler

**Add Event Handler** - Update `internal/tui/api/tui_api_impl.go`:

```go
// ADD simple sector change handler

// OnCurrentSectorChanged handles sector changes (returns immediately)
func (tui *TuiApiImpl) OnCurrentSectorChanged(sectorNumber int) {
    go func() {
        tui.app.QueueUpdateDraw(func() {
            // Update sector-specific UI components
            tui.app.HandleCurrentSectorChanged(sectorNumber)
        })
    }()
}
```

### Step 4.2: Add TUI App Event Handler

**Add App Handler** - Update `internal/tui/app.go`:

```go
// ADD simple sector change handler

// HandleCurrentSectorChanged processes sector change events
func (ta *TwistApp) HandleCurrentSectorChanged(sectorNumber int) {
    // Output sector change to terminal
    msg := fmt.Sprintf("Entered sector %d\n", sectorNumber)
    ta.terminalComponent.Write([]byte(msg))
    
    // Update panels with current sector data via API
    if ta.panelComponent != nil && ta.proxyClient.IsConnected() {
        ta.refreshPanelData(sectorNumber)
    }
}

// ADD helper method to refresh panel data
func (ta *TwistApp) refreshPanelData(sectorNumber int) {
    proxyAPI := ta.proxyClient.GetCurrentAPI()
    if proxyAPI != nil {
        // Get sector info and update panel
        sectorInfo, err := proxyAPI.GetSectorInfo(sectorNumber)
        if err == nil {
            ta.panelComponent.UpdateSectorInfo(sectorInfo)
        }
        
        // Get player info and update panel (UpdateTraderInfo shows current player)
        playerInfo, err := proxyAPI.GetPlayerInfo()
        if err == nil {
            ta.panelComponent.UpdateTraderInfo(playerInfo)  // Keep existing method name
        }
    }
}

// UPDATE existing HandleConnectionStatusChanged to add panel API setup
// EXACT LOCATION: Add after line with ta.statusComponent.SetProxyAPI()
func (ta *TwistApp) HandleConnectionStatusChanged(status coreapi.ConnectionStatus, address string) {
    ta.app.QueueUpdateDraw(func() {
        switch status {
        case coreapi.ConnectionStatusConnected:
            ta.connected = true
            ta.serverAddress = address
            ta.menuComponent.SetConnectedMenu()
            ta.statusComponent.SetConnectionStatus(true, address)
            
            // Set ProxyAPI on status component after connection established
            if ta.proxyClient.IsConnected() {
                currentAPI := ta.proxyClient.GetCurrentAPI()
                ta.statusComponent.SetProxyAPI(currentAPI)
                ta.panelComponent.SetProxyAPI(currentAPI)  // ADD this line after existing SetProxyAPI call
            }
            // ... rest unchanged
        }
    })
}
```

### Step 4.3: Add Parser Integration for Sector Changes

**Parser Callback Integration** - Update `internal/streaming/parser/sector.go`:

**Current Parser Flow Analysis**:
- ✅ Parser tracks `sp.ctx.State.CurrentSectorIndex` in `initializeSectorData()` (line 66)
- ✅ Parser saves sector in `SectorCompleted()` after parsing complete (line 136-138)
- ✅ Pipeline has `tuiAPI api.TuiAPI` field for direct callbacks (line 29 in pipeline.go)

**Add Proxy Reference to Parser** - Update `internal/streaming/parser/sector.go`:

```go
// ADD proxy reference to SectorProcessor for state updates
type SectorProcessor struct {
    ctx   *ParserContext
    utils *ParseUtils
    proxy *proxy.Proxy  // ADD: Direct proxy reference for state updates
}

// UPDATE NewSectorProcessor to accept proxy reference
func NewSectorProcessor(ctx *ParserContext, proxyInstance *proxy.Proxy) *SectorProcessor {
    return &SectorProcessor{
        ctx:   ctx,
        utils: NewParseUtils(ctx),
        proxy: proxyInstance,  // Store proxy reference
    }
}

// UPDATE initializeSectorData to notify proxy of sector change (line 64-70 area)
func (sp *SectorProcessor) initializeSectorData(line string) {
    // Extract sector number from "Sector : 1234"
    if idx := strings.Index(line, "Sector :"); idx != -1 {
        sectorStr := strings.TrimSpace(line[idx+8:])
        if spaceIdx := strings.Index(sectorStr, " "); spaceIdx != -1 {
            sectorStr = sectorStr[:spaceIdx]
        }
        
        sectorNum := sp.utils.StrToIntSafe(sectorStr)
        if sectorNum > 0 {
            sp.ctx.State.CurrentSectorIndex = sectorNum
            sp.ctx.State.CurrentSector = &database.TSector{}
            
            // NOTIFY proxy of sector change - triggers TuiAPI callback automatically
            if sp.proxy != nil {
                sp.proxy.SetCurrentSector(sectorNum)
            }
        }
    }
}
```

**Integration Point**: `SetCurrentSector()` automatically triggers `OnCurrentSectorChanged()` callback to TUI.

**Phase 4.4 Complete**: TUI components fully migrated to use API exclusively.

## Testing Phase 4 (Simplified Approach)

### Manual Testing Steps

**Phase 4.1 Testing**:
1. **Dependency Check**: Verify `internal/tui/components/panels.go` no longer imports `internal/database`
2. **Compilation**: Ensure code compiles with new API types
3. **Panel Display**: Verify panels show placeholder text properly

**Phase 4.2 Testing**:
1. **API Methods**: Test `GetSectorInfo()` returns data from database
2. **Data Conversion**: Verify database sector data converts to API format correctly
3. **Error Handling**: Test API methods with invalid sector numbers

**Phase 4.3 Testing**:
1. **State Tracking**: Test `GetCurrentSector()` and `GetPlayerInfo()` return current data
2. **State Updates**: Test `SetCurrentSector()` and `SetPlayerName()` update state
3. **Thread Safety**: Basic concurrency test of state getter/setter methods

**Phase 4.4 Testing**:
1. **Event Flow**: Test parser → callback → TuiAPI → TUI event chain
2. **Panel Updates**: Verify panels update via API when sector changes
3. **Integration**: Test complete flow from parser to UI display

### Key Validation Points

- [ ] **No database imports** in TUI components
- [ ] **API methods work** - GetSectorInfo, GetCurrentSector, GetPlayerInfo
- [ ] **Simple state tracking** - current sector and player name
- [ ] **Event callbacks** - OnCurrentSectorChanged triggers UI updates
- [ ] **Panel components** use API data exclusively
- [ ] **Data conversion** - database types convert to API types correctly

### Expected Behavior

- Panel components get game data via ProxyAPI calls only
- Basic sector tracking works without complex StateManager
- Simple parser callbacks trigger sector change events
- No direct access to database/parser from TUI code
- All API methods follow simple direct delegation pattern

## Success Criteria (Simplified Phase 4)

✅ **🚨 DATABASE DEPENDENCY FIXED**: Remove `"twist/internal/database"` import from TUI  
✅ **Direct Database Access Eliminated**: TUI uses ProxyAPI exclusively  
✅ **Basic Game State API**: GetCurrentSector, GetSectorInfo, GetPlayerInfo work  
✅ **Simple State Tracking**: Basic currentSector and playerName fields in proxy  
✅ **Minimal Events**: OnCurrentSectorChanged callback implemented  
✅ **Panel Migration**: Uses API data types exclusively  
✅ **Simple Direct Delegation**: Follows successful Phase 2-3 patterns  

## Files to Modify Summary

### Core API Files:
- `internal/api/api.go` - Add PlayerInfo, SectorInfo types and extend ProxyAPI/TuiAPI interfaces

### Proxy Changes:
- `internal/proxy/proxy.go` - Add GetDatabase(), currentSector/playerName fields, getter/setter methods
- `internal/proxy/proxy_api_impl.go` - Add game state methods with simple delegation
- `internal/proxy/game_state_converters.go` - Simple conversion functions

### TUI Integration:
- `internal/tui/api/tui_api_impl.go` - Implement OnCurrentSectorChanged handler
- `internal/tui/app.go` - Add HandleCurrentSectorChanged, refreshPanelData methods
- `internal/tui/components/panels.go` - **🚨 CRITICAL: Remove database import, use API types only**

### Optional Parser Integration:
- `internal/streaming/parser/sector.go` - Add simple callback mechanism

## Key Architectural Benefits

**Simplified vs Original Complex Approach**:
- **No StateManager** - Eliminated 300+ lines of complex state management code
- **Direct Delegation** - Consistent with successful Phase 2-3 patterns  
- **Simple State Tracking** - Basic fields in proxy instead of complex caching
- **Lightweight Conversion** - Simple field mapping instead of complex data transformation
- **Incremental Implementation** - Small sub-phases reduce risk
- **Immediate Benefits** - Fixes critical dependency violation in Phase 4.1

## Risk Mitigation

1. **Incremental Sub-Phases**: Each sub-phase is small and testable independently
2. **Simple Patterns**: Uses proven thin orchestration from previous phases  
3. **Direct Database Access**: Leverages existing database performance
4. **Minimal Parser Changes**: Optional parser integration reduces complexity
5. **Consistent Architecture**: Maintains established API design patterns

## Next Steps (Phase 5)

Phase 5 will complete architectural separation:
- Move remaining modules to proxy package (`internal/streaming` → `internal/proxy/streaming`)
- Remove any remaining legacy coupling between modules
- Enforce import restrictions (TUI can only import `internal/api`)
- Add comprehensive integration tests
- Consider advanced features if needed (caching, complex state management, etc.)

This simplified Phase 4 establishes **complete game state management through API** using simple, proven patterns while eliminating the critical dependency violations that compromise the clean API architecture.
