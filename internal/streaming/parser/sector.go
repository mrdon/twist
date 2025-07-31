package parser

import (
	"strings"
	"twist/internal/database"
)

// SectorProcessor handles sector-specific parsing logic
type SectorProcessor struct {
	ctx   *ParserContext
	utils *ParseUtils
}

// NewSectorProcessor creates a new sector processor
func NewSectorProcessor(ctx *ParserContext) *SectorProcessor {
	return &SectorProcessor{
		ctx:   ctx,
		utils: NewParseUtils(ctx),
	}
}

// ProcessSectorLine processes a line in sector display mode
func (sp *SectorProcessor) ProcessSectorLine(line string) {
	cleanLine := sp.utils.StripANSI(line)
	
	// Handle different sector positions
	switch sp.ctx.State.SectorPosition {
	case SpNormal:
		sp.processSectorHeader(cleanLine)
	case SpPorts:
		sp.processPortsSection(cleanLine)
	case SpPlanets:
		sp.processPlanetsSection(cleanLine)
	case SpShips:
		sp.processShipsSection(cleanLine)
	case SpMines:
		sp.processMinesSection(cleanLine)
	case SpTraders:
		sp.processTradersSection(cleanLine)
	}
}

// processSectorHeader processes the main sector information
func (sp *SectorProcessor) processSectorHeader(line string) {
	// TWX sector header parsing logic
	if strings.Contains(line, "Sector :") {
		sp.initializeSectorData(line)
	} else if strings.Contains(line, "Warps to Sector(s) :") {
		sp.processWarpLine(line)
	} else if strings.Contains(line, "Class") {
		sp.processSectorClass(line)
	}
}

// initializeSectorData initializes a new sector from header line
func (sp *SectorProcessor) initializeSectorData(line string) {
	// Extract sector number from "Sector : 1234"
	if idx := strings.Index(line, "Sector :"); idx != -1 {
		sectorStr := strings.TrimSpace(line[idx+8:])
		if spaceIdx := strings.Index(sectorStr, " "); spaceIdx != -1 {
			sectorStr = sectorStr[:spaceIdx]
		}
		
		sectorNum := sp.utils.StrToIntSafe(sectorStr)
		if sectorNum > 0 {
			sp.ctx.State.CurrentSectorIndex = sectorNum
			sp.ctx.State.CurrentSector = &database.TSector{}
		}
	}
}

// processWarpLine processes warp destination information
func (sp *SectorProcessor) processWarpLine(line string) {
	// Extract warp destinations from "Warps to Sector(s) : 1 2 3"
	if idx := strings.Index(line, ":"); idx != -1 {
		warpStr := strings.TrimSpace(line[idx+1:])
		warpFields := strings.Fields(warpStr)
		
		for _, warpField := range warpFields {
			if warpNum := sp.utils.StrToIntSafe(warpField); warpNum > 0 {
				sp.addWarp(sp.ctx.State.CurrentSectorIndex, warpNum)
			}
		}
	}
}

// processSectorClass processes sector class information
func (sp *SectorProcessor) processSectorClass(line string) {
	if sp.ctx.State.CurrentSector != nil {
		// Parse sector class information - store in NavHaz for now
		if strings.Contains(line, "Class") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "Class" && i+1 < len(fields) {
					sp.ctx.State.CurrentSector.NavHaz = sp.utils.StrToIntSafe(fields[i+1])
					break
				}
			}
		}
	}
}

// addWarp adds a warp connection to the database
func (sp *SectorProcessor) addWarp(fromSector, toSector int) {
	// For now just note the warp - would need proper warp storage implementation
}

// processPortsSection processes the ports section
func (sp *SectorProcessor) processPortsSection(line string) {
	// Port parsing logic would go here
}

// processPlanetsSection processes the planets section
func (sp *SectorProcessor) processPlanetsSection(line string) {
	// Planet parsing logic would go here
}

// processShipsSection processes the ships section
func (sp *SectorProcessor) processShipsSection(line string) {
	// Ship parsing logic would go here
}

// processMinesSection processes the mines section
func (sp *SectorProcessor) processMinesSection(line string) {
	// Mine parsing logic would go here
}

// processTradersSection processes the traders section
func (sp *SectorProcessor) processTradersSection(line string) {
	// Trader parsing logic would go here
}

// SectorCompleted finalizes the current sector data
func (sp *SectorProcessor) SectorCompleted() {
	if sp.ctx.State.CurrentSector != nil && !sp.ctx.State.SectorSaved {
		if err := sp.ctx.DB.SaveSector(*sp.ctx.State.CurrentSector, sp.ctx.State.CurrentSectorIndex); err == nil {
			sp.ctx.State.SectorSaved = true
		}
	}
}