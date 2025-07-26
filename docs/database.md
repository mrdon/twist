# TWX Database Parsing and Storage System

## Project Overview

This project implements a complete port of the TWX (TradeWars eXtended) proxy's data parsing and storage system from Pascal to Go. The goal is to create a 100% compatible system that can parse TradeWars 2002 game data from telnet streams and store it in a modern database.

## Background

### TradeWars 2002 Context
- **TradeWars 2002** is a classic text-based space trading game played via telnet
- **TWX Proxy** is a popular enhancement tool that sits between the game client and server
- TWX parses game text in real-time to build a comprehensive database of game state
- This database enables features like autopilot, trading analysis, and scripting

### Why Port TWX?
- TWX uses a custom binary file format that's prone to corruption
- The Pascal codebase is 20+ years old and hard to maintain
- Modern Go implementation provides better performance and reliability
- SQLite backend offers ACID transactions and better data integrity
- Planned future enhancement: Port TWX's scripting engine to Go

## Original TWX Architecture

### Data Flow
```
Telnet Stream → ANSI Processing → Line Parsing → State Machine → Database Storage
```

### Key Components (from TWX source)
1. **Process.pas** - Main parsing engine
2. **Database.pas** - File-based storage system  
3. **State Machine** - Tracks what type of game screen is being displayed
4. **Data Structures** - Pascal records matching game data format

### TWX Parsing Approach
- **Line-by-line processing** - Each telnet line processed independently
- **Simple string matching** - Uses `Copy(Line, 1, 10) = 'Beacon  : '` instead of regex
- **State machine** - `FCurrentDisplay` tracks context (sector, port, density scan, etc.)
- **Parameter extraction** - `GetParameter()` function splits space-separated values
- **Character stripping** - Removes parentheses and formatting characters

### TWX Data Structures (Pascal)

#### TSector Record
```pascal
TSector = record
  Warp          : array[1..6] of Word;     // Sector connections
  SPort         : TPort;                   // Port information
  NavHaz        : Byte;                    // Navigation hazard %
  Figs,                                    // Fighter details
  Mines_Armid,
  Mines_Limpet  : TSpaceObject;           // Space objects
  Constellation,
  Beacon        : string[40];             // Sector identifiers
  UpDate        : TDateTime;              // Last update time
  Anomaly       : Boolean;                // Anomaly present
  Density       : LongInt;                // Sector density
  Warps         : Byte;                   // Number of warps (1-6)
  Explored      : TSectorExploredType;    // Exploration level
  Ships,
  Traders,
  Planets,
  Vars          : LongInt;                // Linked list pointers
end;
```

#### TPort Record
```pascal
TPort = record
  Name           : string[40];
  Dead           : Boolean;
  BuildTime,
  ClassIndex     : Byte;                  // Port class (0-9)
  BuyProduct     : array[TProductType] of Boolean;
  ProductPercent : array[TProductType] of Byte;
  ProductAmount  : array[TProductType] of Word;
  UpDate         : TDateTime;
end;
```

#### Supporting Types
```pascal
TFighterType = (ftToll, ftDefensive, ftOffensive, ftNone);
TSectorExploredType = (etNo, etCalc, etDensity, etHolo);
TProductType = (ptFuelOre, ptOrganics, ptEquipment);
```

## Current Implementation Status

### Completed
- ✅ Basic sector parsing (Sector header, Beacon, Ports, Warps)
- ✅ ANSI escape sequence stripping
- ✅ TWX-style string matching approach
- ✅ Simple logging to data.log file
- ✅ Streaming data processing (handles fragmented telnet packets)

### In Progress
- 🟡 SQLite database schema design
- 🟡 Complete TWX data structure port

### TODO List
1. **High Priority**
   - Create SQLite database schema matching TWX TSector structure
   - Implement database interface with SQLite backend
   - Port TWX display state machine (dSector, dPort, dDensity, etc)
   - Implement complete ProcessLine function from TWX
   - Port prompt detection and processing
   - Create Go structs matching TWX records (TSector, TPort, etc)

2. **Medium Priority**
   - Port sector position tracking (spNormal, spPorts, spPlanets, etc)
   - Port port parsing (ProcessPortLine)
   - Port density scanning parsing
   - Port warp line parsing (computer plotted courses)
   - Port ship/trader/planet list parsing
   - Port navigation hazard, mines, fighters parsing
   - Port GetParameter utility to handle edge cases
   - Add database migration system for schema changes
   - Add comprehensive test suite with real TWX data

3. **Low Priority**
   - Port CIM (Computer Information Matrix) parsing
   - Port fighter scan parsing
   - Port QuickStats parsing (player status)

## Technical Architecture

### Current Go Implementation

#### File Structure
```
internal/streaming/
├── pipeline.go          # Data streaming pipeline
├── sector_parser.go     # Main parsing logic (TWX-style)
└── (future)
    ├── database.go      # SQLite interface
    ├── structs.go       # Go versions of TWX records
    └── state_machine.go # Display state tracking
```

#### Key Functions (Current)
```go
// Main parsing entry point
func (sp *SectorParser) ParseText(text string)

// TWX-style line processing  
func (sp *SectorParser) processSectorLine(line string)

// Utility functions matching TWX
func (sp *SectorParser) strToIntSafe(s string) int
func (sp *SectorParser) getParameter(line string, paramNum int) string
func (sp *SectorParser) stripChar(line *string, char rune)
```

### Planned SQLite Schema

#### Design Principles
- **TWX Compatibility** - Field names match TWX exactly (`NavHaz`, `SPort_Name`, etc.)
- **Normalized Structure** - Separate tables for dynamic lists (ships, traders, planets)
- **Performance** - Proper indexing for fast sector/warp lookups
- **Future-Proof** - Easy to extend for scripting engine compatibility

#### Core Tables
```sql
-- Main sectors table (matches TWX TSector)
CREATE TABLE sectors (
    Index INTEGER PRIMARY KEY,              -- TWX field name
    Warp1 INTEGER DEFAULT 0,               -- TWX: Warp[1..6]
    Warp2 INTEGER DEFAULT 0,
    Warp3 INTEGER DEFAULT 0,
    Warp4 INTEGER DEFAULT 0,
    Warp5 INTEGER DEFAULT 0,
    Warp6 INTEGER DEFAULT 0,
    Constellation TEXT DEFAULT '',
    Beacon TEXT DEFAULT '',
    NavHaz INTEGER DEFAULT 0,              -- Exact TWX field name
    Density INTEGER DEFAULT -1,            -- TWX default value
    Anomaly BOOLEAN DEFAULT FALSE,
    Warps INTEGER DEFAULT 0,               -- Warp count
    Explored INTEGER DEFAULT 0,            -- TSectorExploredType
    UpDate DATETIME,                       -- TWX uses 'UpDate'
    
    -- Embedded SPort data (TPort fields)
    SPort_Name TEXT DEFAULT '',
    SPort_Dead BOOLEAN DEFAULT FALSE,
    SPort_ClassIndex INTEGER DEFAULT -1,
    SPort_BuildTime INTEGER DEFAULT 0,
    SPort_UpDate DATETIME,
    
    -- Space objects
    Figs_Quantity INTEGER DEFAULT 0,
    Figs_Owner TEXT DEFAULT '',
    Figs_Type INTEGER DEFAULT 0           -- TFighterType enum
);

-- Dynamic lists (referenced by sector)
CREATE TABLE ships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sector INTEGER,
    name TEXT,
    owner TEXT,
    ship_type TEXT,
    fighters INTEGER DEFAULT 0,
    FOREIGN KEY (sector) REFERENCES sectors(Index)
);

CREATE TABLE traders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sector INTEGER,
    name TEXT,
    ship_type TEXT,
    ship_name TEXT,
    fighters INTEGER DEFAULT 0,
    FOREIGN KEY (sector) REFERENCES sectors(Index)
);

CREATE TABLE planets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sector INTEGER,
    name TEXT,
    FOREIGN KEY (sector) REFERENCES sectors(Index)
);
```

## TradeWars Game Data Format

### Sample Sector Display
```
Sector  : 1 in The Federation (unexplored).
Beacon  : FedSpace, FedLaw Enforced
Ports   : Sol, Class 0 (Special)
Planets : (M) Terra
Warps to Sector(s) :  (2) - (3) - (4) - (5) - (6) - (7)

Command [TL=00:00:00]:[1] (?=Help)? : 
```

### TWX Parsing Patterns
```go
// Sector header
if len(line) >= 10 && line[:10] == "Sector  : " {
    sectorNum := strToIntSafe(getParameter(line, 3))
    // Extract constellation after "in"
}

// Beacon
if len(line) >= 10 && line[:10] == "Beacon  : " {
    beacon := line[10:]
}

// Port
if len(line) >= 10 && line[:10] == "Ports   : " {
    // Parse port name and class
}

// Warps (signals sector completion)
if len(line) >= 20 && line[:20] == "Warps to Sector(s) :" {
    // Strip parentheses, extract warp numbers
    // Save completed sector to database
}
```

### ANSI Processing
TradeWars sends data with ANSI color codes that must be stripped:
```
\x1b[1;32mSector  \x1b[33m: \x1b[36m1 \x1b[0;32min \x1b[1mThe Federation
```

Pattern: `\x1b\[[0-9;]*[mK]` (matches ANSI escape sequences)

## Implementation Guidelines

### Code Style
- **Follow TWX patterns** - Use their proven parsing approach
- **Keep TWX compatibility** - Field names, data types, behaviors must match
- **Simple over clever** - String matching over regex, direct over abstracted
- **Comprehensive logging** - Debug-level logging for all parsing decisions

### Testing Strategy
- **Real game data** - Test with actual TWX database exports
- **Edge cases** - Malformed sectors, network fragmentation, ANSI corruption
- **Performance** - Large databases (20,000+ sectors), high update rates
- **Compatibility** - Ensure data matches TWX output exactly

### Error Handling
- **Graceful degradation** - Continue parsing on malformed lines
- **Data validation** - Check sector numbers, warp destinations
- **Transaction safety** - Use SQLite transactions for atomic updates
- **Logging** - Record all parsing errors for debugging

## Future Phases

### Phase 1: Core Database (Current)
- Complete TWX parsing port
- SQLite storage implementation
- Full test coverage

### Phase 2: Scripting Engine Port
- Port TWX's Pascal scripting language to Go
- Maintain exact API compatibility
- Database interface layer for script access

### Phase 3: Modern Enhancements
- REST API for external tools
- Real-time WebSocket updates
- Advanced analytics and reporting
- Modern client interfaces

## References

### Original TWX Source Files
- `twx-src/Process.pas` - Main parsing engine
- `twx-src/Database.pas` - Storage system
- `twx-src/Core.pas` - Core interfaces and types

### Key TWX Functions to Port
```pascal
procedure ProcessLine(Line: String);           // Main line processor
procedure ProcessSectorLine(Line: String);     // Sector data parsing
procedure ProcessPortLine(Line: String);       // Port data parsing
procedure ProcessPrompt(Line: string);         // Command prompt detection
procedure SectorCompleted;                     // Save sector to database
function LoadSector(I: Integer): TSector;      // Retrieve sector
procedure SaveSector(S: TSector; Index: Integer; ...); // Store sector
```

### Test Data Sources
- `raw_input.log` - Real telnet stream data with ANSI codes
- `twist_debug.log` - Current parsing debug output
- `data.log` - Parsed sector data output

## Development Notes

### Challenges Identified
- **Telnet fragmentation** - Data arrives byte-by-byte, lines can be split
- **ANSI complexity** - Color codes embedded throughout text
- **State management** - Need to track parsing context across packets
- **Data integrity** - Ensure atomic updates and consistency
- **Performance** - Handle high-frequency updates efficiently

### Key Decisions Made
- **SQLite over file format** - Better reliability and performance
- **TWX compatibility first** - Exact field names and behaviors
- **Go over other languages** - Modern, fast, good concurrency
- **String matching over regex** - Simpler, more reliable, matches TWX

### Integration Points
- **Streaming pipeline** - Connects to telnet processor
- **Future scripting engine** - Database interface layer needed
- **External tools** - Consider API access for analysis tools
- **Backup/export** - Easy database portability important