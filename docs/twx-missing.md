# TWX Features Missing from TWIST

This document catalogs features found in TWX (TradeWars eXtended) that are not yet implemented in TWIST, identified through source code analysis of the TWX codebase.

## Overview

TWX represents the mature evolution of TradeWars proxy functionality, developed over many years with extensive automation and analysis capabilities. TWIST currently implements basic proxy functionality but lacks many of the advanced features that made TWX powerful for TradeWars automation.

## Major Missing Feature Categories

### 1. Bot Management System

**TWX Implementation**: Complete bot switching and management system
- `SwitchBot()` functionality for changing active bots
- `GetActiveBotName()`, `GetActiveBotDir()` for bot queries  
- Bot-specific configurations and login scripts
- Per-bot variable storage and script directories
- Auto-start capabilities and bot lifecycle management

**TWIST Status**: ❌ Not implemented
- No bot concept - single script execution model
- No bot switching or management
- No per-bot configurations

**Priority**: High - Core automation feature

### 2. Advanced Variable System

**TWX Implementation**: Comprehensive persistent variable system
- Global variables persistent across script runs (`TGlobalVarItem`)
- Sector-specific variables (`TSectorVar`) with per-sector storage
- Array variables with indexing support
- Variable lifecycle management and persistence

**TWIST Status**: ❌ Partially implemented
- Basic script variables exist but not persistent
- No global variable system
- No sector-specific variables
- No array variable support

**Priority**: High - Essential for complex scripts

### 3. Timer and Event System

**TWX Implementation**: Built-in timer management (`TTimerItem`)
- Named timers with configurable intervals
- Timer lifecycle management (create, delete, pause, resume)
- Event-driven script execution via timers
- Performance counter integration

**TWIST Status**: ❌ Not implemented
- No built-in timer system
- Scripts must implement their own timing
- No event-driven execution model

**Priority**: Medium - Important for automation

### 4. Observer Pattern Architecture

**TWX Implementation**: Full Observer pattern implementation
- `IObserver`/`ISubject` interfaces for clean event communication
- `TNotificationType` enumeration for different event types
- Event bubbling system (`TModBubble`) for UI updates
- Clean separation between modules via events

**TWIST Status**: ❌ Not implemented
- Direct method calls between modules
- No event system for module communication
- Tight coupling between components

**Priority**: High - Architectural foundation (covered in API separation plan)

### 5. Export/Import System

**TWX Implementation**: Complete database sharing system
- Binary TWX file format with structured headers (`TExportHeader`, `TExportSector`)
- CRC32 checksum verification for data integrity
- Version management for forward/backward compatibility
- Sector data export with timestamps and update tracking

**TWIST Status**: ❌ Not implemented
- No database export/import functionality
- No data sharing capabilities
- No backup/restore system

**Priority**: Medium - Useful for data sharing

### 6. Advanced Database Features

**TWX Implementation**: Sophisticated data management
- Per-sector custom variables with linked list storage
- Ship and trader tracking with detailed records
- Port analysis with economic data and timestamps
- Fighter and mine management with ownership tracking
- Historical data with update timestamps

**TWIST Status**: ❌ Partially implemented
- Basic sector/port parsing exists
- No per-sector variables
- No ship/trader historical tracking
- Limited economic analysis

**Priority**: Medium - Enhanced game analysis

### 7. Advanced Game State Tracking

**TWX Implementation**: Comprehensive state management
- Detailed ship equipment tracking (holds, weapons, equipment)
- Player statistics (experience, alignment, corporation)
- Economic analysis and credit tracking
- Combat system integration with fighter types
- Equipment categories: photons, armids, limpets, genesis torps, cloaks, beacons, etc.

**TWIST Status**: ❌ Basic implementation
- Basic player stats tracked
- No detailed equipment tracking
- No economic analysis tools
- No combat system integration

**Priority**: Low - Enhanced features

### 8. Authentication and Security

**TWX Implementation**: Authentication system
- `Auth.pas` module with authentication handling
- Observer pattern for authentication events (`ntAuthenticationDone`, `ntAuthenticationFailed`)
- Built-in security and login management

**TWIST Status**: ❌ Not implemented
- No authentication system
- Direct connection model only

**Priority**: Low - Security enhancement

### 9. Menu and GUI Integration

**TWX Implementation**: Sophisticated menu system
- `Menu.pas` with comprehensive menu management
- Script menu integration
- GUI module (`GUI.pas`) for interface management
- Form-based UI with multiple specialized forms

**TWIST Status**: ❌ Basic TUI implementation
- Simple terminal-based interface
- Basic menu system in TUI
- No GUI integration planned (by design)

**Priority**: N/A - Different UI approach

### 10. Process and Data Extraction

**TWX Implementation**: Advanced game data processing
- `Process.pas` with sophisticated parsing (`TModExtractor`)
- Sector position tracking (`TSectorPosition`)
- Display state management (`TDisplay`)
- Fighter scan processing (`TFigScanType`)
- Real-time game state extraction and analysis

**TWIST Status**: ❌ Basic implementation
- Basic ANSI parsing and sector detection
- No advanced display state tracking
- Limited game state extraction

**Priority**: Medium - Enhanced game integration

## Implementation Priority

### Phase 1 (High Priority - Core Functionality)
1. **Observer Pattern Architecture** - Foundation for clean module separation
2. **Bot Management System** - Core automation capability
3. **Advanced Variable System** - Essential for complex scripts

### Phase 2 (Medium Priority - Enhanced Features)  
1. **Timer and Event System** - Automation enhancement
2. **Export/Import System** - Data sharing and backup
3. **Advanced Database Features** - Enhanced analysis
4. **Process and Data Extraction** - Better game integration

### Phase 3 (Low Priority - Polish Features)
1. **Advanced Game State Tracking** - Detailed analysis tools
2. **Authentication System** - Security enhancement

## API Design Implications

The missing features have been accounted for in the API design with "Future TWX-inspired functionality" sections, ensuring the architecture can support these features when implemented:

- **25+ additional ProxyAPI methods** for advanced functionality
- **20+ additional TuiAPI methods** for comprehensive event handling  
- **10+ new data structures** supporting advanced capabilities

## Notes on TWX Source Analysis

The TWX source code analysis was performed on the Delphi/Pascal codebase located at `/home/mrdon/dev/twxp/Source/TWX27/`. Key architectural insights were derived from:

- `Core.pas` - Module architecture and interfaces
- `Global.pas` - Global variable and timer management
- `Script.pas` - Script interpretation and trigger system
- `Database.pas` - Data structures and storage
- `Process.pas` - Game data extraction and processing
- `Observer.pas` - Observer pattern implementation
- `TWXExport.pas` - Export/import system

The analysis focused on identifying architectural patterns and feature gaps rather than attempting to replicate TWX's exact implementation approach.