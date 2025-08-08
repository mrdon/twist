# TWX Architecture Analysis

## Overview

TWX (Trade Wars eXtended) is a text-based command system and scripting engine designed for the Trade Wars 2002 game. The architecture is built around a modular design with separate components handling different aspects of the system.

## Core Components

### Script Engine (Script.pas)

The heart of the command system, implementing:

- **TModInterpreter**: Main script manager controlling all loaded scripts
- **TScript**: Individual script execution context with:
  - Compiled bytecode execution
  - Variable management and scoping
  - Trigger system for text-based events
  - Control flow (labels, gotos, gosub/return)

**Key Features:**
- Script compilation from source to bytecode
- Six trigger types: Text, TextLine, TextOut, Delay, Event, TextAuto
- Script lifecycle management (load, execute, terminate)
- Library command system for reusable functionality

### Command System (ScriptCmd.pas)

Implements all available script commands:

- **BuildCommandList()**: Central command registry at line 5292
- Command handlers for:
  - Text processing and manipulation
  - Variable operations and arrays
  - Control flow (if/else, loops)
  - Game interactions (send, triggers)
  - Database operations
  - File I/O and system functions

**Architecture Pattern:**
```pascal
TScriptCmdHandler = function(Script: TObject; Params: array of TCmdParam): TCmdAction;
```

### Command Framework (ScriptRef.pas)

Provides the foundation for the command system:

- **TScriptCmd**: Command definition with parameter validation
- **TScriptRef**: Command and constant registry
- **TCmdParam**: Type-safe parameter handling with automatic conversions
- **TCmdAction**: Command execution results (caNone, caStop, caPause, caAuth)

### Terminal Menu System (Menu.pas)

Text-based terminal interface triggered by '$' character for in-game menu access:

- **TModMenu**: Central controller for terminal menu operations
  - Menu navigation state tracking within terminal session
  - Custom script menu creation via `AddCustomMenu()`
  - Menu lookup by name via `GetMenuByName()`  
  - Script input collection with `BeginScriptInput()`
  - Menu lifecycle management (open/close operations)

- **TTWXMenuItem**: Individual terminal menu items
  - Hotkey-driven navigation (single character selection)
  - Event-driven activation via `TMenuEvent` callbacks
  - Hierarchical parent-child menu relationships
  - Parameter storage for multi-stage input collection
  - Script ownership for dynamic script-generated menus

**Terminal Menu Architecture:**
- '$' character intercept triggers menu activation from game terminal
- Two-stage input handling: initial activation + completion callback pattern
- Built-in navigation options (exit, list, help) configurable per menu
- Script macro support for automated terminal interactions
- Reference-based identification for script integration

**Core Menu Categories:**
- **Main Menu** (`TWX_MAIN`): System functions (connect, burst commands, client management)
- **Script Menu** (`TWX_SCRIPT`): Script control (load, terminate, debug, variable dumps)
- **Data Menu** (`TWX_DATA`): Game database queries (sectors, traders, route plotting, bubbles)
- **Port Menu** (`TWX_PORT`): Port operations (display, listings, class filtering)
- **Setup Menu** (`TWX_SETUP`): System configuration (ports, logging, reconnection)
- **Database Menu** (`TWX_DATABASE`): Database management (create, edit, selection)

**Implementation Details:**
- Menu handlers implemented as `mi*` procedures (e.g., `miBurst`, `miLoad`)
- Two-stage input: initial handler → `OnLineComplete` callback for complex inputs
- Contextual help system integrated per menu item
- Script-generated custom menus via `ADDMENU` command
- Input validation and parameter collection through menu properties

**Terminal Integration Points:**

- **Process.pas** (`TModExtractor:1520`): Menu activation core
  - MenuKey property ('$') intercepts terminal input
  - `ProcessOutBound()` detects menu key and activates `TWX_MAIN` menu
  - Active menu input routing via `TWXMenu.MenuText()` 
  - Escape key handling for script operations

- **Script.pas**: Script-menu interaction
  - Special input menus (`TWX_SCRIPTTEXT`, `TWX_SCRIPTKEY`) for script GetInput commands
  - Menu item lifecycle tracking in `MenuItemList` for cleanup
  - Script termination closes associated menus

- **ScriptCmd.pas**: Script command interface for terminal menus
  - `ADDMENU`: Creates custom menus (parent, name, description, hotkey, reference, prompt, closeMenu)
  - `CLOSEMENU`/`OPENMENU`: Programmatic menu control
  - `GETMENUVALUE`/`SETMENUVALUE`: Menu state manipulation
  - `SETMENUHELP`/`SETMENUOPTIONS`: Menu configuration
  - `SETMENUKEY`: Changes terminal menu activation character

**Terminal Menu Flow:**
1. User types '$' in game terminal → Menu system intercepts and opens `TWX_MAIN`
2. Single-character hotkey navigation → `MenuText()` processes selection
3. Menu item activation → Corresponding `mi*` handler executes functionality  
4. Complex inputs → Two-stage collection via completion callbacks
5. Menu exit → Returns terminal to normal game input mode

## Porting Considerations for Twist

**Display System Requirements:**
- **ANSI Color Support**: TWX uses extensive ANSI color codes (`ANSI_2`, `ANSI_10`, etc.)
  - Menu colors: `MENU_LIGHT` (ANSI_15), `MENU_MID` (ANSI_10), `MENU_DARK` (ANSI_2)
  - Color-coded game data display with contextual meaning
  - Line clearing with `ANSI_CLEARLINE` for prompt management

- **Terminal Control Sequences**:
  - Carriage return (`#13`) + line clear for prompt updates
  - Cursor positioning for menu overlay on game text
  - Dynamic prompt rendering based on menu state

**Input Processing Architecture:**
- **Character-by-character processing** in `MenuKey()` procedure
- **Input validation** and hotkey matching against menu items
- **Two-stage input collection** for complex parameters
- **Line buffering** via `FLine` property for text collection
- **Special key handling**: '?', 'Q', ESC for menu navigation

**State Management:**
- **Menu hierarchy tracking** via parent-child relationships
- **Current menu state** in `FCurrentMenu` with null checks
- **Input script binding** for GetInput command integration
- **Parameter collection** via `AddParam()` for multi-stage inputs
- **Menu cleanup** on script termination

**Critical Implementation Details:**
- **Prompt formatting**: `GetPrompt()` returns `MENU_LIGHT + FPrompt + MENU_MID + '> ' + ANSI_7 + FLine`
- **Option display**: `DumpOptions()` shows hierarchical menu structure with color coding
- **ClearLine property**: Controls whether menu overwrites existing terminal content
- **Menu validation**: Extensive null checking for `CurrentMenu` state
- **Error handling**: Script runtime errors during menu activation

**Integration Points to Replicate:**
- **Process.pas integration**: Menu key detection in outbound data stream
- **Script command interface**: 8 script commands for menu manipulation
- **Database integration**: Menu operations query/display game database
- **Server broadcasting**: All output via `TWXServer.Broadcast()` for multi-client support

**Key Functions to Implement:**
- `ProcessOutBound()` menu key detection
- `MenuText()` input processing engine  
- `DumpOptions()` menu display formatter
- `GetPrompt()` prompt generation
- Script command handlers (`CmdAddMenu`, `CmdOpenMenu`, etc.)

**Modern Go Equivalents:**
- Replace Pascal's manual memory management with Go's garbage collector
- Use Go's string handling instead of Pascal string manipulation
- Implement ANSI terminal package for color/cursor control
- Channel-based input processing instead of procedural callbacks

### Global Coordination (Global.pas)

Central module management:

- Global module instances (TWXInterpreter, TWXMenu, TWXDatabase, etc.)
- Global variable system with persistence
- Timer management for delayed operations
- Module lifecycle coordination

### Data Processing (Process.pas)

Game data extraction and processing:

- **TModExtractor**: Processes inbound/outbound game data
- Text parsing for game state updates
- Sector, port, trader, and ship data extraction
- ANSI code handling and stripping

## Command Execution Flow

1. **Script Loading**: Source compiled to bytecode with parameter references
2. **Execution Loop**: Bytecode interpreter processes commands sequentially
3. **Command Dispatch**: Commands looked up in registry and dispatched to handlers
4. **Parameter Processing**: Type conversion and validation of command parameters
5. **Trigger Processing**: Text events matched against active triggers
6. **State Management**: Variables, labels, and execution context maintained

## Compiled Bytecode Format

Commands stored as:
```
ScriptID:Byte | LineNumber:Word | CmdID:Word | Params | 0:Byte
```

Parameters encoded with type prefixes:
- `PARAM_VAR`: Variable reference with optional indexing
- `PARAM_CONST`: String constant
- `PARAM_SYSCONST`: System constant
- `PARAM_CHAR`: Character code
- `PARAM_PROGVAR`: Program variable

## Trigger System

Six trigger types for event-driven scripting:

- **Text**: Matches anywhere in incoming text
- **TextLine**: Matches complete lines
- **TextOut**: Intercepts outgoing text
- **Delay**: Timer-based triggers
- **Event**: Program event notifications
- **TextAuto**: Automated response triggers with lifecycle

## Key Design Patterns

- **Module Pattern**: Separate concerns into distinct modules
- **Command Pattern**: Uniform interface for all script commands
- **Observer Pattern**: Trigger system for event notifications
- **Interpreter Pattern**: Bytecode execution engine
- **Builder Pattern**: Command and constant registration

## Integration Points

- **Database Layer**: Persistent storage for game data
- **Network Layer**: TCP client/server for game communication
- **UI Layer**: Text-based menus and input handling
- **Logging System**: Debug and audit trail functionality
- **File System**: Script loading and configuration management

This architecture provides a robust foundation for text-based command processing with strong separation of concerns and extensibility through the command registry system.