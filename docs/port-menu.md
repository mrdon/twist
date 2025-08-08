# TWX Terminal Menu System Port Plan

## Overview

This plan outlines porting TWX's terminal-based menu system to Twist, building on the existing TUI architecture. The terminal menu system is activated by the '$' character during game sessions and provides hierarchical text-based menus for in-game interaction, separate from the existing GUI menu system.

## Analysis of Existing Architecture

### Current TUI System (`internal/tui/`)
Twist already has a comprehensive GUI menu system:
- **MenuComponent**: Top menu bar with Session/Edit/View/Terminal/Help
- **DropdownMenu**: GUI dropdown menus with keyboard shortcuts
- **TwistMenu**: Custom styled menu components
- **GlobalShortcutManager**: Application-wide keyboard shortcuts
- **InputHandler**: Multi-modal input processing
- **TerminalComponent**: Full ANSI terminal emulation with color support

### Current Proxy System (`internal/proxy/`)
- **TWXParser.ProcessOutBound()**: Already processes outbound data (line 2295)
- **Script command system**: 100+ TWX commands implemented in `vm/commands/`
- **ScriptManager**: Direct script execution via `ExecuteCommand()`
- **Pipeline**: Data streaming with event system

### Integration Points
- **TuiAPI**: Interface between proxy and TUI
- **Event system**: Observer pattern for data flow
- **ANSI support**: Complete terminal color/control sequence handling

### No API Extensions Required

The terminal menu system operates entirely within the proxy layer, just like the original TWX implementation. Menu operations are handled by intercepting and modifying the normal data flow between client and server.

**Key Architectural Principles:**
- **Proxy-Only Operation**: Terminal menus exist only in the proxy layer
- **Data Stream Integration**: Menu output is injected into the normal game data stream  
- **Input Interception**: User input is intercepted and processed by menu system when active
- **Transparent to TUI**: TUI layer sees menu output as normal game text with ANSI formatting
- **No API Changes**: Existing `OnData()` and `SendData()` are sufficient

## Required New Components

### 1. Terminal Menu System (`internal/proxy/menu/`)

#### 1.1 Terminal Menu Manager (`terminal_menu.go`)
**Separate from GUI menus** - handles in-game terminal menus triggered by '$'

```go
type TerminalMenuManager struct {
    currentMenu     *TerminalMenuItem
    activeMenus     map[string]*TerminalMenuItem
    menuKey         rune  // default '$'
    inputBuffer     string
    isActive        bool
    proxy          *Proxy  // For injecting data into stream
}

func (tmm *TerminalMenuManager) ProcessMenuKey(data string) bool
func (tmm *TerminalMenuManager) MenuText(input string) error  
func (tmm *TerminalMenuManager) ActivateMainMenu() error
func (tmm *TerminalMenuManager) InjectOutput(text string)
```

#### 1.2 Terminal Menu Items (`terminal_menu_item.go`) 
**Different from GUI MenuItems** - these are for terminal-based interaction

```go
type TerminalMenuItem struct {
    Name        string
    Description string
    Hotkey      rune
    Parent      *TerminalMenuItem
    Children    []*TerminalMenuItem
    Handler     TerminalMenuHandler
    Parameters  []string
    Reference   string
    Prompt      string
    CloseMenu   bool
    ScriptOwner string  // Script ID that owns this menu
}

type TerminalMenuHandler func(*TerminalMenuItem, []string) error
```

#### 1.3 Built-in Terminal Menu Categories (`categories.go`)
```go
var (
    TWX_MAIN     = "Main Menu"
    TWX_SCRIPT   = "Script Menu" 
    TWX_DATA     = "Data Menu"
    TWX_PORT     = "Port Menu"
    TWX_SETUP    = "Setup Menu"
    TWX_DATABASE = "Database Menu"
)
```

### 2. Terminal Menu Display (`internal/proxy/menu/display/`)

#### 2.1 ANSI Terminal Output (`ansi_output.go`)
Uses existing ANSI support but formats for terminal display:

```go
const (
    MENU_LIGHT = "\x1b[37m"  // ANSI_15 - white
    MENU_MID   = "\x1b[36m"  // ANSI_10 - cyan  
    MENU_DARK  = "\x1b[32m"  // ANSI_2 - dark green
)

func FormatMenuPrompt(prompt, line string) string
func FormatMenuOptions(items []*TerminalMenuItem) string
func ClearMenuLine() string
```

#### 2.2 Terminal Menu Renderer (`renderer.go`)
```go
type TerminalMenuRenderer struct {
    manager *TerminalMenuManager
}

func (r *TerminalMenuRenderer) RenderMenu(menu *TerminalMenuItem)
func (r *TerminalMenuRenderer) RenderPrompt(prompt, input string) 
func (r *TerminalMenuRenderer) RenderOptions(items []*TerminalMenuItem)
```

### 3. Integration with Existing Systems

#### 3.1 Proxy Integration (`internal/proxy/proxy.go`)
Add terminal menu manager to existing Proxy struct:

```go
type Proxy struct {
    // ... existing fields ...
    terminalMenuManager *menu.TerminalMenuManager
}

// Add method to inject menu output into data stream
func (p *Proxy) InjectInboundData(data []byte) {
    // Send through normal pipeline - TUI receives via OnData()
    if p.pipeline != nil {
        p.pipeline.ProcessInbound(data)
    }
}

// Modify SendData to intercept user input for menus
func (p *Proxy) SendData(data []byte) error {
    dataStr := string(data)
    
    // Check if terminal menu is active and should handle input
    if p.terminalMenuManager != nil && p.terminalMenuManager.IsActive() {
        return p.terminalMenuManager.MenuText(dataStr)
    }
    
    // Normal game input processing
    return p.sendToServer(data)
}
```

#### 3.2 TWXParser Integration (`internal/proxy/streaming/twx_parser.go`)
Modify `ProcessOutBound()` to detect menu key:

```go
func (p *TWXParser) ProcessOutBound(data string) bool {
    // Check for terminal menu activation before sending to server
    if strings.Contains(data, "$") && p.proxy != nil && p.proxy.terminalMenuManager != nil {
        // Activate menu and suppress sending '$' to server
        if p.proxy.terminalMenuManager.ProcessMenuKey(data) {
            return false // Don't send '$' to server
        }
    }
    
    // ... existing outbound processing ...
    return true // Send to server normally
}
```

### 4. Script Command Integration (`internal/proxy/scripting/vm/commands/`)

#### 4.1 New Menu Commands (`menu.go`)
Add to existing command registry:

```go
// In registry.go
func RegisterMenuCommands(registry *CommandRegistry) {
    registry.Register("ADDMENU", CmdAddMenu)
    registry.Register("OPENMENU", CmdOpenMenu)  
    registry.Register("CLOSEMENU", CmdCloseMenu)
    registry.Register("GETMENUVALUE", CmdGetMenuValue)
    registry.Register("SETMENUVALUE", CmdSetMenuValue)
    registry.Register("SETMENUHELP", CmdSetMenuHelp)
    registry.Register("SETMENUOPTIONS", CmdSetMenuOptions)
    registry.Register("SETMENUKEY", CmdSetMenuKey)
}

func CmdAddMenu(vm *VM, params []Parameter) error {
    // Implementation matches TWX ADDMENU behavior
    return vm.terminalMenuManager.AddCustomMenu(params)
}
```

## Implementation Plan

### Phase 1: Core Terminal Menu Framework
**Agent Task**: Create foundational menu data structures and basic manager

**Deliverables:**
- `internal/proxy/menu/terminal_menu_item.go` - TerminalMenuItem struct with all properties
- `internal/proxy/menu/terminal_menu_manager.go` - TerminalMenuManager with basic state management
- `internal/proxy/menu/display/ansi_output.go` - ANSI color constants and formatting functions
- Unit tests for menu item creation and basic manager operations

**Acceptance Criteria:**
- TerminalMenuItem struct matches TWX MenuItem capabilities
- TerminalMenuManager can create, store, and retrieve menu items
- ANSI output functions produce correct color codes
- All tests pass

**Files to Create:**
```
internal/proxy/menu/
├── terminal_menu_item.go
├── terminal_menu_manager.go
├── display/
│   └── ansi_output.go
└── menu_test.go
```

### Phase 2: Menu Key Detection & Data Stream Integration
**Agent Task**: Integrate terminal menu system with proxy data stream processing

**Deliverables:**
- Modify `internal/proxy/proxy.go` to include TerminalMenuManager
- Modify `internal/proxy/streaming/twx_parser.go` ProcessOutBound() to detect '$'
- Add input routing logic in proxy to distinguish menu input vs game input
- Implement data stream injection for menu output

**Acceptance Criteria:**
- '$' character in outbound data activates terminal menu system
- Menu input is intercepted by proxy and routed to TerminalMenuManager
- Menu output is injected into inbound data stream and appears in TUI terminal
- Game input bypasses menu system when no menu is active
- No existing functionality is broken

**Files to Modify:**
```
internal/proxy/proxy.go (add terminalMenuManager field, modify input/output routing)
internal/proxy/streaming/twx_parser.go (modify ProcessOutBound to detect '$')
internal/proxy/menu/terminal_menu_manager.go (output via data stream injection)
```

### Phase 3: TWX_MAIN Menu Implementation
**Agent Task**: Implement the main terminal menu with core system functions

**Deliverables:**
- `internal/proxy/menu/categories.go` - Built-in menu definitions
- `internal/proxy/menu/handlers/main_menu.go` - TWX_MAIN menu handlers
- Menu activation on '$' key press
- Basic navigation with hotkeys (B, C, D, Q, etc.)
- Menu display with ANSI formatting matching TWX

**Acceptance Criteria:**
- '$' opens TWX_MAIN menu with proper ANSI display
- Hotkey navigation works (B for burst, C for connect, etc.)
- Menu options display matches TWX formatting exactly
- 'Q' exits menu and returns to game
- '?' shows help for current menu

**Menu Structure to Implement:**
```
TWX_MAIN:
├── (B)urst Commands
├── (L)oad Script
├── (T)erminate Script
├── (S)cript Menu
├── (V)iew Data Menu
├── (P)ort Menu
├── (Q)uit Menu
└── (?)Help
```

**Note**: Connection management (Connect/Disconnect) is handled by the TUI layer through the GUI menus, not the terminal menu system.

### Phase 4: Script Integration Menus
**Agent Task**: Implement TWX_SCRIPT and database-related menus

**Deliverables:**
- `internal/proxy/menu/handlers/script_menu.go` - Script control menu handlers
- `internal/proxy/menu/handlers/data_menu.go` - Database query menu handlers
- Integration with existing ScriptManager for script operations
- Integration with existing database system for queries

**Acceptance Criteria:**
- TWX_SCRIPT menu provides script loading, termination, debug functions
- TWX_DATA menu shows sectors, traders, routes from game database
- Script operations work through existing ScriptManager API
- Database queries display properly formatted results
- Menu navigation between categories works correctly

**Menu Structures to Implement:**
```
TWX_SCRIPT:
├── (L)oad Script
├── (T)erminate Script  
├── (P)ause Script
├── (R)esume Script
├── (D)ebug Script
├── (V)ariable Dump
└── (Q)Back to Main

TWX_DATA:
├── (S)ector Display
├── (T)rader List
├── (P)ort List
├── (R)oute Plot
├── (B)ubble Info
└── (Q)Back to Main
```

### Phase 5: Script Command Interface
**Agent Task**: Implement the 8 TWX script commands for menu manipulation

**Deliverables:**
- `internal/proxy/scripting/vm/commands/menu.go` - Menu script commands
- Register commands in existing command registry
- Integration with TerminalMenuManager for custom menus
- Two-stage input collection system

**Acceptance Criteria:**
- All 8 menu commands work: ADDMENU, OPENMENU, CLOSEMENU, GETMENUVALUE, SETMENUVALUE, SETMENUHELP, SETMENUOPTIONS, SETMENUKEY
- Scripts can create custom menus that appear in terminal
- Custom menus support hotkeys, descriptions, and handlers
- Two-stage input collection works for complex menu operations
- Script-created menus are cleaned up when scripts terminate

**Commands to Implement:**
```go
ADDMENU parent, name, description, hotkey, reference, prompt, closeMenu
OPENMENU menuName
CLOSEMENU menuName  
GETMENUVALUE menuName
SETMENUVALUE menuName, value
SETMENUHELP menuName, helpText
SETMENUOPTIONS menuName, options
SETMENUKEY newKey
```

### Phase 6: Advanced Features & Polish
**Agent Task**: Implement advanced menu features and ensure TWX compatibility

**Deliverables:**
- `internal/proxy/menu/input/collector.go` - Two-stage input collection
- `internal/proxy/menu/help_system.go` - Contextual help system
- Menu cleanup on script termination
- Error handling and input validation
- Performance optimization

**Acceptance Criteria:**
- Two-stage input collection works for complex operations
- Help system shows contextual help with '?' key
- Menus are properly cleaned up when scripts end
- Invalid input is handled gracefully with error messages
- Menu operations don't cause noticeable game latency
- Memory usage is reasonable and doesn't leak

**Additional Features:**
- Menu parameter validation
- Menu state persistence across connections
- Menu history and navigation breadcrumbs
- Menu accessibility features (screen reader compatible ANSI)

## Implementation Guidelines for Agents

### Code Standards
- Follow existing Twist coding patterns and directory structure
- Use existing debug.Log() pattern for debugging (remove before final commits)
- Maintain TWX compatibility - match behavior exactly
- Write comprehensive tests for each component
- Use existing error handling patterns

### Integration Points
- **Data Stream Integration**: Menu output injected into normal inbound data flow
- **Input Interception**: Menu input intercepted in proxy before reaching server
- **Existing Systems**: Integrate with ScriptManager, database, and event bus within proxy layer
- **No API Changes**: Use existing OnData()/SendData() for all communication
- **Transparent Operation**: TUI layer unaware of menu system existence

### Testing Requirements
- Unit tests for all new components
- Integration tests for menu activation and navigation
- Test script command functionality
- Test database integration
- Performance testing for menu operations

### Documentation
- Add godoc comments to all public functions
- Update relevant documentation files
- Include examples of menu usage
- Document any breaking changes or new dependencies

## Key Implementation Details

### Terminal Menu vs GUI Menu Separation
- **GUI menus** (`internal/tui/components/menu.go`) - Application menus
- **Terminal menus** (`internal/proxy/menu/`) - In-game terminal menus
- **No shared components** - Completely separate systems
- **Different input handling** - GUI uses tview events, terminal uses raw input

### Menu Output Routing
```go
// Terminal menu output is injected into the data stream
func (tmm *TerminalMenuManager) SendOutput(text string) {
    // Inject menu output into inbound data stream
    // This makes it appear as if it came from the game server
    tmm.proxy.injectInboundData([]byte(text))
}

// Proxy injects menu data into normal data flow
func (p *Proxy) injectInboundData(data []byte) {
    // Send through normal pipeline - TUI receives via OnData()
    p.pipeline.ProcessInbound(data)
}
```

### Integration with Existing Event System
```go
// Use existing event bus for menu events
event := streaming.Event{
    Type: "TerminalMenuActivated", 
    Data: map[string]interface{}{
        "menu": "TWX_MAIN",
    },
    Source: "TerminalMenuManager",
}
p.eventBus.Publish(event)
```

## Testing Strategy

### Integration Tests
- Menu activation during game session
- Navigation through terminal menu hierarchy
- Script command integration (ADDMENU, etc.)
- Database query menu operations
- Two-stage input collection workflows

### Unit Tests  
- Menu key detection in ProcessOutBound()
- Menu item creation and management
- ANSI output formatting
- Input validation and parsing

## Success Criteria

1. **Menu Activation**: '$' during game session opens TWX_MAIN
2. **Navigation**: Hotkey-based menu navigation works
3. **Script Integration**: All 8 menu script commands functional
4. **Database Queries**: Menu-driven data display works
5. **Visual Compatibility**: ANSI output matches TWX formatting
6. **No GUI Interference**: Terminal menus don't affect GUI menus
7. **Performance**: Menu operations don't impact game latency

This implementation leverages Twist's existing architecture while adding the missing TWX terminal menu functionality as a separate, complementary system.