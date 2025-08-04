package streaming

import (
	"fmt"
	"twist/internal/api"
	"twist/internal/debug"
)

// database_integration.go - Clean database integration points
// This file focuses solely on connecting the parser to the database layer

// saveSectorToDatabase saves the current sector data to database
func (p *TWXParser) saveSectorToDatabase() error {
	if p.currentSectorIndex <= 0 {
		debug.Log("TWXParser: Invalid current sector index %d, skipping save", p.currentSectorIndex)
		return nil
	}
	
	// Debug: Check if parser's database instance is valid
	if p.database == nil {
		debug.Log("TWXParser: Database instance is nil")
		return fmt.Errorf("parser database instance is nil")
	}
	
	// Debug: Test parser's database connection
	if db := p.database.GetDB(); db != nil {
		var tableCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sectors'").Scan(&tableCount); err != nil {
			debug.Log("TWXParser: Failed to check parser's database tables: %v", err)
			return fmt.Errorf("parser's database connection is broken: %w", err)
		}
		if tableCount == 0 {
			debug.Log("TWXParser: Parser's database has no sectors table")
			return fmt.Errorf("parser's database has no sectors table")
		}
		debug.Log("TWXParser: Parser's database has %d sectors table(s)", tableCount)
	} else {
		debug.Log("TWXParser: Parser's database GetDB() returned nil")
		return fmt.Errorf("parser's database GetDB() returned nil")
	}
	
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
		debug.Log("TWXParser: Failed to save sector %d to database with collections: %v", p.currentSectorIndex, err)
		return err
	}
	
	debug.Log("TWXParser: Saved sector %d to database", p.currentSectorIndex)
	
	// Notify TUI API if available
	if p.tuiAPI != nil {
		sectorInfo := p.buildSectorInfo(sectorData)
		p.tuiAPI.OnCurrentSectorChanged(sectorInfo)
		debug.Log("TWXParser: Notified TUI API of sector %d change", p.currentSectorIndex)
	}
	
	return nil
}

// savePlayerStatsToDatabase saves current player stats to database
func (p *TWXParser) savePlayerStatsToDatabase() error {
	// Convert to database format using converter
	converter := NewPlayerStatsConverter()
	dbStats := converter.ToDatabase(p.playerStats)
	
	// Save to database
	if err := p.database.SavePlayerStats(dbStats); err != nil {
		debug.Log("TWXParser: Failed to save player stats to database: %v", err)
		return err
	}
	
	debug.Log("TWXParser: Saved player stats to database - Turns: %d, Credits: %d", 
		p.playerStats.Turns, p.playerStats.Credits)
	return nil
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
	
	return api.SectorInfo{
		Number:        sectorData.Index,
		NavHaz:        sectorData.NavHaz,
		HasTraders:    len(sectorData.Traders),
		Constellation: sectorData.Constellation,
		Beacon:        sectorData.Beacon,
		Warps:         warps,
	}
}