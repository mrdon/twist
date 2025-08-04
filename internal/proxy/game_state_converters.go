package proxy

import (
	"twist/internal/api"
	"twist/internal/debug"
	"twist/internal/proxy/database"
)

// convertDatabaseSectorToAPI converts database TSector to API SectorInfo
func convertDatabaseSectorToAPI(sectorNum int, dbSector database.TSector) api.SectorInfo {
	debug.Log("Converter: Converting sector %d - dbSector.Warps=%d, dbSector.Warp=%v", 
		sectorNum, dbSector.Warps, dbSector.Warp)
	
	// Extract valid warp connections from the Warp array
	var warps []int
	for i := 0; i < int(dbSector.Warps) && i < 6; i++ {
		if dbSector.Warp[i] > 0 {
			debug.Log("Converter: Adding warp %d (index %d) for sector %d", dbSector.Warp[i], i, sectorNum)
			warps = append(warps, dbSector.Warp[i])
		} else {
			debug.Log("Converter: Skipping invalid warp %d (index %d) for sector %d", dbSector.Warp[i], i, sectorNum)
		}
	}
	
	debug.Log("Converter: Final warps for sector %d: %v (extracted %d out of %d)", 
		sectorNum, warps, len(warps), dbSector.Warps)
	
	// Determine port type from class index
	portType := portClassToTypeString(dbSector.SPort.ClassIndex)
	
	return api.SectorInfo{
		Number:        sectorNum,
		NavHaz:        dbSector.NavHaz,
		HasTraders:    len(dbSector.Traders),
		Constellation: dbSector.Constellation,
		Beacon:        dbSector.Beacon,
		Warps:         warps,
		PortType:      portType,
	}
}

// portClassToTypeString converts port class index to port type string
func portClassToTypeString(classIndex int) string {
	switch classIndex {
	case 1:
		return "BBS"
	case 2:
		return "BSB"
	case 3:
		return "SBB"
	case 4:
		return "SSB"
	case 5:
		return "SBS"
	case 6:
		return "BSS"
	case 7:
		return "SSS"
	case 8:
		return "BBB"
	case 9:
		return "STD" // Special case for stardock/federation ports
	default:
		return ""    // No port or unknown class
	}
}

// convertDatabasePlayerToAPI converts current sector and player name to API PlayerInfo
func convertDatabasePlayerToAPI(currentSector int, playerName string) api.PlayerInfo {
	return api.PlayerInfo{
		Name:          playerName,
		CurrentSector: currentSector,
	}
}