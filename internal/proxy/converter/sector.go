package converter

import (
	"twist/internal/api"
	"twist/internal/proxy/database"
)

// ConvertTSectorToSectorInfo converts database TSector to API SectorInfo
func ConvertTSectorToSectorInfo(sectorNum int, dbSector database.TSector) (api.SectorInfo, error) {
	
	// Extract valid warp connections from the Warp array
	var warps []int
	// Iterate through all 6 warp slots and collect non-zero values
	// Don't rely on dbSector.Warps count as it may be incorrect
	for i := 0; i < 6; i++ {
		if dbSector.Warp[i] > 0 {
			warps = append(warps, dbSector.Warp[i])
		}
	}
	
	sectorInfo := api.SectorInfo{
		Number:        sectorNum,
		NavHaz:        dbSector.NavHaz,
		HasTraders:    len(dbSector.Traders),
		Constellation: dbSector.Constellation,
		Beacon:        dbSector.Beacon,
		Warps:         warps,
		HasPort:       false, // Will be set when port data is loaded separately
		Visited:       dbSector.Explored == 3, // Only EtHolo (3) counts as truly visited
	}
	
	return sectorInfo, nil
}