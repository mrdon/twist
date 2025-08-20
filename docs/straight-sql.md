# Straight SQL Implementation Plan

## Overview

Replace the current three-layer architecture (Database Objects → Intermediate Objects → API Info Objects) with direct SQL updates from parsers using **Squirrel query builder** for **discovered field updates**. Only fields that are actually parsed/discovered during a parsing session are written to the database, preventing overwrites of unparsed data.

## Current Problem

1. Parsers populate intermediate objects (`SectorData`, `PortData`, etc.)
2. Converters transform to database objects (`TSector`, `TPort`, etc.)
3. **Full object saves overwrite all fields, even unparsed/unchanged ones**
4. Complex locking to protect intermediate state
5. Additional conversion step to create API Info objects

## Solution: Discovered Field Tracking

**Goal**: Only update database fields that were actually parsed/discovered in the current parsing session.

**Example**: If parser discovers "credits: 1000" and "turns: 50" from an info display, write only those two fields. Leave all other player stat fields (fighters, shields, etc.) unchanged in the database.

**Note**: This is *discovered field tracking*, not *dirty field tracking*. We don't check if the value changed from what's in the database - we just write what we discovered. This avoids the complexity and performance cost of SELECT-before-UPDATE patterns while solving the main problem of overwriting unparsed data.

## Architecture Trade-offs

### Field Definition Count
Each database field will be defined in **6 places** (vs 4 in current architecture):
1. SQL schema (`schema.go`)
2. Database struct (`structs.go`) 
3. API struct (`api.go`)
4. **[NEW]** Column constant (`const ColPlayerCredits = "credits"`)
5. **[NEW]** Tracker method (`SetCredits(val int)`) 
6. **[NEW]** Database query method (`GetPlayerStatsInfo()`)

### Brittleness Mitigation
- **Comprehensive integration tests** to verify column constants match schema
- **Runtime SQL errors** make mismatches obvious during development
- **Centralized column references** in constants package for easy maintenance
- **Benefits outweigh costs** - eliminates field overwrite problem entirely

## Solution Architecture

### Core Pattern: Squirrel Query Builder + Field Tracking

Use **Squirrel** (lightweight Go SQL builder) with simple field tracking maps to build dynamic UPDATE statements that only affect discovered fields:

```go
import "github.com/Masterminds/squirrel"

// Simple update tracker pattern
type SectorUpdateTracker struct {
    sectorIndex int
    updates     map[string]interface{}
}

func NewSectorUpdate(sectorIndex int) *SectorUpdateTracker {
    return &SectorUpdateTracker{
        sectorIndex: sectorIndex,
        updates:     make(map[string]interface{}),
    }
}

func (s *SectorUpdateTracker) SetConstellation(val string) *SectorUpdateTracker {
    s.updates["constellation"] = val
    return s
}

func (s *SectorUpdateTracker) SetBeacon(val string) *SectorUpdateTracker {
    s.updates["beacon"] = val
    return s
}

func (s *SectorUpdateTracker) Execute(db *sql.DB) error {
    if len(s.updates) == 0 {
        return nil // No updates to perform
    }
    
    psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
    query := psql.Update("sectors").
        SetMap(s.updates).
        Where(squirrel.Eq{"sector_index": s.sectorIndex})
    
    sql, args, err := query.ToSql()
    if err != nil {
        return err
    }
    
    _, err = db.Exec(sql, args...)
    return err
}
```

### Parser Usage Pattern

Parsers use update trackers to collect discovered fields, then execute when parsing completes:

```go
// In parser - session reset on new parsing section
func (p *TWXParser) handleSectorStart(line string) {
    // Reset incomplete trackers from previous session
    if p.sectorFieldTracker != nil && p.sectorFieldTracker.HasUpdates() {
        debug.Log("PARSER: Discarding incomplete sector tracker - new sector detected")
    }
    if p.sectorCollections != nil && p.sectorCollections.HasData() {
        debug.Log("PARSER: Discarding incomplete sector collections - new sector detected")
    }
    
    // Start new discovered field session
    p.sectorFieldTracker = NewSectorTracker(sectorIndex)
    p.sectorCollections = NewSectorCollections(sectorIndex)
    p.sectorFieldTracker.SetConstellation(constellation) // Field discovered - add to update set
}

func (p *TWXParser) handleSectorBeacon(line string) {
    p.sectorFieldTracker.SetBeacon(beacon) // Another field discovered - add to update set
}

func (p *TWXParser) handleSectorShips(line string) {
    // Collections handled separately from fields
    ships := parseShipsFromLine(line)
    for _, ship := range ships {
        p.sectorCollections.AddShip(ship.Name, ship.Owner, ship.ShipType, ship.Fighters)
    }
}

func (p *TWXParser) sectorCompleted() {
    // Execute field updates with only discovered fields
    // Example SQL: UPDATE sectors SET constellation = ?, beacon = ? WHERE sector_index = ?
    // (Notice: warp fields, nav_haz, etc. are NOT updated since they weren't discovered)
    if p.sectorFieldTracker != nil && p.sectorFieldTracker.HasUpdates() {
        err := p.sectorFieldTracker.Execute(p.database.GetDB())
        if err != nil {
            debug.Log("Failed to update sector fields: %v", err)
        }
    }
    
    // Execute collection updates (full replacement)
    if p.sectorCollections != nil && p.sectorCollections.HasData() {
        err := p.sectorCollections.Execute(p.database.GetDB())
        if err != nil {
            debug.Log("Failed to update sector collections: %v", err)
        }
    }
    
    // Read fresh, complete data for API event
    if p.tuiAPI != nil {
        fullSectorInfo, err := p.database.GetSectorInfo(p.currentSectorIndex)
        if err == nil {
            p.tuiAPI.OnCurrentSectorChanged(fullSectorInfo)
        } else {
            debug.Log("Failed to read sector info for API event: %v", err)
        }
    }
    
    // Reset trackers for next session
    p.sectorFieldTracker = nil
    p.sectorCollections = nil
}
```

**Key Benefits**: 
- **Surgical field updates**: Only discovered fields (constellation, beacon) are updated, preserving other data
- **Complete collection replacement**: Ships/traders/planets fully replaced when discovered
- **Session boundary cleanup**: Incomplete trackers are discarded when new parsing sessions start
- **Fresh API data**: Consumers always get complete, current database state

### Dependencies

Add Squirrel: `go get github.com/Masterminds/squirrel`

- **Lightweight**: Single dependency, works with existing `database/sql`
- **Battle-tested**: Widely used in production Go applications  
- **Flexible**: Dynamic SQL generation without reflection overhead

## Phase 1: Player Info Display Parsing (Test Implementation)

### Target: `internal/proxy/streaming/info_parser.go`

The info display parser currently populates `PlayerStats` fields when parsing lines like:
- `Turns left : 150`
- `Credits    : 1000000`
- `Fighters   : 500`

### Implementation Steps

1. **Add Squirrel Dependency**

```bash
go get github.com/Masterminds/squirrel
```

2. **Create Column Constants**

Create `internal/proxy/streaming/columns.go`:

```go
package streaming

// Player stats column constants - single source of truth for column names
const (
    ColPlayerTurns         = "turns"
    ColPlayerCredits       = "credits" 
    ColPlayerFighters      = "fighters"
    ColPlayerShields       = "shields"
    ColPlayerTotalHolds    = "total_holds"
    ColPlayerOreHolds      = "ore_holds"
    ColPlayerOrgHolds      = "org_holds"
    ColPlayerEquHolds      = "equ_holds"
    ColPlayerColHolds      = "col_holds"
    ColPlayerPhotons       = "photons"
    ColPlayerArmids        = "armids"
    ColPlayerLimpets       = "limpets"
    ColPlayerGenTorps      = "gen_torps"
    ColPlayerTwarpType     = "twarp_type"
    ColPlayerCloaks        = "cloaks"
    ColPlayerBeacons       = "beacons"
    ColPlayerAtomics       = "atomics"
    ColPlayerCorbomite     = "corbomite"
    ColPlayerEprobes       = "eprobes"
    ColPlayerMineDisr      = "mine_disr"
    ColPlayerAlignment     = "alignment"
    ColPlayerExperience    = "experience"
    ColPlayerCorp          = "corp"
    ColPlayerShipNumber    = "ship_number"
    ColPlayerPsychicProbe  = "psychic_probe"
    ColPlayerPlanetScanner = "planet_scanner"
    ColPlayerScanType      = "scan_type"
    ColPlayerShipClass     = "ship_class"
    ColPlayerCurrentSector = "current_sector"
    ColPlayerPlayerName    = "player_name"
)

// Sector column constants
const (
    ColSectorConstellation = "constellation"
    ColSectorBeacon        = "beacon"
    ColSectorNavHaz        = "nav_haz"
    ColSectorWarp1         = "warp1"
    ColSectorWarp2         = "warp2"
    ColSectorWarp3         = "warp3"
    ColSectorWarp4         = "warp4"
    ColSectorWarp5         = "warp5"
    ColSectorWarp6         = "warp6"
    ColSectorDensity       = "density"
    ColSectorAnomaly       = "anomaly"
    ColSectorExplored      = "explored"
)
```

3. **Create Simple Update Trackers**

Create `internal/proxy/streaming/update_trackers.go`:

```go
package streaming

import (
    "database/sql"
    "twist/internal/api"
    "github.com/Masterminds/squirrel"
)

type PlayerStatsTracker struct {
    updates map[string]interface{}
}

func NewPlayerStatsTracker() *PlayerStatsTracker {
    return &PlayerStatsTracker{
        updates: make(map[string]interface{}),
    }
}

func (p *PlayerStatsTracker) SetTurns(turns int) *PlayerStatsTracker {
    p.updates[ColPlayerTurns] = turns
    return p
}

func (p *PlayerStatsTracker) SetCredits(credits int) *PlayerStatsTracker {
    p.updates[ColPlayerCredits] = credits
    return p
}

func (p *PlayerStatsTracker) SetFighters(fighters int) *PlayerStatsTracker {
    p.updates[ColPlayerFighters] = fighters
    return p
}

// ... add setters for other commonly parsed fields

func (p *PlayerStatsTracker) Execute(db *sql.DB) error {
    if len(p.updates) == 0 {
        return nil
    }
    
    psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
    
    // Ensure record exists, then update
    _, err := db.Exec("INSERT OR IGNORE INTO player_stats (id) VALUES (1)")
    if err != nil {
        return err
    }
    
    query := psql.Update("player_stats").
        SetMap(p.updates).
        Where(squirrel.Eq{"id": 1})
    
    sql, args, err := query.ToSql()
    if err != nil {
        return err
    }
    
    _, err = db.Exec(sql, args...)
    return err
}

// Don't build partial API objects - read fresh from database instead
func (p *PlayerStatsTracker) HasUpdates() bool {
    return len(p.updates) > 0
}
```

2. **Modify Parser to Use Table Builders**

In `internal/proxy/streaming/info_parser.go`, import and use the table builder:

```go
import (
    "twist/internal/proxy/database/builders"
)

// Add to TWXParser struct
type TWXParser struct {
    // ... existing fields
    playerStatsBuilder *builders.PlayerStatsBuilder // Replaces direct playerStats usage
}

// In info parsing handlers, replace:
// p.playerStats.Credits = credits
// With:
if p.playerStatsBuilder == nil {
    p.playerStatsBuilder = builders.NewPlayerStats()
}
p.playerStatsBuilder.WithCredits(credits)
```

4. **Update Database Query Methods**

Add fresh database read methods to `internal/proxy/database/database.go`:

```go
func (d *SQLiteDatabase) GetPlayerStatsInfo() (api.PlayerStatsInfo, error) {
    info := api.PlayerStatsInfo{}
    
    query := `
        SELECT turns, credits, fighters, shields, total_holds, 
               ore_holds, org_holds, equ_holds, col_holds, photons,
               armids, limpets, gen_torps, twarp_type, cloaks,
               beacons, atomics, corbomite, eprobes, mine_disr,
               alignment, experience, corp, ship_number, 
               psychic_probe, planet_scanner, scan_type,
               ship_class, current_sector, player_name
        FROM player_stats WHERE id = 1`
    
    row := d.db.QueryRow(query)
    err := row.Scan(
        &info.Turns, &info.Credits, &info.Fighters, &info.Shields, &info.TotalHolds,
        &info.OreHolds, &info.OrgHolds, &info.EquHolds, &info.ColHolds, &info.Photons,
        &info.Armids, &info.Limpets, &info.GenTorps, &info.TwarpType, &info.Cloaks,
        &info.Beacons, &info.Atomics, &info.Corbomite, &info.Eprobes, &info.MineDisr,
        &info.Alignment, &info.Experience, &info.Corp, &info.ShipNumber,
        &info.PsychicProbe, &info.PlanetScanner, &info.ScanType,
        &info.ShipClass, &info.CurrentSector, &info.PlayerName,
    )
    
    return info, err
}
```

5. **Update Info Display Completion** 

When info display parsing completes (in `handleCommandPrompt` or similar):

```go
func (p *TWXParser) completeInfoDisplay() {
    if p.playerStatsTracker != nil && p.playerStatsTracker.HasUpdates() {
        // 1. Execute SQL update with ONLY discovered fields
        // Example: UPDATE player_stats SET credits = 1000, turns = 50 WHERE id = 1
        // (fighters, shields, experience, etc. are preserved unchanged)
        err := p.playerStatsTracker.Execute(p.database.GetDB())
        if err != nil {
            debug.Log("Failed to update player stats: %v", err)
            return
        }
        
        // 2. Read complete, fresh data from database for API event
        if p.tuiAPI != nil {
            fullPlayerStats, err := p.database.GetPlayerStatsInfo()  // Fresh DB read
            if err == nil {
                p.tuiAPI.OnPlayerStatsUpdated(fullPlayerStats)  // Complete, accurate data
            }
        }
        
        // Reset for next parsing session
        p.playerStatsTracker = nil
    }
}
```

6. **Add Schema Alignment Tests**

Create `internal/proxy/streaming/schema_alignment_test.go` with comprehensive tests to ensure constants match actual database schema:

```go
package streaming

import (
    "database/sql"
    "fmt"
    "reflect"
    "strings"
    "testing"
    "twist/internal/proxy/database"
    "github.com/stretchr/testify/assert"
)

func TestColumnConstantsMatchSchema(t *testing.T) {
    db := setupTestDB()
    
    // Test all player stats column constants
    playerColumns := []string{
        ColPlayerTurns, ColPlayerCredits, ColPlayerFighters,
        ColPlayerShields, ColPlayerTotalHolds, ColPlayerOreHolds,
        ColPlayerOrgHolds, ColPlayerEquHolds, ColPlayerColHolds,
        ColPlayerPhotons, ColPlayerArmids, ColPlayerLimpets,
        ColPlayerGenTorps, ColPlayerTwarpType, ColPlayerCloaks,
        ColPlayerBeacons, ColPlayerAtomics, ColPlayerCorbomite,
        ColPlayerEprobes, ColPlayerMineDisr, ColPlayerAlignment,
        ColPlayerExperience, ColPlayerCorp, ColPlayerShipNumber,
        ColPlayerPsychicProbe, ColPlayerPlanetScanner, ColPlayerScanType,
        ColPlayerShipClass, ColPlayerCurrentSector, ColPlayerPlayerName,
    }
    
    for _, col := range playerColumns {
        // This will fail if constant doesn't match real column
        _, err := db.Exec(fmt.Sprintf("SELECT %s FROM player_stats LIMIT 1", col))
        assert.NoError(t, err, "Column constant %s doesn't exist in player_stats schema", col)
    }
    
    // Test sector columns
    sectorColumns := []string{
        ColSectorConstellation, ColSectorBeacon, ColSectorNavHaz,
        ColSectorWarp1, ColSectorWarp2, ColSectorWarp3,
        ColSectorWarp4, ColSectorWarp5, ColSectorWarp6,
        ColSectorDensity, ColSectorAnomaly, ColSectorExplored,
    }
    
    for _, col := range sectorColumns {
        _, err := db.Exec(fmt.Sprintf("SELECT %s FROM sectors LIMIT 1", col))
        assert.NoError(t, err, "Column constant %s doesn't exist in sectors schema", col)
    }
}

func TestConstantsMatchStructTags(t *testing.T) {
    // Verify constants match the JSON tags in database structs
    // This catches cases where struct tags are updated but constants aren't
    
    playerStatsType := reflect.TypeOf(database.TPlayerStats{})
    
    // Test a few critical fields
    creditsField, _ := playerStatsType.FieldByName("Credits")
    expectedCreditsCol := strings.Split(creditsField.Tag.Get("json"), ",")[0]
    assert.Equal(t, expectedCreditsCol, ColPlayerCredits, "Credits constant doesn't match struct tag")
    
    turnsField, _ := playerStatsType.FieldByName("Turns")
    expectedTurnsCol := strings.Split(turnsField.Tag.Get("json"), ",")[0]
    assert.Equal(t, expectedTurnsCol, ColPlayerTurns, "Turns constant doesn't match struct tag")
    
    fightersField, _ := playerStatsType.FieldByName("Fighters")
    expectedFightersCol := strings.Split(fightersField.Tag.Get("json"), ",")[0]
    assert.Equal(t, expectedFightersCol, ColPlayerFighters, "Fighters constant doesn't match struct tag")
}

func TestAllConstantsHaveCorrespondingSetters(t *testing.T) {
    // Verify every column constant has a corresponding SetXxx method on tracker
    tracker := NewPlayerStatsTracker()
    trackerType := reflect.TypeOf(tracker)
    
    criticalConstants := map[string]string{
        ColPlayerCredits:  "SetCredits",
        ColPlayerTurns:    "SetTurns", 
        ColPlayerFighters: "SetFighters",
        // Add more as you implement them
    }
    
    for constant, expectedMethod := range criticalConstants {
        _, exists := trackerType.MethodByName(expectedMethod)
        assert.True(t, exists, "Missing setter method %s for constant %s", expectedMethod, constant)
    }
}

func TestSchemaEvolutionDetection(t *testing.T) {
    // This test will fail if someone adds columns to the schema without updating constants
    db := setupTestDB()
    
    // Get actual columns from database
    rows, err := db.Query("PRAGMA table_info(player_stats)")
    assert.NoError(t, err)
    defer rows.Close()
    
    var actualColumns []string
    for rows.Next() {
        var cid int
        var name, dataType string
        var notNull, pk int
        var defaultValue sql.NullString
        
        err = rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
        assert.NoError(t, err)
        
        // Skip metadata columns
        if name != "id" && name != "updated_at" {
            actualColumns = append(actualColumns, name)
        }
    }
    
    // Define expected columns based on our constants
    expectedColumns := []string{
        ColPlayerTurns, ColPlayerCredits, ColPlayerFighters,
        ColPlayerShields, ColPlayerTotalHolds, ColPlayerOreHolds,
        ColPlayerOrgHolds, ColPlayerEquHolds, ColPlayerColHolds,
        ColPlayerPhotons, ColPlayerArmids, ColPlayerLimpets,
        ColPlayerGenTorps, ColPlayerTwarpType, ColPlayerCloaks,
        ColPlayerBeacons, ColPlayerAtomics, ColPlayerCorbomite,
        ColPlayerEprobes, ColPlayerMineDisr, ColPlayerAlignment,
        ColPlayerExperience, ColPlayerCorp, ColPlayerShipNumber,
        ColPlayerPsychicProbe, ColPlayerPlanetScanner, ColPlayerScanType,
        ColPlayerShipClass, ColPlayerCurrentSector, ColPlayerPlayerName,
    }
    
    // Check for missing constants (columns exist but no constant defined)
    for _, actualCol := range actualColumns {
        found := false
        for _, expectedCol := range expectedColumns {
            if actualCol == expectedCol {
                found = true
                break
            }
        }
        assert.True(t, found, "Database column '%s' exists but no constant defined - update column constants!", actualCol)
    }
    
    // Check for extra constants (constants defined but column doesn't exist)
    for _, expectedCol := range expectedColumns {
        found := false
        for _, actualCol := range actualColumns {
            if expectedCol == actualCol {
                found = true
                break
            }
        }
        assert.True(t, found, "Constant '%s' defined but column doesn't exist in database", expectedCol)
    }
}

func TestPlayerStatsTrackerIntegration(t *testing.T) {
    db := setupTestDB()
    tracker := NewPlayerStatsTracker()
    
    // Test discovered field updates
    tracker.SetCredits(1000).SetTurns(50).SetFighters(25)
    err := tracker.Execute(db)
    assert.NoError(t, err)
    
    // Verify only discovered fields were updated
    info, err := database.GetPlayerStatsInfo()
    assert.NoError(t, err)
    assert.Equal(t, 1000, info.Credits)
    assert.Equal(t, 50, info.Turns) 
    assert.Equal(t, 25, info.Fighters)
}

func setupTestDB() *sql.DB {
    // Create in-memory SQLite database with schema
    db, _ := sql.Open("sqlite", ":memory:")
    sqliteDB := database.NewDatabase()
    sqliteDB.CreateDatabase(":memory:")
    return sqliteDB.GetDB()
}
```

7. **Remove Converter Usage**

Remove calls to `PlayerStatsConverter` and direct `SavePlayerStats()` calls in info parsing code paths.

## Phase 2: Sector Display Parsing

### Target: `internal/proxy/streaming/twx_parser.go` sector handlers

Create `internal/proxy/database/builders/sectors.go`:

```go
package builders

type SectorsBuilder struct {
    sectorIndex   int  // WHERE clause value
    discovered    map[string]bool
    constellation *string
    beacon        *string
    navHaz        *int
    warp1         *int
    warp2         *int
    warp3         *int
    warp4         *int
    warp5         *int
    warp6         *int
    density       *int
    anomaly       *bool
    explored      *int
}

func NewSectors(sectorIndex int) *SectorsBuilder {
    return &SectorsBuilder{
        sectorIndex: sectorIndex,
        discovered:  make(map[string]bool),
    }
}

func (b *SectorsBuilder) WithConstellation(constellation string) *SectorsBuilder {
    b.constellation = &constellation
    b.discovered["constellation"] = true
    return b
}

func (b *SectorsBuilder) WithWarps(warps [6]int) *SectorsBuilder {
    b.warp1 = &warps[0]
    b.warp2 = &warps[1]
    b.warp3 = &warps[2]
    b.warp4 = &warps[3]
    b.warp5 = &warps[4]
    b.warp6 = &warps[5]
    b.discovered["warps"] = true
    return b
}

func (b *SectorsBuilder) Execute(db *sql.DB) error {
    // UPDATE sectors SET ... WHERE sector_index = ?
}

func (b *SectorsBuilder) BuildSectorInfo() api.SectorInfo {
    info := api.SectorInfo{Number: b.sectorIndex}
    // Populate from discovered fields
    return info
}
```

Modify sector parsing handlers to use the sectors builder:
- `handleSectorStart()` → `p.sectorsBuilder = builders.NewSectors(sectorNum)`
- `handleSectorWarps()` → `p.sectorsBuilder.WithWarps(warps)`  
- `handleSectorBeacon()` → `p.sectorsBuilder.WithBeacon(beacon)`
- `sectorCompleted()` → `p.sectorsBuilder.Execute()` and fire TUI events

## Phase 3: Port Display Parsing

### Target: Port parsing in sector display

Create `internal/proxy/database/builders/ports.go`:

```go
package builders

type PortsBuilder struct {
    sectorIndex      int  // WHERE clause value
    discovered       map[string]bool
    name             *string
    classIndex       *int
    dead             *bool
    buildTime        *int
    buyFuelOre       *bool
    buyOrganics      *bool
    buyEquipment     *bool
    percentFuelOre   *int
    percentOrganics  *int
    percentEquipment *int
    amountFuelOre    *int
    amountOrganics   *int
    amountEquipment  *int
}

func NewPorts(sectorIndex int) *PortsBuilder {
    return &PortsBuilder{
        sectorIndex: sectorIndex,
        discovered:  make(map[string]bool),
    }
}

func (b *PortsBuilder) Execute(db *sql.DB) error {
    // UPDATE ports SET ... WHERE sector_index = ?
}

func (b *PortsBuilder) BuildPortInfo() api.PortInfo {
    // Build API object from discovered fields
}
```

## Phase 4: Collections (Ships, Traders, Planets)

### Pattern for Collections

Collections require full replacement since we can't do incremental updates. Create separate collection trackers:

**File: `internal/proxy/streaming/collection_trackers.go`**
```go
package streaming

import (
    "database/sql"
    "twist/internal/debug"
)

// SectorCollections manages all collection trackers for a sector
type SectorCollections struct {
    sectorIndex    int
    shipsTracker   *ShipsCollectionTracker
    tradersTracker *TradersCollectionTracker
    planetsTracker *PlanetsCollectionTracker
}

func NewSectorCollections(sectorIndex int) *SectorCollections {
    return &SectorCollections{
        sectorIndex:    sectorIndex,
        shipsTracker:   NewShipsCollectionTracker(sectorIndex),
        tradersTracker: NewTradersCollectionTracker(sectorIndex),
        planetsTracker: NewPlanetsCollectionTracker(sectorIndex),
    }
}

func (sc *SectorCollections) AddShip(name, owner, shipType string, fighters int) {
    sc.shipsTracker.AddShip(name, owner, shipType, fighters)
}

func (sc *SectorCollections) AddTrader(name, shipName, shipType string, fighters int) {
    sc.tradersTracker.AddTrader(name, shipName, shipType, fighters)
}

func (sc *SectorCollections) AddPlanet(name, owner string, fighters int, citadel, stardock bool) {
    sc.planetsTracker.AddPlanet(name, owner, fighters, citadel, stardock)
}

func (sc *SectorCollections) HasData() bool {
    return sc.shipsTracker.HasShips() || 
           sc.tradersTracker.HasTraders() || 
           sc.planetsTracker.HasPlanets()
}

func (sc *SectorCollections) Execute(db *sql.DB) error {
    // Execute all collection updates in sequence
    if sc.shipsTracker.HasShips() {
        if err := sc.shipsTracker.Execute(db); err != nil {
            return err
        }
    }
    
    if sc.tradersTracker.HasTraders() {
        if err := sc.tradersTracker.Execute(db); err != nil {
            return err
        }
    }
    
    if sc.planetsTracker.HasPlanets() {
        if err := sc.planetsTracker.Execute(db); err != nil {
            return err
        }
    }
    
    return nil
}

// Individual collection trackers
type ShipsCollectionTracker struct {
    sectorIndex int
    ships       []ShipData
}

type ShipData struct {
    Name     string
    Owner    string
    ShipType string
    Fighters int
}

func NewShipsCollectionTracker(sectorIndex int) *ShipsCollectionTracker {
    return &ShipsCollectionTracker{
        sectorIndex: sectorIndex,
        ships:       make([]ShipData, 0),
    }
}

func (s *ShipsCollectionTracker) AddShip(name, owner, shipType string, fighters int) {
    s.ships = append(s.ships, ShipData{
        Name:     name,
        Owner:    owner, 
        ShipType: shipType,
        Fighters: fighters,
    })
}

func (s *ShipsCollectionTracker) HasShips() bool {
    return len(s.ships) > 0
}

func (s *ShipsCollectionTracker) Execute(db *sql.DB) error {
    // Atomic replace: DELETE + INSERT in transaction
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    
    // Clear existing ships for this sector
    _, err = tx.Exec("DELETE FROM ships WHERE sector_index = ?", s.sectorIndex)
    if err != nil {
        tx.Rollback()
        return err
    }
    
    // Insert discovered ships
    for _, ship := range s.ships {
        _, err = tx.Exec(`
            INSERT INTO ships (sector_index, name, owner, ship_type, fighters) 
            VALUES (?, ?, ?, ?, ?)`,
            s.sectorIndex, ship.Name, ship.Owner, ship.ShipType, ship.Fighters)
        if err != nil {
            tx.Rollback()
            return err
        }
    }
    
    err = tx.Commit()
    if err != nil {
        return err
    }
    
    debug.Log("COLLECTIONS: Updated %d ships for sector %d", len(s.ships), s.sectorIndex)
    return nil
}
```

Similar patterns for `TradersCollectionTracker` and `PlanetsCollectionTracker`.

## Phase 5: Remove Intermediate Objects

### Files to Remove/Modify

1. **Remove entirely:**
   - `internal/proxy/streaming/data_converters.go`
   - Most of `internal/proxy/database/database.go` bulk save methods
   - Intermediate structs in `twx_parser.go` (`SectorData`, `PortData`, etc.)

2. **Modify:**
   - Remove converter calls in `database_integration.go`
   - Replace `saveSectorToDatabase()` with update builder execution
   - Remove intermediate object field assignments throughout parsers

## Phase 6: Direct Info Object Queries

### Pattern for Reading Data

When TUI/API requests data, query directly into Info objects:

```go
func (d *Database) GetSectorInfo(sectorIndex int) (api.SectorInfo, error) {
    info := api.SectorInfo{Number: sectorIndex}
    
    row := d.db.QueryRow(`
        SELECT constellation, beacon, nav_haz, 
               warp1, warp2, warp3, warp4, warp5, warp6,
               density, anomaly, explored
        FROM sectors WHERE sector_index = ?`, sectorIndex)
    
    var constellation, beacon sql.NullString
    var navHaz, density sql.NullInt64
    var warps [6]sql.NullInt64
    var anomaly sql.NullBool
    var explored sql.NullInt64
    
    err := row.Scan(&constellation, &beacon, &navHaz,
                   &warps[0], &warps[1], &warps[2], &warps[3], &warps[4], &warps[5],
                   &density, &anomaly, &explored)
    
    // Populate info object from nullable database values
    if constellation.Valid {
        info.Constellation = constellation.String
    }
    // ... etc
    
    return info, err
}
```

## Code Impact Analysis

### Files Requiring Major Changes

#### **Parser Files (15+ modification points)**
- `internal/proxy/streaming/info_parser.go` - All `handleInfo*` methods currently populate `p.playerStats`
- `internal/proxy/streaming/sector_parser.go` - Ship/trader/planet parsing populates intermediate arrays
- `internal/proxy/streaming/twx_parser.go` - Parser state includes all intermediate objects
- `internal/proxy/streaming/port_parser.go` - Creates `ProductInfo` and populates `PortData`

#### **Database Integration (8+ converter calls to replace)**
- `internal/proxy/streaming/database_integration.go` - `saveSectorToDatabase()`, `buildSectorInfo()`
- All converter method calls: `ToDatabase()`, `FromDatabase()`, `convertPortData()`
- Bulk save methods: `SaveSector()`, `SavePlayerStats()`, `SaveSectorWithCollections()`

#### **Converter Layer (to be removed entirely)**  
- `internal/proxy/streaming/data_converters.go` - 420+ lines of conversion logic
- `internal/proxy/converter/player.go` - Player stats conversions
- `internal/proxy/converter/sector.go` - Sector conversions  
- `internal/proxy/converter/port.go` - Port conversions

#### **API Event Building (12+ locations)**
- All TUI event firing that builds API objects from intermediate data
- `firePlayerStatsEvent()`, trader events, sector events, port events
- `buildSectorInfo()`, `convertToAPIPortInfo()` functions

#### **Testing (25+ test files to delete or update)**
- Unit tests that create/validate intermediate objects - **DELETE if not worth updating**
- `data_converters_test.go` - **DELETE (testing code we're removing)**
- `trader_event_firing_test.go` - **UPDATE if it tests important behavior, DELETE if just testing intermediate objects**
- Database integration tests that test bulk save patterns - **DELETE (testing deprecated patterns)**
- Keep black-box integration tests in `integration/parsing/` - **UPDATE to validate database state instead of intermediate objects**

### Current Data Flow (to be replaced)

```
Parser Handlers → Intermediate Objects → Converters → Database Bulk Save → API Events
     ↓                    ↓                  ↓             ↓                ↓
handleInfoCredits    p.playerStats    ToDatabase()   SavePlayerStats   converter-built
handleSectorShips    currentShips     ToDatabase()   SaveSector        API objects  
handlePortData       currentSector    convertPort()  SavePort          buildSectorInfo()
```

### New Data Flow (straight-sql pattern)

```
Parser Handlers → Field/Collection Trackers → Direct SQL Updates → Fresh DB Reads → API Events
     ↓                     ↓                        ↓                    ↓              ↓
handleInfoCredits    playerStatsTracker.SetCredits()  UPDATE player_stats  GetPlayerStatsInfo()  complete
handleSectorShips    sectorCollections.AddShip()      DELETE+INSERT ships  GetSectorInfo()      fresh data
handlePortData       portsTracker.SetName()           UPDATE ports         GetPortInfo()        from DB
```

## Implementation Status

✅ **Phase 1 COMPLETE**: Player info parsing (test the pattern) - `info_parser.go`
✅ **Phase 2 COMPLETE**: Basic sector fields (constellation, beacon, navhaz, warps) - `twx_parser.go` 
✅ **Phase 3 COMPLETE**: Port data parsing - `port_parser.go`
✅ **Phase 4 COMPLETE**: Collections (ships, traders, planets) - `sector_parser.go`
🔄 **Phase 5**: Remove converter layer - delete `data_converters.go` and `converter/` package
🔄 **Phase 6**: Update/delete tests:

## Implementation Order
   - **DELETE** unit tests that only validate intermediate object creation/conversion
   - **DELETE** `data_converters_test.go`, bulk save pattern tests, converter tests
   - **UPDATE** integration tests in `integration/parsing/` to validate database state instead of intermediate objects
   - **KEEP** black-box integration tests that validate end-to-end parsing behavior

## Tracker-to-Table Mapping

Each tracker maps to exactly one database table:

✅ **Implemented:**
- `PlayerStatsTracker` → `player_stats` table
- `SectorTracker` → `sectors` table  
- `PortTracker` → `ports` table
- `ShipsCollectionTracker` → `ships` table
- `TradersCollectionTracker` → `traders` table
- `PlanetsCollectionTracker` → `planets` table

🔄 **Future (if needed):**
- `MessageHistoryTracker` → `message_history` table
- `ScriptVarsTracker` → `script_vars` table

## Parser Tracker Usage

Each parser uses the trackers it needs:

✅ **Implemented:**
- **Info Parser**: Uses `PlayerStatsTracker`
- **Sector Parser**: Uses `SectorTracker`, `PortTracker`, `SectorCollections` (ships, traders, planets)
- **TWX Parser**: Orchestrates all trackers through parser state management

🔄 **Future (if needed):**
- **Message Parser**: Uses `MessageHistoryTracker`  
- **Script Variable Parser**: Uses `ScriptVarsTracker`

## Implementation Changes Made

### Phase 1: Player Info Parsing ✅
- **Created** `internal/proxy/streaming/update_trackers.go` with `PlayerStatsTracker`
- **Created** `internal/proxy/streaming/columns.go` with player column constants
- **Added** `GetPlayerStatsInfo()` method to database interface and implementation
- **Modified** `internal/proxy/streaming/info_parser.go` to use PlayerStatsTracker
- **Updated** parser completion to execute tracker and fire fresh API events

### Phase 2: Sector Display Parsing ✅
- **Extended** `update_trackers.go` with `SectorTracker` 
- **Added** sector column constants to `columns.go`
- **Added** `GetSectorInfo()` method for fresh database reads
- **Modified** sector parsing handlers to use SectorTracker instead of intermediate objects
- **Updated** sector completion to execute tracker and fire API events

### Phase 3: Port Display Parsing ✅
- **Extended** `update_trackers.go` with `PortTracker`
- **Added** port column constants to `columns.go` 
- **Added** `GetPortInfo()` method for fresh database reads
- **Modified** port parsing handlers to route through PortTracker
- **Removed** legacy port handling while keeping parsing logic
- **Updated** port completion to execute tracker and fire API events

### Phase 4: Collections (Ships, Traders, Planets) ✅
- **Created** `internal/proxy/streaming/collection_trackers.go`
- **Implemented** `SectorCollections` with atomic replacement pattern
- **Added** individual collection trackers: `ShipsCollectionTracker`, `TradersCollectionTracker`, `PlanetsCollectionTracker`
- **Integrated** collections with sector parsing using `AddShip()`, `AddTrader()`, `AddPlanet()` calls
- **Added** collection execution with transactional DELETE+INSERT pattern

### Architectural Deviations from Original Plan
1. **Trackers vs Builders**: Implemented as "Trackers" instead of "Builders" - same pattern, different naming
2. **Single File Organization**: Combined all trackers in `update_trackers.go` and `collection_trackers.go` instead of separate builder packages
3. **Direct Database Methods**: Added `GetPlayerStatsInfo()`, `GetSectorInfo()`, `GetPortInfo()` methods instead of generic query builders
4. **Legacy Code Handling**: Removed legacy port handling during Phase 3 but preserved parsing logic
5. **Collection Integration**: Collections were fully integrated from the start, not deferred to separate phase

## Key Files Modified

✅ **Created:**
- `internal/proxy/streaming/update_trackers.go` - PlayerStatsTracker, SectorTracker, PortTracker
- `internal/proxy/streaming/collection_trackers.go` - All collection trackers with atomic replacement
- `internal/proxy/streaming/columns.go` - Column constants for all phases

✅ **Modified:**
- `internal/proxy/streaming/twx_parser.go` - Added tracker fields, modified completion handlers
- `internal/proxy/streaming/info_parser.go` - Uses PlayerStatsTracker instead of intermediate objects
- `internal/proxy/streaming/port_parser.go` - Routes through PortTracker, removed legacy saves
- `internal/proxy/database/database.go` - Added GetPlayerStatsInfo(), GetSectorInfo(), GetPortInfo()

🔄 **Remaining:**
- `internal/proxy/streaming/data_converters.go` - Remove converter layer
- Converter calls in database integration - Replace with direct tracker usage
- Test cleanup and updates

## Testing Strategy

**Priority**: Focus on integration tests that validate end-to-end behavior, not implementation details.

### Keep These Tests (High Value)
1. **Black-box integration tests** in `integration/parsing/` that validate:
   - Parser correctly processes game data and stores in database
   - TUI receives correct API events  
   - End-to-end parsing behavior matches TWX exactly
   - Database contains expected data after parsing sessions

### Delete These Tests (Low Value / High Maintenance)
1. **Unit tests** that only validate intermediate object creation
2. **Converter tests** (`data_converters_test.go`) - testing code we're removing
3. **Bulk save pattern tests** - testing deprecated database patterns
4. **Intermediate object validation tests** - internal implementation details

### Update These Tests (Medium Value)
1. **Integration tests that validate database state** - change assertions from intermediate object inspection to database queries
2. **Parser behavior tests** - keep test logic but change validation from object inspection to database state verification

### Test Migration Pattern
```go
// OLD: Testing intermediate objects
assert.Equal(t, 1000, parser.playerStats.Credits)
assert.Equal(t, "Sol", parser.currentSector.Constellation)

// NEW: Testing database state  
playerInfo, _ := database.GetPlayerStatsInfo()
assert.Equal(t, 1000, playerInfo.Credits)
sectorInfo, _ := database.GetSectorInfo(1234)
assert.Equal(t, "Sol", sectorInfo.Constellation)
```

**Goal**: Maintain test coverage of critical parsing behavior while eliminating maintenance burden of testing internal implementation details.

## Implementation Philosophy

### Error Handling: "The Spice Must Flow"
- Parsers continue processing on errors to maintain data flow
- Log errors but don't stop parsing - partial data is better than no data
- Individual tracker failures don't stop other trackers from executing

### Start Simple, Add Complexity When Needed
- No transactions around multiple tracker executions initially
- No batching of database operations initially  
- No complex rollback mechanisms - commit to the new pattern ("YOLO style")
- Add optimizations only when performance issues are observed

### CI Integration
- Schema alignment tests run with `make test` 
- CI automatically catches column constant/schema drift
- No additional tooling needed for maintenance

### Memory & Performance
- Collection trackers are per-sector/per-session - no cross-sector accumulation
- `Execute() → GetInfo()` pattern acceptable for initial implementation
- High-frequency parsing optimization deferred until proven necessary

**Implementation approach**: Prove the pattern with player info parsing, then expand systematically through all parsing domains. Delete unit tests ruthlessly, preserve integration tests, commit fully to the straight-sql approach.