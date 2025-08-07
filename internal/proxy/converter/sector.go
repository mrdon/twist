package converter

import (
	"twist/internal/api"
	"twist/internal/proxy/database"
)

// ConvertTSectorToSectorInfo converts database TSector to API SectorInfo
func ConvertTSectorToSectorInfo(sectorNum int, dbSector database.TSector) (api.SectorInfo, error) {
	
	// Extract valid warp connections from the Warp array
	var warps []int
	for i := 0; i < int(dbSector.Warps) && i < 6; i++ {
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
	}
	
	return sectorInfo, nil
}