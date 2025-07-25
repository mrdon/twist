# Trade Wars 2002 Client - Product Requirements Document

## Overview

A modern client for the classic Trade Wars 2002 game that provides a text-based user interface (TUI) with enhanced features like chat extraction, sector mapping, and scripting capabilities. The client connects to telnet-based Trade Wars servers and provides both display and data extraction functionality.

## Architecture

The application consists of two main components within a single executable:

1. **Telnet Proxy System** - Handles server communication and data processing
2. **TUI Interface** - Provides the user interface and interaction

### Component Separation
- Code will be organized in separate packages to allow future splitting into separate executables
- Direct method calls between proxy and TUI (no network communication initially)

## MVP Requirements

### Telnet Proxy System
- **Server Connection**: Connect to remote telnet servers (default: `telnet://crunchers-twgs.com:2002`)
- **Character Translation**: Real-time translation from DOS Code Page 437 to UTF-8 for proper ANSI display
- **Data Flow**: Pass-through proxy that forwards raw text to TUI after translation
- **Basic Parsing**: Parse chat messages from game text (extract but still send raw text to TUI)

### TUI Interface
- **Three-Panel Layout**: 
  - **Left panel (25% width)**: Player information and ship stats
    - Trader Info: Sector, Turns, Experience, Alignment, Credits (with colored progress bars)
    - Holds: Cargo capacity breakdown (Total, Fuel Ore, Organics, Equipment, Colonists, Empty)
    - Quick Query: Text input field for quick commands
    - Stats: Session tracking (profit, etc.)
  - **Middle panel (50% width)**: Main terminal output from proxy connection
    - Game text with proper ANSI coloring (green, blue, yellow text preservation)
    - Command prompt at bottom with optional timer display
    - Scrollable terminal history
  - **Right panel (25% width)**: Sector information and mapping
    - Sector map visualization showing numbered sectors and connections
    - Current sector details (Port info, Density, NavHaz status)
    - Notepad area for player notes
- **Bottom Chat Panel**: Full-width panel for extracted chat messages
  - Communication channels (Federation comm-link, Subspace radio, etc.)
  - Chat history with timestamps
- **Top Menu**: Connect/disconnect options with custom server address input
- **Input Handling**: Immediate keystroke forwarding to proxy (terminal-like behavior)
- **Connection States**: Display "Not Connected" message when disconnected
- **Error Handling**: Show connection errors in middle panel

### Technical Requirements
- **Language**: Go
- **UI Framework**: bubbletea, bubbles, and lipgloss
- **Layout**: 25% / 50% / 25% panel distribution
- **Performance**: Real-time character translation for responsiveness

## Future Enhancements

### Advanced Proxy Features
- **Multi-Client Support**: Allow multiple TUI clients to connect to same game session
- **Telnet Service**: Expose telnet service on configurable port for external clients
- **State Management**: Store extracted game data (sectors, ports, ship status)
- **Advanced Parsing**: Extract detailed game information (current sector, port details, player stats)

### Enhanced TUI Features
- **Interactive Panels**: 
  - Left panel: Clickable quick actions, real-time stat updates, interactive progress bars
  - Right panel: Interactive sector map with click-to-navigate, sector details on hover
- **Advanced Chat System**: 
  - Multi-channel support with tabs
  - Chat filtering and search
  - Player highlighting and ignore lists
  - Chat logging to files
- **Script Support**: Menu option for running game scripts with progress indicators
- **Configuration**: Settings for appearance, server presets, keybindings, panel layouts
- **Multiple Views**: Different layout modes optimized for trading, combat, exploration
- **Enhanced Notepad**: Rich text notes with sector linking and search functionality

### Data Extraction & Analysis
- **Game State Tracking**: Monitor player position, credits, cargo, etc.
- **Market Analysis**: Track port prices and trade opportunities
- **Sector Mapping**: Build and display universe map from exploration
- **Player Tracking**: Monitor other players' activities and locations
- **Trade Route Optimization**: Suggest profitable trade routes

### Scripting & Automation
- **Script Engine**: Execute automated trading and exploration scripts
- **Event Triggers**: React to specific game events or conditions
- **Macro Recording**: Record and replay common action sequences
- **API Integration**: Allow external tools to query game state

## Success Criteria

### MVP Success
- Successfully connect to Trade Wars servers
- Properly display ANSI graphics and text
- Functional three-panel layout with basic interaction
- Stable connection management with error handling
- Clean separation of proxy and TUI code for future extensibility

### Long-term Success
- Feature parity with classic Trade Wars clients
- Enhanced user experience through modern UI paradigms
- Extensible architecture supporting third-party integrations
- Active community adoption and contribution