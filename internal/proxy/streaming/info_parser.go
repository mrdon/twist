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

	// Reset incomplete tracker from previous session
	if p.playerStatsTracker != nil && p.playerStatsTracker.HasUpdates() {
		debug.Log("INFO_PARSER: Discarding incomplete player stats tracker - new info display detected")
	}

	// Start new discovered field session
	p.playerStatsTracker = NewPlayerStatsTracker()
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

	// Execute SQL update with ONLY discovered fields
	if p.playerStatsTracker != nil && p.playerStatsTracker.HasUpdates() {
		err := p.playerStatsTracker.Execute(p.database.GetDB())
		if err != nil {
			debug.Log("INFO_PARSER: Failed to update player stats: %v", err)
			return
		}

		// Read complete, fresh data from database for API event
		if p.tuiAPI != nil {
			fullPlayerStats, err := p.database.GetPlayerStatsInfo()
			if err == nil {
				p.firePlayerStatsEventDirect(fullPlayerStats)
			} else {
				debug.Log("INFO_PARSER: Failed to read player stats info for API event: %v", err)
			}
		}

		// Reset tracker for next parsing session
		p.playerStatsTracker = nil
	}
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
			if p.playerStatsTracker != nil {
				p.playerStatsTracker.SetExperience(experience)
			}
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
				if p.playerStatsTracker != nil {
					p.playerStatsTracker.SetAlignment(alignment)
				}
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
		if p.playerStatsTracker != nil {
			p.playerStatsTracker.SetShipNumber(1)
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
		if p.playerStatsTracker != nil {
			p.playerStatsTracker.SetTurns(turns)
		}
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
			if p.playerStatsTracker != nil {
				p.playerStatsTracker.SetTotalHolds(totalHolds)
			}

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
			if p.playerStatsTracker != nil {
				p.playerStatsTracker.SetOreHolds(oreHolds)
			}
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
			if p.playerStatsTracker != nil {
				p.playerStatsTracker.SetOrgHolds(orgHolds)
			}
		}
	}

	// Equipment and Colonist holds set to 0 as defaults (not shown in this format)
	if p.playerStatsTracker != nil {
		p.playerStatsTracker.SetEquHolds(0)
		p.playerStatsTracker.SetColHolds(0)
	}
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
		if p.playerStatsTracker != nil {
			p.playerStatsTracker.SetFighters(fighters)
		}
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
		if p.playerStatsTracker != nil {
			p.playerStatsTracker.SetEprobes(eprobes)
		}
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
		if p.playerStatsTracker != nil {
			p.playerStatsTracker.SetCredits(credits)
		}

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
			if p.playerStatsTracker != nil {
				p.playerStatsTracker.SetCurrentSector(sectorNum)
			}
		}
	}
}
