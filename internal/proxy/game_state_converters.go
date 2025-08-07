package proxy

import (
	"twist/internal/api"
	"twist/internal/proxy/converter"
	"twist/internal/proxy/database"
)

// convertDatabaseSectorToAPI converts database TSector to API SectorInfo
func convertDatabaseSectorToAPI(sectorNum int, dbSector database.TSector) api.SectorInfo {
	// Use the converter package directly
	sectorInfo, err := converter.ConvertTSectorToSectorInfo(sectorNum, dbSector)
	if err != nil {
		
		// Fallback to basic conversion without port info
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
			HasPort:       false, // Will be determined separately
		}
	}
	
	return sectorInfo
}

// portClassToTypeString converts port class index to port type string
func portClassToTypeString(classIndex int) string {
	return converter.ConvertPortClassToString(classIndex)
}

// convertDatabasePlayerToAPI converts current sector and player name to API PlayerInfo
func convertDatabasePlayerToAPI(currentSector int, playerName string) api.PlayerInfo {
	return converter.ConvertToPlayerInfo(currentSector, playerName)
}