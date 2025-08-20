package database

// Helper methods for safe warp array modifications

// SetSectorWarp safely sets a warp connection and updates the warp count
func (d *SQLiteDatabase) SetSectorWarp(sector *TSector, warpIndex int, targetSector int) {
	if warpIndex >= 0 && warpIndex < 6 {
		sector.Warp[warpIndex] = targetSector
		// Recalculate warp count
		calculatedWarps := 0
		for _, warp := range sector.Warp {
			if warp > 0 {
				calculatedWarps++
			}
		}
		sector.Warps = calculatedWarps
	}
}

// ClearSectorWarp safely clears a warp connection and updates the warp count
func (d *SQLiteDatabase) ClearSectorWarp(sector *TSector, warpIndex int) {
	if warpIndex >= 0 && warpIndex < 6 {
		sector.Warp[warpIndex] = 0
		// Recalculate warp count
		calculatedWarps := 0
		for _, warp := range sector.Warp {
			if warp > 0 {
				calculatedWarps++
			}
		}
		sector.Warps = calculatedWarps
	}
}

// SetSectorWarps safely sets the entire warp array and updates the count
func (d *SQLiteDatabase) SetSectorWarps(sector *TSector, warps [6]int) {
	sector.Warp = warps
	// Recalculate warp count
	calculatedWarps := 0
	for _, warp := range sector.Warp {
		if warp > 0 {
			calculatedWarps++
		}
	}
	sector.Warps = calculatedWarps
}

// UpdateWarpCount recalculates and updates the warp count for a sector
func UpdateWarpCount(sector *TSector) {
	calculatedWarps := 0
	for _, warp := range sector.Warp {
		if warp > 0 {
			calculatedWarps++
		}
	}
	// Only update if we have actual warp connections, otherwise preserve existing count
	if calculatedWarps > 0 {
		sector.Warps = calculatedWarps
	}
}
