package proxy

import (
	"twist/internal/api"
	"twist/internal/proxy/database"
)

// convertDatabaseSectorToAPI converts database TSector to API SectorInfo
func convertDatabaseSectorToAPI(sectorNum int, dbSector database.TSector) api.SectorInfo {
	return api.SectorInfo{
		Number:        sectorNum,
		NavHaz:        dbSector.NavHaz,
		HasTraders:    len(dbSector.Traders),
		Constellation: dbSector.Constellation,
		Beacon:        dbSector.Beacon,
	}
}

// convertDatabasePlayerToAPI converts current sector and player name to API PlayerInfo
func convertDatabasePlayerToAPI(currentSector int, playerName string) api.PlayerInfo {
	return api.PlayerInfo{
		Name:          playerName,
		CurrentSector: currentSector,
	}
}