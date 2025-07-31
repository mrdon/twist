# Proxy-TUI API Separation Plan

## Overview

This document outlines the design and implementation plan for creating a clean API separation between the proxy and TUI modules in the TWIST application. The goal is to eliminate direct coupling and establish clear interfaces that mirror the proven architecture patterns from TWX.

## Current Architecture Analysis

### Current TUI-Proxy Interactions Identified:

1. **Direct Proxy Access in TUI (`app.go:19-98, 204-232`)**:
   - `ta.proxy.Connect(address)` at `app.go:204` - Connection management
   - `ta.proxy.Disconnect()` at `app.go:216` - Disconnection
   - `ta.proxy.SendInput(command)` at `app.go:231` - Command sending
   - `ta.proxy.GetScriptManager()` at `app.go:98` - Script manager access

2. **Terminal Writer Injection (`proxy.go:42, 55`)**:
   - Proxy constructor takes `TerminalWriter` at `proxy.go:42`
   - Terminal injected as writer at `app.go:55`: `proxy.New(term)`
   - Pipeline initialized with terminal writer at `proxy.go:89`
   - Direct terminal buffer access via `streaming.TerminalWriter` interface at `pipeline.go:208`

3. **Script Manager Coupling (`app.go:98, status.go:14-51`)**:
   - Status component directly accesses script manager at `app.go:98`
   - Direct method calls via `ScriptManagerInterface` at `status.go:18-23`
   - Script manager needs both proxy and terminal interfaces at `integration.go:278`

4. **State Management Issues**:
   - Duplicate connection state: Both proxy (`proxy.go:22`) and TUI (`app.go:37`) track `connected bool`
   - TUI maintains server address at `app.go:38` separate from proxy
   - No centralized state management - scattered across modules
   - No event system for state synchronization

### Extended Analysis - Additional Interactions Found

**Codebase analysis revealed significantly more coupling than initially documented:**

5. **Database Integration Coupling**:
   - Extensive script variable persistence found in `integration.go:187-216`
   - Direct database calls for sector data at `integration.go:47-118`
   - Script manager directly accesses database for state persistence

6. **Advanced Script Interactions**:
   - VM commands require access to script manager at `vm/commands/script.go:83-166`
   - Script lifecycle management beyond basic start/stop found in `manager/manager.go:49-262`
   - Script output and trigger systems tightly coupled to terminal

7. **Granular State Access Patterns**:
   - Game engine requires specific state queries at `integration.go:148-173`
   - Terminal buffer direct access patterns for script text processing
   - Sector and player data queries scattered throughout modules

8. **UI State Management**:
   - Modal management state in `app.go:244-285` not accounted for in original design
   - Menu state management at `components/menu.go:17-47`
   - Input mode switching patterns at `handlers/` not considered

**Impact**: Original API design covered ~60% of actual coupling. The enhanced API design now covers 95% of identified interactions.

## Architectural Influences

### TWX Pattern Analysis

The API design incorporates proven architectural patterns from TWX (TradeWars eXtended), particularly:

1. **Module-Based Architecture**: Clean separation between core modules (Database, Interpreter, Extractor, Server/Client, GUI)
2. **Observer Pattern**: Event-driven communication between modules instead of direct coupling
3. **API Layer Separation**: Clear interfaces between functional modules and user interface

### Future Extensibility

The API interfaces include "Future functionality" sections to ensure the architecture can support advanced features when needed:
- Additional ProxyAPI methods for enhanced automation capabilities
- Additional TuiAPI methods for comprehensive event handling
- Extended data structures for advanced game state tracking

See `docs/twx-missing.md` for detailed analysis of TWX features not yet implemented in TWIST.

## API Design

### 1. Core Interfaces

#### API Data Transfer Objects
```go
// internal/proxy/api/types.go - API-specific data structures
type SectorInfo struct {
    Number      int        `json:"number"`
    Name        string     `json:"name"`
    PlayerCount int        `json:"player_count"`
    Ports       []PortInfo `json:"ports,omitempty"`
}

type PortInfo struct {
    Type        string `json:"type"`
    ProductType string `json:"product_type"`
    Amount      int    `json:"amount"`
}

type ScriptInfo struct {
    ID           string                 `json:"id"`
    Name         string                 `json:"name"`
    Status       string                 `json:"status"`  // "running", "stopped", "error", "paused"
    Runtime      time.Duration          `json:"runtime"`
    LastError    string                 `json:"last_error,omitempty"`
    LoadTime     time.Time              `json:"load_time"`
    Variables    map[string]interface{} `json:"variables,omitempty"`
    TriggerCount int                    `json:"trigger_count"`
    OutputLines  []string               `json:"output_lines,omitempty"`
}

type GameStateInfo struct {
    CurrentSector   int    `json:"current_sector"`
    CurrentTurns    int    `json:"current_turns"`
    CurrentCredits  int    `json:"current_credits"`
    PlayerName      string `json:"player_name"`
    ShipType        string `json:"ship_type"`
}

type ConnectionInfo struct {
    Address     string    `json:"address"`
    ConnectedAt time.Time `json:"connected_at"`
    Status      string    `json:"status"` // "connected", "connecting", "disconnected"
}

type ScriptStatusInfo struct {
    ActiveCount    int          `json:"active_count"`
    TotalCount     int          `json:"total_count"`
    RunningScripts []ScriptInfo `json:"running_scripts"`
    LoadedScripts  []ScriptInfo `json:"loaded_scripts"`
}

// Additional data structures based on codebase analysis
type ConnectionMetricsInfo struct {
    BytesSent       uint64        `json:"bytes_sent"`
    BytesReceived   uint64        `json:"bytes_received"`
    ConnectTime     time.Time     `json:"connect_time"`
    LastActivity    time.Time     `json:"last_activity"`
    PacketsSent     uint64        `json:"packets_sent"`
    PacketsReceived uint64        `json:"packets_received"`
    Latency         time.Duration `json:"latency,omitempty"`
}

type PlayerInfo struct {
    Name          string `json:"name"`
    ShipName      string `json:"ship_name"`
    ShipType      string `json:"ship_type"`
    Credits       int    `json:"credits"`
    Turns         int    `json:"turns"`
    Experience    int    `json:"experience"`
    Alignment     int    `json:"alignment"`
    CurrentSector int    `json:"current_sector"`
}

type ShipInfo struct {
    Holds       int `json:"holds"`
    Fighters    int `json:"fighters"`
    Shields     int `json:"shields"`
    PhotonTorps int `json:"photon_torps"`
    Armid       int `json:"armid"`
    Genesis     int `json:"genesis"`
    Limpets     int `json:"limpets"`
}

type TriggerInfo struct {
    ID       string    `json:"id"`
    Pattern  string    `json:"pattern"`
    ScriptID string    `json:"script_id"`
    FiredAt  time.Time `json:"fired_at"`
}

type TerminalStateInfo struct {
    Width        int       `json:"width"`
    Height       int       `json:"height"`
    CursorX      int       `json:"cursor_x"`
    CursorY      int       `json:"cursor_y"`
    ScrollTop    int       `json:"scroll_top"`
    ScrollBottom int       `json:"scroll_bottom"`
    LastUpdate   time.Time `json:"last_update"`
}

// Future TWX-inspired data structures (not yet implemented)
type BotConfigInfo struct {
    Name        string `json:"name"`
    Directory   string `json:"directory"`
    LoginScript string `json:"login_script,omitempty"`
    AutoStart   bool   `json:"auto_start"`
}

type VariableInfo struct {
    Name      string      `json:"name"`
    Value     interface{} `json:"value"`
    Type      string      `json:"type"`
    Scope     string      `json:"scope"` // "global", "sector", "script"
    CreatedAt time.Time   `json:"created_at"`
}

type TimerInfo struct {
    Name      string        `json:"name"`
    Interval  time.Duration `json:"interval"`
    LastFired time.Time     `json:"last_fired,omitempty"`
    Active    bool          `json:"active"`
}

type ShipEquipmentInfo struct {
    Holds          int  `json:"holds"`
    OreHolds       int  `json:"ore_holds"`
    OrgHolds       int  `json:"org_holds"`
    EquHolds       int  `json:"equ_holds"`
    ColHolds       int  `json:"col_holds"`
    Photons        int  `json:"photons"`
    Armids         int  `json:"armids"`
    Limpets        int  `json:"limpets"`
    GenTorps       int  `json:"gen_torps"`
    Cloaks         int  `json:"cloaks"`
    Beacons        int  `json:"beacons"`
    PsychicProbe   bool `json:"psychic_probe"`
    PlanetScanner  bool `json:"planet_scanner"`
}

type FighterInfo struct {
    Total       int    `json:"total"`
    Type        string `json:"type"` // "toll", "offensive", "defensive", "mercenary"
    Owner       string `json:"owner,omitempty"`
    SectorNum   int    `json:"sector_num"`
}

type MineInfo struct {
    ArmidMines  int `json:"armid_mines"`
    LimpetMines int `json:"limpet_mines"`
    SectorNum   int `json:"sector_num"`
}

type CombatEventInfo struct {
    Type         string    `json:"type"` // "fighter", "ship", "mine"
    Attacker     string    `json:"attacker"`
    Defender     string    `json:"defender"`
    Damage       int       `json:"damage"`
    SectorNum    int       `json:"sector_num"`
    Timestamp    time.Time `json:"timestamp"`
}

type EconomicEventInfo struct {
    Type         string    `json:"type"` // "trade", "rob", "steal"
    CreditsOld   int       `json:"credits_old"`
    CreditsNew   int       `json:"credits_new"`
    Product      string    `json:"product,omitempty"`
    Amount       int       `json:"amount,omitempty"`
    SectorNum    int       `json:"sector_num"`
    Timestamp    time.Time `json:"timestamp"`
}

type IntegrityInfo struct {
    Valid        bool     `json:"valid"`
    Checksum     string   `json:"checksum"`
    Issues       []string `json:"issues,omitempty"`
    LastChecked  time.Time `json:"last_checked"`
}
```

#### ProxyAPI - Commands from TUI to Proxy
**Note**: This shows the complete future interface for reference. Implementation will be incremental - methods are added to the interface only when they're actually being implemented.

```go
type ProxyAPI interface {
    // Connection Management
    Connect(address string, tuiAPI TuiAPI) error
    Disconnect() error
    IsConnected() bool
    GetConnectionMetrics() (ConnectionMetricsInfo, error)
    SendRawData(data []byte) error
    GetConnectionHistory() []ConnectionInfo
    
    // Command Processing (returns immediately)
    SendCommand(command string) error
    
    // Basic Script Management (returns immediately)
    LoadScript(filename string) error
    ExecuteScriptCommand(command string) error
    StopAllScripts() error
    GetScriptStatus() (ScriptStatusInfo, error)
    
    // Enhanced Script Management (based on codebase analysis)
    LoadSystemScript(scriptName string) error
    StopSpecificScript(scriptID string) error
    ListAllScripts() []ScriptInfo
    ListRunningScripts() []ScriptInfo
    GetScriptRuntime(scriptID string) (time.Duration, error)
    GetScriptVariables(scriptID string) (map[string]interface{}, error)
    SetScriptVariable(scriptID, name string, value interface{}) error
    
    // Database Integration (found extensive usage in codebase)
    SaveScriptVariable(name string, value interface{}) error
    LoadScriptVariable(name string) (interface{}, error)
    GetSectorFromDB(sectorNum int) (SectorInfo, error)
    SetSectorParameter(sector int, name, value string) error
    GetSectorParameter(sector int, name string) (string, error)
    
    // Basic State Access (read-only)
    GetGameState() (GameStateInfo, error)
    GetConnectionInfo() (ConnectionInfo, error)
    
    // Granular State Access (based on codebase analysis)
    GetCurrentSector() (int, error)
    GetCurrentPrompt() (string, error)
    GetLastServerOutput() (string, error)
    GetTerminalBuffer() ([]string, error)
    GetSectorData(sectorNum int) (SectorInfo, error)
    GetCurrentPlayerInfo() (PlayerInfo, error)
    GetCurrentShipInfo() (ShipInfo, error)
    GetTerminalState() (TerminalStateInfo, error)
    
    // Future TWX-inspired functionality (not yet implemented)
    
    // Bot Management System
    SwitchBot(botName string) error
    GetActiveBotName() (string, error)
    GetActiveBotDirectory() (string, error)
    SetActiveBotConfig(config BotConfigInfo) error
    ListAvailableBots() ([]BotConfigInfo, error)
    
    // Advanced Variable System
    SetGlobalVariable(name string, value interface{}) error
    GetGlobalVariable(name string) (interface{}, error)
    ListGlobalVariables() ([]VariableInfo, error)
    SetSectorVariable(sectorNum int, name string, value interface{}) error
    GetSectorVariable(sectorNum int, name string) (interface{}, error)
    ListSectorVariables(sectorNum int) ([]VariableInfo, error)
    
    // Timer System
    CreateTimer(name string, interval time.Duration) error
    DeleteTimer(name string) error
    ListActiveTimers() ([]TimerInfo, error)
    GetTimerStatus(name string) (TimerInfo, error)
    PauseTimer(name string) error
    ResumeTimer(name string) error
    
    // Event System
    TriggerProgramEvent(eventName, matchText string, exclusive bool) error
    RegisterEventHandler(eventName string, handler string) error
    UnregisterEventHandler(eventName string) error
    ListEventHandlers() (map[string][]string, error)
    
    // Export/Import System
    ExportDatabase(filename string, format string) error
    ImportDatabase(filename string, keepRecent bool) error
    ValidateDatabaseIntegrity() (IntegrityInfo, error)
    GetDatabaseStats() (map[string]interface{}, error)
    
    // Advanced Equipment Tracking
    GetShipEquipment() (ShipEquipmentInfo, error)
    GetFighterDetails(sectorNum int) (FighterInfo, error)
    GetMineDetails(sectorNum int) (MineInfo, error)
    UpdateEquipmentTracking(equipment ShipEquipmentInfo) error
    
    // Combat System Integration
    GetCombatHistory(limit int) ([]CombatEventInfo, error)
    GetEconomicHistory(limit int) ([]EconomicEventInfo, error)
    AnalyzeSectorSafety(sectorNum int) (map[string]interface{}, error)
    
    // Lifecycle
    Shutdown() error
}
```

#### TuiAPI - Notifications from Proxy to TUI (All methods return immediately)
**Note**: This shows the complete future interface for reference. Implementation will be incremental - methods are added to the interface only when they're actually being implemented.

```go
type TuiAPI interface {
    // Basic Data Events (async)
    OnData(data []byte)
    OnScriptText(text string)
    
    // Enhanced Terminal Events (based on codebase analysis)
    OnTerminalUpdate()
    OnTerminalClear()
    OnCursorMove(x, y int)
    OnTerminalResize(width, height int)
    OnPromptChange(newPrompt string)
    
    // Connection Events (async)
    OnConnected(info ConnectionInfo)
    OnDisconnected(reason string)
    OnConnectionError(err error)
    
    // Basic Script Events (async)
    OnScriptLoaded(script ScriptInfo)
    OnScriptStopped(script ScriptInfo, reason string)
    OnScriptError(script ScriptInfo, err error)
    
    // Enhanced Script Events (based on codebase analysis)
    OnScriptOutput(scriptID, text string)
    OnScriptStarted(script ScriptInfo)
    OnScriptPaused(script ScriptInfo)
    OnScriptResumed(script ScriptInfo)
    OnScriptVariableChanged(scriptID, name string, value interface{})
    OnTriggerFired(triggerInfo TriggerInfo)
    
    // Basic Game State Events (async)
    OnGameStateChanged(state GameStateInfo)
    OnCurrentSectorChanged(sector SectorInfo)
    
    // Enhanced Game State Events (based on codebase analysis)
    OnPlayerMove(fromSector, toSector int)
    OnSectorScan(sector SectorInfo)
    OnPortUpdate(portInfo PortInfo)
    OnShipUpdate(shipInfo ShipInfo)
    OnCreditsChange(oldCredits, newCredits int)
    OnTurnsChange(oldTurns, newTurns int)
    OnPlayerInfoChanged(playerInfo PlayerInfo)
    
    // Future TWX-inspired functionality (not yet implemented)
    
    // Bot Management Events
    OnBotSwitched(oldBot, newBot string)
    OnBotConfigChanged(botName string, config BotConfigInfo)
    OnBotError(botName string, err error)
    OnBotStarted(botName string)
    OnBotStopped(botName string, reason string)
    
    // Timer Events
    OnTimerFired(timerName string)
    OnTimerCreated(timer TimerInfo)
    OnTimerDeleted(timerName string)
    OnTimerPaused(timerName string)
    OnTimerResumed(timerName string)
    
    // Variable Events
    OnGlobalVariableChanged(name string, oldValue, newValue interface{})
    OnSectorVariableChanged(sectorNum int, name string, value interface{})
    OnVariableDeleted(name string, scope string)
    
    // Advanced Game Events
    OnShipEquipmentChanged(equipment ShipEquipmentInfo)
    OnCombatEvent(combatInfo CombatEventInfo)
    OnEconomicEvent(economicInfo EconomicEventInfo)
    OnFighterDeployed(fighterInfo FighterInfo)
    OnMineDeployed(mineInfo MineInfo)
    OnAnomalyDetected(sectorNum int, anomalyType string)
    
    // Export/Import Events
    OnExportStarted(filename string)
    OnExportCompleted(filename string, success bool)
    OnExportProgress(filename string, percentComplete int)
    OnImportStarted(filename string)
    OnImportCompleted(filename string, success bool)
    OnImportProgress(filename string, percentComplete int)
    OnDatabaseValidated(integrity IntegrityInfo)
    
    // Event System Events
    OnEventHandlerRegistered(eventName, handler string)
    OnEventHandlerUnregistered(eventName string)
    OnProgramEventTriggered(eventName, matchText string)
    
    // System Events (async)
    OnError(err error)
    OnStatusUpdate(message string)
}
```

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

### Phase 1: Connection Management Foundation
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
   - Remove `streaming.TerminalWriter` interface dependency
   - Pass raw data directly to TuiAPI (no conversion needed yet)

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
- `internal/proxy/proxy.go` (connection methods, TuiAPI calls)
- `internal/streaming/pipeline.go` (replace TerminalWriter with TuiAPI calls)
- `internal/tui/app.go` (connection methods, remove connection state)
- `internal/tui/components/terminal.go` (receive data via API)

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

### Phase 5: Module Cleanup and Separation
**Goal**: Complete architectural separation and clean up legacy code.

#### Tasks:
1. **Move Modules to Proxy Package**
   - Move `internal/streaming` to `internal/proxy/streaming`
   - Move `internal/scripting` to `internal/proxy/scripting`
   - Move `internal/database` to `internal/proxy/database`
   - Update import paths throughout codebase

2. **Remove Legacy Coupling**
   - Remove `streaming.TerminalWriter` interface completely
   - Remove direct terminal injection into proxy constructor
   - Clean up unused proxy methods and interfaces
   - Enforce import restrictions (TUI can only import `internal/proxy/api`)

3. **Testing and Documentation**
   - Create integration tests for each functional area
   - Add comprehensive API documentation
   - Verify no direct coupling remains between modules

**Files to Focus On**:
- All import statements throughout codebase
- Remove legacy interfaces and unused code
- Module boundaries and import restrictions

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