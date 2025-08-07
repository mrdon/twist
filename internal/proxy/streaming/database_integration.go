package streaming

import (
	"fmt"
	"twist/internal/api"
)

// database_integration.go - Clean database integration points
// This file focuses solely on connecting the parser to the database layer

// saveSectorToDatabase saves the current sector data to database
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
	}
	
	
	// Update current sector in player stats (like TWX Database.pas)
	p.playerStats.CurrentSector = p.currentSectorIndex
	if err := p.savePlayerStatsToDatabase(); err != nil {
	} else {
	}
	
	// Notify TUI API if available
	if p.tuiAPI != nil {
		sectorInfo := p.buildSectorInfo(sectorData)
		p.tuiAPI.OnCurrentSectorChanged(sectorInfo)
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
		return err
	}
	
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