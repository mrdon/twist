# TWX Menu System Implementation Status

This document tracks the implementation status of all TWX menu system commands and identifies missing functionality.

## **TWX Main Menu Commands**

### ✅ **FULLY IMPLEMENTED**
- **(B) Burst Commands** - Complete with all sub-options
  - **(B) Send burst** - ✅ Working, exits menu after sending
  - **(R) Repeat last burst** - ✅ Working, exits menu after sending  
  - **(E) Edit/Send last burst** - ✅ Working, exits menu after sending

### ✅ **IMPLEMENTED** (Basic functionality working)
- **(L) Load Script** - ✅ Prompts for script filename, calls script manager
- **(T) Terminate Script** - ✅ Shows script status, terminates scripts

### ✅ **PARTIALLY IMPLEMENTED** (Some features work, others don't)
- **(S) Script Menu** - ✅ Navigation works, submenu created
  - **(L) Load Script** - ✅ Working (same as main Load Script)
  - **(T) Terminate Script** - ✅ Working (terminates all scripts)
  - **(D) Debug Script** - ✅ Working (shows script engine status)
  - **(V) Variable Dump** - ✅ Working (shows basic script info)
  - **(P) Pause Script** - ❌ **NOT IMPLEMENTED** ("functionality not yet implemented") - ✅ **EASILY IMPLEMENTABLE**
  - **(R) Resume Script** - ❌ **NOT IMPLEMENTED** ("functionality not yet implemented") - ✅ **EASILY IMPLEMENTABLE**

- **(V) View Data Menu** - ✅ Navigation works, submenu created  
  - **(S) Sector Display** - ✅ Working (shows database sector info)
  - **(P) Port List** - ✅ Working (queries database for ports)
  - **(T) Trader List** - ❌ **NOT IMPLEMENTED** ("functionality not yet implemented") - ✅ **EASILY IMPLEMENTABLE**
  - **(R) Route Plot** - ❌ **NOT IMPLEMENTED** ("functionality not yet implemented") - ❌ **COMPLEX**
  - **(B) Bubble Info** - ❌ **NOT IMPLEMENTED** ("functionality not yet implemented") - ❌ **COMPLEX**

### ❌ **NOT IMPLEMENTED**
- **(P) Port Menu** - ❌ **NOT IMPLEMENTED** ("functionality not yet implemented") - ⚠️ **MODERATE COMPLEXITY**

## **Summary Statistics**
- **✅ Fully Working**: 6 commands (Burst Commands + 3 sub-items, Load Script, Terminate Script)
- **✅ Partially Working**: 8 commands (Script Menu + Data Menu sub-items) 
- **❌ Not Implemented**: 6 commands total
  - **✅ Easily Implementable**: 3 commands (Script Pause, Script Resume, Trader List)
  - **⚠️ Moderate Complexity**: 1 command (Port Menu)  
  - **❌ Complex**: 2 commands (Route Plot, Bubble Info)

## **Implementation Details**

### Missing Implementations

#### 1. Script Pause/Resume
**Location**: `internal/proxy/menu/terminal_menu_manager.go:701-716`
```go
func (tmm *TerminalMenuManager) handleScriptPause(item *TerminalMenuItem, params []string) error {
    tmm.sendOutput("Script pause functionality not yet implemented.\r\n")
    tmm.displayCurrentMenu()
    return nil
}

func (tmm *TerminalMenuManager) handleScriptResume(item *TerminalMenuItem, params []string) error {
    tmm.sendOutput("Script resume functionality not yet implemented.\r\n")
    tmm.displayCurrentMenu()
    return nil
}
```

#### 2. Data Menu Items
**Location**: `internal/proxy/menu/terminal_menu_manager.go:876-967`
```go
func (tmm *TerminalMenuManager) handleTraderList(item *TerminalMenuItem, params []string) error {
    tmm.sendOutput("Trader list functionality not yet implemented.\r\n")
    // ...
}

func (tmm *TerminalMenuManager) handleRoutePlot(item *TerminalMenuItem, params []string) error {
    tmm.sendOutput("Route plot functionality not yet implemented.\r\n")
    // ...
}

func (tmm *TerminalMenuManager) handleBubbleInfo(item *TerminalMenuItem, params []string) error {
    tmm.sendOutput("Bubble info functionality not yet implemented.\r\n")
    // ...
}
```

#### 3. Port Menu
**Location**: `internal/proxy/menu/terminal_menu_manager.go:499-511`
```go
func (tmm *TerminalMenuManager) handlePortMenu(item *TerminalMenuItem, params []string) error {
    tmm.sendOutput("Port menu functionality not yet implemented.\r\n")
    tmm.displayCurrentMenu()
    return nil
}
```

## **Implementation Priority Recommendations**

### **High Priority**
1. **Port Menu** - Major missing feature from main menu
2. **Script Pause/Resume** - Important for script debugging workflow

### **Medium Priority** 
3. **Trader List** - Data display feature, likely has database backend support
4. **Route Plot** - Navigation assistance feature

### **Low Priority**
5. **Bubble Info** - Specialized data analysis feature

## **Technical Notes**

- All menu handlers follow consistent error handling patterns with panic recovery
- Database interface is available via `tmm.proxyInterface.GetDatabase()`
- Existing data menu items (Sector Display, Port List) show how to query database
- Script manager interface is available via `tmm.proxyInterface.GetScriptManager()`
- Menu system supports input collection for user prompts
- Display formatting utilities available in `internal/proxy/menu/display` package

## **Implementation Feasibility Research**

Based on code analysis, here are the implementation feasibility findings for missing features:

### **Script Pause/Resume** ✅ **EASILY IMPLEMENTABLE**
- **VM Layer Support**: Virtual machines already have `Pause()`, `IsPaused()`, `SetPaused()` methods in `internal/proxy/scripting/vm/vm.go:190-192` and `internal/proxy/scripting/vm/state.go:37-65`
- **Command Support**: PAUSE command already exists in script engine (`internal/proxy/scripting/vm/commands/game.go:14,65-66`)
- **Implementation**: ScriptManager needs `PauseAllScripts()` and `ResumeAllScripts()` methods to iterate through running scripts and call VM pause/resume
- **Estimated Effort**: 30-50 lines of code in ScriptManager

### **Trader List** ✅ **EASILY IMPLEMENTABLE**  
- **Database Support**: `traders` table exists with schema in `internal/proxy/database/schema.go`
- **Data Structure**: `TTrader` struct defined in `internal/proxy/database/structs.go:81-87` with Name, ShipType, ShipName, Figs
- **Query Pattern**: Can follow existing `handleSectorDisplay` and `handlePortList` patterns using database queries
- **Implementation**: Query all traders across sectors and format output
- **Estimated Effort**: 40-60 lines of code following existing display patterns

### **Port Menu** ⚠️ **MODERATE COMPLEXITY**
- **Database Support**: `TPort` struct and port data available via `LoadPort()` method
- **Missing**: Need to define what Port Menu functionality includes (port trading interface, port info display, etc.)
- **Dependency**: Requires understanding TWX Port Menu feature set
- **Estimated Effort**: 100-200 lines depending on scope

### **Route Plot** ❌ **COMPLEX IMPLEMENTATION**
- **Missing Components**: No pathfinding or route calculation algorithms exist
- **Required**: Shortest path algorithm (Dijkstra/A*), warp network analysis
- **Database**: Sector warp data available but needs graph algorithms
- **Estimated Effort**: 300+ lines for proper pathfinding implementation

### **Bubble Info** ❌ **COMPLEX IMPLEMENTATION**  
- **Missing Components**: No bubble detection or space analysis algorithms
- **Required**: Sector grouping logic, deadend detection, strategic analysis
- **Estimated Effort**: 200+ lines for bubble analysis algorithms

## **Updated Priority Recommendations**

### **High Priority - Easy Wins**
1. **Script Pause/Resume** - VM support exists, just need ScriptManager methods
2. **Trader List** - Database support exists, follow existing query patterns

### **Medium Priority** 
3. **Port Menu** - Data available, need to scope functionality

### **Low Priority - Complex**
4. **Route Plot** - Requires pathfinding algorithms
5. **Bubble Info** - Requires space analysis algorithms

## **Next Steps**

1. ✅ Research database schema and available query methods - COMPLETED
2. ✅ Examine script manager interface for pause/resume capabilities - COMPLETED
3. Implement Script Pause/Resume functionality (easiest implementation)
4. Implement Trader List functionality (second easiest)
5. Research TWX Port Menu requirements before implementation
6. Test implementations against TWX compatibility requirements