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

### Menu System (Menu.pas)

Text-based user interface implementation:

- **TModMenu**: Menu controller with navigation state
- **TTWXMenuItem**: Individual menu items with hotkey support
- Hierarchical menu structure
- Input handling and validation
- Script integration for custom menus

**Menu Categories:**
- Main system functions (connect, scripts, data)
- Script management (load, stop, debug)
- Database operations (create, edit, query)
- Game utilities (sector display, plotting)

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