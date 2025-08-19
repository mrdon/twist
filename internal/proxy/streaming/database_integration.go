package streaming

import (
	"database/sql"
	"fmt"
	"twist/internal/api"
	"twist/internal/debug"
)

// database_integration.go - Clean database integration points
// This file focuses solely on connecting the parser to the database layer
//
// PREFERRED SAVE PATTERN: Use specific save functions instead of bulk saves
// Player Stats:
//   - savePlayerCurrentSector() - updates current sector only
//   - savePlayerCredits() - updates credits only  
//   - savePlayerTurns() - updates turns only
//   - saveAllPlayerStats() - full save (for initial setup only)
//
// Sector Data:
//   - saveSectorBasicInfo() - saves constellation, beacon, navhaz (preserves warps)
//   - saveProbeWarp(from, to) - saves individual probe warps
//   - saveSectorPort() - saves port data
//   - saveSectorVisited(sectorIndex) - marks sector as actually visited (EtHolo)
//   - saveSectorProbeData(sectorIndex) - marks sector as having probe data (EtCalc)
//
// Benefits: Prevents data overwrites, better performance, atomic operations

// DEPRECATED: saveSectorToDatabase is deprecated. Use specific save functions instead:
// - saveSectorBasicInfo() - saves constellation, beacon, navhaz (preserves warps)
// - saveProbeWarp() - saves individual probe warps  
// - saveSectorPort() - saves port data
// This prevents accidental data overwrites and provides better performance.
func (p *TWXParser) saveSectorToDatabase() error {
	
	if p.currentSectorIndex <= 0 {
		return nil
	}
	
	// Debug: Check if parser's database instance is valid
	if p.database == nil {
		return fmt.Errorf("parser database instance is nil")
	}
	
	// Debug: Test parser's database connection
	if db := p.database.GetDB(); db != nil {
		var tableCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sectors'").Scan(&tableCount); err != nil {
			return fmt.Errorf("parser's database connection is broken: %w", err)
		}
		if tableCount == 0 {
			return fmt.Errorf("parser's database has no sectors table")
		}
	} else {
		return fmt.Errorf("parser's database GetDB() returned nil")
	}
	
	// Debug: Check what warps we have before conversion
	
	// Build complete sector data from current parsing context
	sectorData := SectorData{
		Index:         p.currentSectorIndex,
		Constellation: p.currentSector.Constellation,
		Beacon:        p.currentSector.Beacon,
		NavHaz:        p.currentSector.NavHaz, // Include parsed NavHaz
		Warps:         p.currentSectorWarps, // Use parsed warps
		Port:          p.currentSector.Port,  // Include port data with build time
		Ships:         append([]ShipInfo{}, p.currentShips...),
		Traders:       append([]TraderInfo{}, p.currentTraders...),
		Planets:       append([]PlanetInfo{}, p.currentPlanets...),
		Mines:         append([]MineInfo{}, p.currentMines...),
		Products:      append([]ProductInfo{}, p.currentProducts...),
		Explored:      true,
	}
	
	
	// Convert to database format using converter
	converter := NewSectorConverter()
	dbSector := converter.ToDatabase(sectorData)
	
	// Convert collections separately for Pascal-compliant SaveSector signature
	dbShips := converter.convertShips(sectorData.Ships)
	dbTraders := converter.convertTraders(sectorData.Traders)
	dbPlanets := converter.convertPlanets(sectorData.Planets)
	
	// Save sector to database using Pascal-compliant signature
	// This mirrors Pascal TWX: SaveSector(FCurrentSector, FCurrentSectorIndex, FShipList, FTraderList, FPlanetList)
	if err := p.database.SaveSectorWithCollections(dbSector, p.currentSectorIndex, dbShips, dbTraders, dbPlanets); err != nil {
		return err
	}
	
	// Save port data separately using common function (Phase 2: ports are in separate table)
	if p.currentSector.Port.Name != "" || p.currentSector.Port.ClassIndex > 0 {
		// Convert port data to database format
		dbPort := converter.convertPortData(p.currentSector.Port)
		if err := p.ensureSectorExistsAndSavePort(dbPort, p.currentSectorIndex); err != nil {
			return fmt.Errorf("failed to save port data: %w", err)
		}
	} else {
		// No port detected in this sector visit - clear any existing port data
		// This ensures database is updated when visiting sectors that no longer have ports
		if err := p.clearPortData(p.currentSectorIndex); err != nil {
			return fmt.Errorf("failed to clear port data: %w", err)
		}
	}
	
	
	// Update current sector in player stats (like TWX Database.pas)
	// NOTE: We don't save player stats here because it would overwrite QuickStats data with zeros
	// Current sector is updated when QuickStats is parsed
	p.playerStats.CurrentSector = p.currentSectorIndex
	
	// Notify TUI API if available
	if p.tuiAPI != nil {
		sectorInfo := p.buildSectorInfo(sectorData)
		debug.Log("DATABASE_INTEGRATION: Firing OnCurrentSectorChanged for sector %d [SOURCE: saveSectorToDatabase]", p.currentSectorIndex)
		p.tuiAPI.OnCurrentSectorChanged(sectorInfo)
	}
	
	return nil
}

// savePlayerCurrentSector updates only the current sector in player stats
func (p *TWXParser) savePlayerCurrentSector() error {
	db := p.database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	
	_, err := db.Exec("UPDATE player_stats SET current_sector = ? WHERE id = 1", p.playerStats.CurrentSector)
	return err
}

// savePlayerCredits updates only the credits in player stats
func (p *TWXParser) savePlayerCredits() error {
	db := p.database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	
	_, err := db.Exec("UPDATE player_stats SET credits = ? WHERE id = 1", p.playerStats.Credits)
	return err
}

// savePlayerTurns updates only the turns in player stats
func (p *TWXParser) savePlayerTurns() error {
	db := p.database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	
	_, err := db.Exec("UPDATE player_stats SET turns = ? WHERE id = 1", p.playerStats.Turns)
	return err
}

// saveAllPlayerStats saves complete player stats (for initial setup or complete refresh)
func (p *TWXParser) saveAllPlayerStats() error {
	converter := NewPlayerStatsConverter()
	dbStats := converter.ToDatabase(p.playerStats)
	return p.database.SavePlayerStats(dbStats)
}

// saveSectorBasicInfo saves sector constellation, beacon, and navhaz (but not warps)
func (p *TWXParser) saveSectorBasicInfo() error {
	if p.currentSectorIndex <= 0 {
		return nil
	}
	
	db := p.database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	
	// Check if sector exists to decide between INSERT and UPDATE
	var exists int
	err := db.QueryRow("SELECT COUNT(*) FROM sectors WHERE sector_index = ?", p.currentSectorIndex).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if sector exists: %w", err)
	}
	
	if exists == 0 {
		// INSERT new sector with basic info only (no warps, no exploration status change)
		_, err = db.Exec(`
			INSERT INTO sectors (sector_index, constellation, beacon, nav_haz)
			VALUES (?, ?, ?, ?)
		`, p.currentSectorIndex, p.currentSector.Constellation, p.currentSector.Beacon, p.currentSector.NavHaz)
	} else {
		// UPDATE only the basic info fields, preserving warps and exploration status
		_, err = db.Exec(`
			UPDATE sectors 
			SET constellation = ?, beacon = ?, nav_haz = ?
			WHERE sector_index = ?
		`, p.currentSector.Constellation, p.currentSector.Beacon, p.currentSector.NavHaz, p.currentSectorIndex)
	}
	
	if err != nil {
		return err
	}
	
	// Fire sector update event only if data actually changed after successful save
	if p.tuiAPI != nil {
		// Check if sector data actually changed by comparing with database
		if p.hasSectorDataChanged(p.currentSectorIndex) {
			// Build sector info from current parser data
			sectorData := p.buildSectorData()
			sectorInfo := p.buildSectorInfo(sectorData)
			debug.Log("DATABASE: Firing OnSectorUpdated for basic info sector %d", p.currentSectorIndex)
			p.tuiAPI.OnSectorUpdated(sectorInfo)
		} else {
			debug.Log("DATABASE: Skipping OnSectorUpdated for sector %d - no data changes detected", p.currentSectorIndex)
		}
	}
	
	return nil
}

// saveSectorWarps saves the parsed warps from currentSectorWarps to the database
func (p *TWXParser) saveSectorWarps() error {
	if p.currentSectorIndex <= 0 {
		return nil
	}
	
	// Don't overwrite probe warps - those are handled by saveProbeWarp
	if p.probeMode || p.probeDiscoveredSectors[p.currentSectorIndex] {
		return nil
	}
	
	// Never save all zeros - that means no warps were parsed
	hasNonZeroWarp := false
	for _, warp := range p.currentSectorWarps {
		if warp > 0 {
			hasNonZeroWarp = true
			break
		}
	}
	if !hasNonZeroWarp {
		debug.Log("WARP: Skipping save of all-zero warps for sector %d", p.currentSectorIndex)
		return nil
	}
	
	db := p.database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	
	// Check if sector exists first
	var exists int
	err := db.QueryRow("SELECT COUNT(*) FROM sectors WHERE sector_index = ?", p.currentSectorIndex).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if sector exists: %w", err)
	}
	
	if exists == 0 {
		// Insert new sector with warps only (minimal data)
		_, err = db.Exec(`
			INSERT INTO sectors (sector_index, warp1, warp2, warp3, warp4, warp5, warp6)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, p.currentSectorIndex, p.currentSectorWarps[0], p.currentSectorWarps[1], p.currentSectorWarps[2], 
		   p.currentSectorWarps[3], p.currentSectorWarps[4], p.currentSectorWarps[5])
	} else {
		// Update warp data for existing sector
		_, err = db.Exec(`
			UPDATE sectors 
			SET warp1 = ?, warp2 = ?, warp3 = ?, warp4 = ?, warp5 = ?, warp6 = ?
			WHERE sector_index = ?
		`, p.currentSectorWarps[0], p.currentSectorWarps[1], p.currentSectorWarps[2], 
		   p.currentSectorWarps[3], p.currentSectorWarps[4], p.currentSectorWarps[5], 
		   p.currentSectorIndex)
	}
	
	if err != nil {
		return fmt.Errorf("failed to save sector warps: %w", err)
	}
	
	debug.Log("WARP: Saved warps for sector %d: %v", p.currentSectorIndex, p.currentSectorWarps)
	return nil
}

// saveProbeWarp saves a single probe warp from one sector to another
func (p *TWXParser) saveProbeWarp(fromSector, toSector int) error {
	debug.Log("saveProbeWarp: adding warp %d -> %d", fromSector, toSector)
	
	db := p.database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	
	// First, get existing warps to find an empty slot
	var warp1, warp2, warp3, warp4, warp5, warp6 int
	row := db.QueryRow("SELECT warp1, warp2, warp3, warp4, warp5, warp6 FROM sectors WHERE sector_index = ?", fromSector)
	err := row.Scan(&warp1, &warp2, &warp3, &warp4, &warp5, &warp6)
	
	if err == sql.ErrNoRows {
		// Sector doesn't exist, create it with the warp and mark as probe data (EtCalc = 1)
		_, err = db.Exec(`
			INSERT INTO sectors (sector_index, warp1, explored) VALUES (?, ?, 1)
		`, fromSector, toSector)
		debug.Log("saveProbeWarp: created new sector %d with warp to %d", fromSector, toSector)
		return err
	} else if err != nil {
		return fmt.Errorf("failed to query existing warps for sector %d: %w", fromSector, err)
	}
	
	// Check if warp already exists
	warps := []int{warp1, warp2, warp3, warp4, warp5, warp6}
	for _, warp := range warps {
		if warp == toSector {
			debug.Log("saveProbeWarp: warp %d -> %d already exists", fromSector, toSector)
			return nil
		}
	}
	
	// Find first empty slot (0 value) and update it
	for i, warp := range warps {
		if warp == 0 {
			warpCol := fmt.Sprintf("warp%d", i+1)
			_, err = db.Exec(fmt.Sprintf("UPDATE sectors SET %s = ? WHERE sector_index = ?", warpCol), toSector, fromSector)
			debug.Log("saveProbeWarp: added warp to %s in sector %d", warpCol, fromSector)
			return err
		}
	}
	
	debug.Log("saveProbeWarp: no empty warp slots in sector %d", fromSector)
	return nil // All warp slots full
}

// saveSectorPort saves port data for the current sector (only when we actually have port data)
func (p *TWXParser) saveSectorPort() error {
	if p.currentSectorIndex <= 0 {
		return nil
	}
	
	// Only save if we actually have port data to save
	if p.currentSector.Port.Name != "" || p.currentSector.Port.ClassIndex > 0 {
		converter := NewSectorConverter()
		dbPort := converter.convertPortData(p.currentSector.Port)
		err := p.ensureSectorExistsAndSavePort(dbPort, p.currentSectorIndex)
		if err != nil {
			return err
		}
		
		// Fire port update event after successful save
		if p.tuiAPI != nil {
			portInfo := converter.convertToAPIPortInfo(p.currentSector.Port, p.currentSectorIndex)
			debug.Log("DATABASE: Firing OnPortUpdated for sector %d: %s", p.currentSectorIndex, p.currentSector.Port.Name)
			p.tuiAPI.OnPortUpdated(portInfo)
		}
		
		return nil
	}
	
	// Do nothing if no port data - we don't know if sector has a port or not
	return nil
}

// clearSectorPort explicitly clears port data when we know the sector has no port
func (p *TWXParser) clearSectorPort() error {
	if p.currentSectorIndex <= 0 {
		return nil
	}
	
	return p.clearPortData(p.currentSectorIndex)
}

// saveSectorVisited marks a sector as actually visited by the player (EtHolo)
func (p *TWXParser) saveSectorVisited(sectorIndex int) {
	if sectorIndex <= 0 {
		return
	}
	
	db := p.database.GetDB()
	if db == nil {
		panic(fmt.Sprintf("Database connection is nil when saving sector %d as visited", sectorIndex))
	}
	
	// Use INSERT OR IGNORE to create sector if it doesn't exist, then UPDATE to mark as EtHolo (3)
	if _, err := db.Exec("INSERT OR IGNORE INTO sectors (sector_index) VALUES (?)", sectorIndex); err != nil {
		panic(fmt.Sprintf("Failed to insert sector %d: %v", sectorIndex, err))
	}
	
	if _, err := db.Exec("UPDATE sectors SET explored = 3 WHERE sector_index = ?", sectorIndex); err != nil {
		panic(fmt.Sprintf("Failed to mark sector %d as visited: %v", sectorIndex, err))
	}
}

// saveSectorProbeData marks a sector as having probe/calculated data (EtCalc)
func (p *TWXParser) saveSectorProbeData(sectorIndex int) {
	if sectorIndex <= 0 {
		return
	}
	
	db := p.database.GetDB()
	if db == nil {
		panic(fmt.Sprintf("Database connection is nil when saving sector %d probe data", sectorIndex))
	}
	
	// Only update exploration status if sector doesn't exist or has lower exploration status
	// Don't downgrade EtHolo (3) or EtDensity (2) to EtCalc (1)
	if _, err := db.Exec(`
		INSERT OR IGNORE INTO sectors (sector_index, explored) VALUES (?, 1);
		UPDATE sectors SET explored = 1 WHERE sector_index = ? AND explored < 1
	`, sectorIndex, sectorIndex); err != nil {
		panic(fmt.Sprintf("Failed to save probe data for sector %d: %v", sectorIndex, err))
	}
	
	// Fire sector update event only if data actually changed after successful probe data save
	if p.tuiAPI != nil {
		// Check if sector data actually changed by comparing with database
		if p.hasSectorDataChanged(sectorIndex) {
			// For probe data, we need to get the sector info from database since we don't have current parser data
			// We'll create a minimal SectorInfo with what we know
			sectorInfo := api.SectorInfo{
				Number:  sectorIndex,
				Visited: false, // This is probe data, not actually visited
			}
			debug.Log("DATABASE: Firing OnSectorUpdated for probe data sector %d", sectorIndex)
			p.tuiAPI.OnSectorUpdated(sectorInfo)
		} else {
			debug.Log("DATABASE: Skipping OnSectorUpdated for probe data sector %d - no data changes detected", sectorIndex)
		}
	}
}

// buildSectorInfo converts SectorData to api.SectorInfo for TUI API
func (p *TWXParser) buildSectorInfo(sectorData SectorData) api.SectorInfo {
	// Extract non-zero warps from the array
	var warps []int
	for _, warp := range sectorData.Warps {
		if warp > 0 {
			warps = append(warps, warp)
		}
	}
	
	sectorInfo := api.SectorInfo{
		Number:        sectorData.Index,
		NavHaz:        sectorData.NavHaz,
		HasTraders:    len(sectorData.Traders),
		Constellation: sectorData.Constellation,
		Beacon:        sectorData.Beacon,
		Warps:         warps,
		HasPort:       false, // Default to false
	}
	
	// Phase 2: Set HasPort flag by checking if port exists in ports table
	if p.database != nil {
		if portData, err := p.database.LoadPort(sectorData.Index); err == nil && portData.ClassIndex > 0 {
			sectorInfo.HasPort = true
		}
	}
	
	return sectorInfo
}

// hasSectorDataChanged checks if the sector data in memory differs from what's stored in the database
func (p *TWXParser) hasSectorDataChanged(sectorIndex int) bool {
	if p.database == nil {
		return true // Assume changed if no database connection
	}
	
	db := p.database.GetDB()
	if db == nil {
		return true // Assume changed if no database connection
	}
	
	// Load current database values for comparison
	var dbConstellation, dbBeacon sql.NullString
	var dbNavHaz sql.NullInt64
	var dbWarp1, dbWarp2, dbWarp3, dbWarp4, dbWarp5, dbWarp6 sql.NullInt64
	var dbExplored sql.NullInt64
	
	row := db.QueryRow(`
		SELECT constellation, beacon, nav_haz, warp1, warp2, warp3, warp4, warp5, warp6, explored
		FROM sectors WHERE sector_index = ?
	`, sectorIndex)
	
	err := row.Scan(&dbConstellation, &dbBeacon, &dbNavHaz, &dbWarp1, &dbWarp2, &dbWarp3, &dbWarp4, &dbWarp5, &dbWarp6, &dbExplored)
	if err == sql.ErrNoRows {
		return true // New sector, definitely changed
	} else if err != nil {
		return true // Error reading, assume changed
	}
	
	// Compare constellation
	currentConstellation := p.currentSector.Constellation
	if (dbConstellation.Valid && dbConstellation.String != currentConstellation) || 
	   (!dbConstellation.Valid && currentConstellation != "") {
		return true
	}
	
	// Compare beacon
	currentBeacon := p.currentSector.Beacon
	if (dbBeacon.Valid && dbBeacon.String != currentBeacon) || 
	   (!dbBeacon.Valid && currentBeacon != "") {
		return true
	}
	
	// Compare nav haz
	currentNavHaz := int64(p.currentSector.NavHaz)
	if (dbNavHaz.Valid && dbNavHaz.Int64 != currentNavHaz) || 
	   (!dbNavHaz.Valid && currentNavHaz != 0) {
		return true
	}
	
	// Compare warps (only if we have current sector warp data and aren't in probe mode)
	if sectorIndex == p.currentSectorIndex && !p.probeMode {
		dbWarps := []sql.NullInt64{dbWarp1, dbWarp2, dbWarp3, dbWarp4, dbWarp5, dbWarp6}
		for i, currentWarp := range p.currentSectorWarps {
			if i < len(dbWarps) {
				dbWarp := dbWarps[i]
				currentWarpInt64 := int64(currentWarp)
				if (dbWarp.Valid && dbWarp.Int64 != currentWarpInt64) || 
				   (!dbWarp.Valid && currentWarp != 0) {
					return true
				}
			}
		}
	}
	
	return false // No changes detected
}