package proxy

import (
	"twist/internal/api"
	"twist/internal/proxy/database"
)

// convertDatabaseSectorToAPI converts database TSector to API SectorInfo
func convertDatabaseSectorToAPI(sectorNum int, dbSector database.TSector) api.SectorInfo {
	// Extract valid warp connections from the Warp array
	var warps []int
	for i := 0; i < int(dbSector.Warps) && i < 6; i++ {
		if dbSector.Warp[i] > 0 {
			warps = append(warps, dbSector.Warp[i])
		}
	}
	
	return api.SectorInfo{
		Number:        sectorNum,
		NavHaz:        dbSector.NavHaz,
		HasTraders:    len(dbSector.Traders),
		Constellation: dbSector.Constellation,
		Beacon:        dbSector.Beacon,
		Warps:         warps,
	}
}

// convertDatabasePlayerToAPI converts current sector and player name to API PlayerInfo
func convertDatabasePlayerToAPI(currentSector int, playerName string) api.PlayerInfo {
	return api.PlayerInfo{
		Name:          playerName,
		CurrentSector: currentSector,
	}
}