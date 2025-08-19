package streaming

import (
	"strings"
	"twist/internal/debug"
)

// InfoDisplay represents the state of info display parsing
type InfoDisplay struct {
	Active   bool // True when we're inside an <Info> display
	Complete bool // True when info parsing is complete
}

// InfoDisplayType tracks what type of info display we're parsing
type InfoDisplayType int

const (
	InfoDisplayNone InfoDisplayType = iota
	InfoDisplayPlayer
)

// setupInfoHandlers sets up handlers for info display parsing
func (p *TWXParser) setupInfoHandlers() {
	// Info display headers - detect start of info display
	p.AddHandler("<Info>", p.handleInfoDisplayStart)
	
	// Info display fields - only active when inside info display
	p.AddHandler("Trader Name    :", p.handleInfoTraderName)
	p.AddHandler("Rank and Exp   :", p.handleInfoRankExp)
	p.AddHandler("Ship Info      :", p.handleInfoShipInfo)
	p.AddHandler("Turns left     :", p.handleInfoTurnsLeft)
	p.AddHandler("Total Holds    :", p.handleInfoTotalHolds)
	p.AddHandler("Fighters       :", p.handleInfoFighters)
	p.AddHandler("Ether Probes   :", p.handleInfoEtherProbes)
	p.AddHandler("Credits        :", p.handleInfoCredits)
	p.AddHandler("Current Sector :", p.handleInfoCurrentSector)
}

// Add info display state to TWXParser
func (p *TWXParser) initInfoDisplay() {
	p.infoDisplay = InfoDisplay{Active: false, Complete: false}
}

// handleInfoDisplayStart detects the start of an info display
func (p *TWXParser) handleInfoDisplayStart(line string) {
	defer p.recoverFromPanic("handleInfoDisplayStart")
	
	p.infoDisplay.Active = true
	p.infoDisplay.Complete = false
	
	// Don't reset player stats - preserve existing data from quick stats, etc.
	// Info display will update/overwrite specific fields as they're parsed
}

// checkInfoDisplayEnd checks if we've reached the end of info display
func (p *TWXParser) checkInfoDisplayEnd(line string) {
	if !p.infoDisplay.Active {
		return
	}
	
	// Info display ends with command prompt (not blank lines, as there are blank lines within the display)
	if strings.Contains(line, "Command [") {
		p.completeInfoDisplay()
	}
}

// completeInfoDisplay finalizes the info display parsing
func (p *TWXParser) completeInfoDisplay() {
	defer p.recoverFromPanic("completeInfoDisplay")
	
	if !p.infoDisplay.Active {
		return
	}
	
	p.infoDisplay.Active = false
	p.infoDisplay.Complete = true
	
	// Save player stats to database
	p.savePlayerStats()
	
	// Fire player stats event
	p.firePlayerStatsEvent(p.playerStats)
}

// handleInfoTraderName parses trader name from info display
func (p *TWXParser) handleInfoTraderName(line string) {
	if !p.infoDisplay.Active {
		return
	}
	
	defer p.recoverFromPanic("handleInfoTraderName")
	
	// Parse format: "Trader Name    : Private 1st Class mrdon"
	if len(line) > 17 { // "Trader Name    : ".length = 17
		traderName := strings.TrimSpace(line[17:])
		// Store trader name if needed
		_ = traderName
	}
}

// handleInfoRankExp parses rank and experience from info display
func (p *TWXParser) handleInfoRankExp(line string) {
	if !p.infoDisplay.Active {
		return
	}
	
	defer p.recoverFromPanic("handleInfoRankExp")
	
	// Parse format: "Rank and Exp   : 4 points, Alignment=28 Tolerant"
	if len(line) > 17 { // "Rank and Exp   : ".length = 17
		rankExpInfo := strings.TrimSpace(line[17:])
		
		// Extract experience points
		pointsPos := strings.Index(rankExpInfo, " points")
		if pointsPos > 0 {
			expStr := strings.TrimSpace(rankExpInfo[:pointsPos])
			experience := p.parseIntSafe(expStr)
			p.playerStats.Experience = experience
		}
		
		// Extract alignment
		alignPos := strings.Index(rankExpInfo, "Alignment=")
		if alignPos >= 0 {
			alignStart := alignPos + 10 // After "Alignment="
			alignEnd := strings.Index(rankExpInfo[alignStart:], " ")
			if alignEnd == -1 {
				alignEnd = len(rankExpInfo) - alignStart
			}
			
			if alignEnd > 0 {
				alignStr := rankExpInfo[alignStart : alignStart+alignEnd]
				alignment := p.parseIntSafe(alignStr)
				p.playerStats.Alignment = alignment
			}
		}
	}
}

// handleInfoShipInfo parses ship information from info display
func (p *TWXParser) handleInfoShipInfo(line string) {
	if !p.infoDisplay.Active {
		return
	}
	
	defer p.recoverFromPanic("handleInfoShipInfo")
	
	// Parse format: "Ship Info      : Le Richelieu Merchant Cruiser Ported=3 Kills=0"
	if len(line) > 17 { // "Ship Info      : ".length = 17
		// Don't try to parse ship class from info display - too many variations
		// Quick stats already provides ShipClass in abbreviated form (like "MerCru")
		// Just preserve existing ShipClass and set defaults for other fields
		
		// Ship number defaults to 1 if not specified elsewhere
		if p.playerStats.ShipNumber == 0 {
			p.playerStats.ShipNumber = 1
		}
	}
}

// handleInfoTurnsLeft parses turns left from info display
func (p *TWXParser) handleInfoTurnsLeft(line string) {
	if !p.infoDisplay.Active {
		return
	}
	
	defer p.recoverFromPanic("handleInfoTurnsLeft")
	
	// Parse format: "Turns left     : 19993"
	if len(line) > 17 { // "Turns left     : ".length = 17
		turnsStr := strings.TrimSpace(line[17:])
		// Remove commas like TWX does
		turnsStr = strings.ReplaceAll(turnsStr, ",", "")
		turns := p.parseIntSafe(turnsStr)
		p.playerStats.Turns = turns
	}
}

// handleInfoTotalHolds parses cargo holds from info display
func (p *TWXParser) handleInfoTotalHolds(line string) {
	if !p.infoDisplay.Active {
		return
	}
	
	defer p.recoverFromPanic("handleInfoTotalHolds")
	
	// Parse format: "Total Holds    : 20 - Fuel Ore=2 Organics=3 Empty=15"
	if len(line) > 17 { // "Total Holds    : ".length = 17
		holdsInfo := strings.TrimSpace(line[17:])
		
		// Extract total holds (before the dash)
		dashPos := strings.Index(holdsInfo, " -")
		if dashPos > 0 {
			totalStr := strings.TrimSpace(holdsInfo[:dashPos])
			totalHolds := p.parseIntSafe(totalStr)
			p.playerStats.TotalHolds = totalHolds
			
			// Parse cargo breakdown
			cargoInfo := strings.TrimSpace(holdsInfo[dashPos+2:]) // After " - "
			p.parseCargoHolds(cargoInfo)
		}
	}
}

// parseCargoHolds parses the cargo breakdown from holds line
func (p *TWXParser) parseCargoHolds(cargoInfo string) {
	defer p.recoverFromPanic("parseCargoHolds")
	
	// Parse format: "Fuel Ore=2 Organics=3 Empty=15"
	
	// Extract Fuel Ore
	if orePos := strings.Index(cargoInfo, "Fuel Ore="); orePos >= 0 {
		oreStart := orePos + 9 // After "Fuel Ore="
		oreEnd := strings.IndexAny(cargoInfo[oreStart:], " \t")
		if oreEnd == -1 {
			oreEnd = len(cargoInfo) - oreStart
		}
		if oreEnd > 0 {
			oreStr := cargoInfo[oreStart : oreStart+oreEnd]
			oreHolds := p.parseIntSafe(oreStr)
			p.playerStats.OreHolds = oreHolds
		}
	}
	
	// Extract Organics
	if orgPos := strings.Index(cargoInfo, "Organics="); orgPos >= 0 {
		orgStart := orgPos + 9 // After "Organics="
		orgEnd := strings.IndexAny(cargoInfo[orgStart:], " \t")
		if orgEnd == -1 {
			orgEnd = len(cargoInfo) - orgStart
		}
		if orgEnd > 0 {
			orgStr := cargoInfo[orgStart : orgStart+orgEnd]
			orgHolds := p.parseIntSafe(orgStr)
			p.playerStats.OrgHolds = orgHolds
		}
	}
	
	// Equipment holds would be parsed similarly if present
	p.playerStats.EquHolds = 0 // Default as not shown in this format
	
	// Colonist holds would be parsed similarly if present  
	p.playerStats.ColHolds = 0 // Default as not shown in this format
}

// handleInfoFighters parses fighters from info display
func (p *TWXParser) handleInfoFighters(line string) {
	if !p.infoDisplay.Active {
		return
	}
	
	defer p.recoverFromPanic("handleInfoFighters")
	
	// Parse format: "Fighters       : 2,500"
	if len(line) > 17 { // "Fighters       : ".length = 17
		fightersStr := strings.TrimSpace(line[17:])
		// Remove commas like TWX does
		fightersStr = strings.ReplaceAll(fightersStr, ",", "")
		fighters := p.parseIntSafe(fightersStr)
		p.playerStats.Fighters = fighters
	}
}

// handleInfoEtherProbes parses ether probes from info display
func (p *TWXParser) handleInfoEtherProbes(line string) {
	if !p.infoDisplay.Active {
		return
	}
	
	defer p.recoverFromPanic("handleInfoEtherProbes")
	
	// Parse format: "Ether Probes   : 25"
	if len(line) > 17 { // "Ether Probes   : ".length = 17
		eprobesStr := strings.TrimSpace(line[17:])
		// Remove commas like TWX does
		eprobesStr = strings.ReplaceAll(eprobesStr, ",", "")
		eprobes := p.parseIntSafe(eprobesStr)
		p.playerStats.Eprobes = eprobes
	}
}

// handleInfoCredits parses credits from info display
func (p *TWXParser) handleInfoCredits(line string) {
	if !p.infoDisplay.Active {
		return
	}
	
	defer p.recoverFromPanic("handleInfoCredits")
	
	// Parse format: "Credits        : 140,585"
	if len(line) > 17 { // "Credits        : ".length = 17
		creditsStr := strings.TrimSpace(line[17:])
		// Remove commas like TWX does
		creditsStr = strings.ReplaceAll(creditsStr, ",", "")
		credits := p.parseIntSafe(creditsStr)
		p.playerStats.Credits = credits
		
		// Credits is typically the last field in info display, so trigger completion
		p.completeInfoDisplay()
	}
}

// handleInfoCurrentSector parses current sector from info display
func (p *TWXParser) handleInfoCurrentSector(line string) {
	if !p.infoDisplay.Active {
		return
	}
	
	defer p.recoverFromPanic("handleInfoCurrentSector")
	
	// Parse format: "Current Sector : 190"
	if len(line) > 17 { // "Current Sector : ".length = 17
		sectorStr := strings.TrimSpace(line[17:])
		sectorNum := p.parseIntSafe(sectorStr)
		// Update current sector if valid
		if sectorNum > 0 {
			p.currentSectorIndex = sectorNum
		}
	}
}

// savePlayerStats saves the parsed player stats to the database
func (p *TWXParser) savePlayerStats() {
	defer p.recoverFromPanic("savePlayerStats")
	
	if p.database == nil {
		return
	}
	
	// Convert to database format and save using existing converter
	converter := NewPlayerStatsConverter()
	dbPlayerStats := converter.ToDatabase(p.playerStats)
	
	err := p.database.SavePlayerStats(dbPlayerStats)
	if err != nil {
		debug.Log("INFO_PARSER: Error saving player stats: %v", err)
	}
}