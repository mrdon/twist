package streaming

import (
	"strings"
	"twist/internal/debug"
)

// QuickStatsDisplay represents the state of quick stats parsing
type QuickStatsDisplay struct {
	Active   bool // True when we're parsing quick stats lines
	Complete bool // True when quick stats parsing is complete
}

// setupQuickStatsHandlers sets up handlers for quick stats parsing
func (p *TWXParser) setupQuickStatsHandlers() {
	// Quick stats patterns - detect lines containing separator character
	// Format: " Sect 1│Turns 1,600│Creds 10,000│Figs 30│Shlds 0│..."
	p.AddHandler("│", p.handleQuickStatsLine)
	p.AddHandler(" Ship", p.handleQuickStatsLine) // Ship line format variant
}

// initQuickStatsDisplay initializes quick stats parsing state
func (p *TWXParser) initQuickStatsDisplay() {
	p.quickStatsDisplay = QuickStatsDisplay{Active: false, Complete: false}
}

// startQuickStatsSession starts a new quick stats parsing session
func (p *TWXParser) startQuickStatsSession() {
	if p.quickStatsDisplay.Active {
		return // Already in session
	}

	p.quickStatsDisplay.Active = true
	p.quickStatsDisplay.Complete = false

	// Reset incomplete tracker from previous session
	if p.playerStatsTracker != nil && p.playerStatsTracker.HasUpdates() {
		debug.Info("QUICK_STATS: Discarding incomplete player stats tracker - new quick stats detected")
	}

	// Start new discovered field session
	p.playerStatsTracker = NewPlayerStatsTracker()
}

// completeQuickStatsSession finalizes quick stats parsing
func (p *TWXParser) completeQuickStatsSession() {
	if !p.quickStatsDisplay.Active {
		return
	}

	defer p.recoverFromPanic("completeQuickStatsSession")

	p.quickStatsDisplay.Active = false
	p.quickStatsDisplay.Complete = true

	// Execute SQL update with ONLY discovered fields
	if p.playerStatsTracker != nil && p.playerStatsTracker.HasUpdates() {
		err := p.playerStatsTracker.Execute(p.database.GetDB())
		if err != nil {
			debug.Info("QUICK_STATS: Failed to update player stats", "error", err)
			return
		}

		// Read complete, fresh data from database for API event
		if p.tuiAPI != nil {
			fullPlayerStats, err := p.database.GetPlayerStatsInfo()
			if err == nil {
				p.firePlayerStatsEventDirect(fullPlayerStats)
			} else {
				debug.Info("QUICK_STATS: Failed to read player stats info for API event", "error", err)
			}
		}

		// Reset tracker for next parsing session
		p.playerStatsTracker = nil
	}
}

// checkQuickStatsEnd checks if we should end the quick stats session
func (p *TWXParser) checkQuickStatsEnd(line string) {
	if !p.quickStatsDisplay.Active {
		return
	}

	// End quick stats on command prompt or blank line (quick stats are typically on one line)
	if strings.Contains(line, "Command [") || strings.TrimSpace(line) == "" {
		p.completeQuickStatsSession()
	}
}

// handleQuickStatsLine processes quick stats lines with separator-based format
// Format: " Sect 1│Turns 1,600│Creds 10,000│Figs 30│Shlds 0│Hlds 40│Ore 0│Org 0│Equ 0"
func (p *TWXParser) handleQuickStatsLine(line string) {
	defer p.recoverFromPanic("handleQuickStatsLine")

	// Start quick stats session
	p.startQuickStatsSession()

	// Skip lines that don't start with space (quick stats lines start with space)
	if !strings.HasPrefix(line, " ") {
		return
	}

	// Remove leading space
	if len(line) < 2 {
		return
	}
	content := line[1:]

	// Split on the separator character '│'
	var values []string
	if strings.Contains(content, "│") {
		values = strings.Split(content, "│")
	} else {
		// No recognized separator found - might be ship line or other format
		return
	}

	// Process each key-value pair
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		// Split each value into parts (key value)
		parts := strings.Fields(value)
		if len(parts) < 2 {
			continue
		}

		key := parts[0]
		val := parts[1]

		// Process each statistic using tracker
		p.processQuickStatField(key, val, parts)
	}

}

// processQuickStatField processes a single key-value pair from quick stats
func (p *TWXParser) processQuickStatField(key, val string, parts []string) {
	switch key {
	case "Turns":
		p.playerStatsTracker.SetCorp(0) // No corp displayed if player not member
		p.playerStatsTracker.SetTurns(p.parseIntSafeWithCommas(val))
	case "Creds":
		p.playerStatsTracker.SetCredits(p.parseIntSafeWithCommas(val))
	case "Figs":
		p.playerStatsTracker.SetFighters(p.parseIntSafeWithCommas(val))
	case "Shlds":
		p.playerStatsTracker.SetShields(p.parseIntSafeWithCommas(val))
	case "Crbo":
		p.playerStatsTracker.SetCorbomite(p.parseIntSafeWithCommas(val))
	case "Hlds":
		p.playerStatsTracker.SetTotalHolds(p.parseIntSafe(val))
	case "Ore":
		p.playerStatsTracker.SetOreHolds(p.parseIntSafe(val))
	case "Org":
		p.playerStatsTracker.SetOrgHolds(p.parseIntSafe(val))
	case "Equ":
		p.playerStatsTracker.SetEquHolds(p.parseIntSafe(val))
	case "Col":
		p.playerStatsTracker.SetColHolds(p.parseIntSafe(val))
	case "Phot":
		p.playerStatsTracker.SetPhotons(p.parseIntSafe(val))
	case "Armd":
		p.playerStatsTracker.SetArmids(p.parseIntSafe(val))
	case "Lmpt":
		p.playerStatsTracker.SetLimpets(p.parseIntSafe(val))
	case "GTorp":
		p.playerStatsTracker.SetGenTorps(p.parseIntSafe(val))
	case "Clks":
		p.playerStatsTracker.SetCloaks(p.parseIntSafe(val))
	case "Beacns":
		p.playerStatsTracker.SetBeacons(p.parseIntSafe(val))
	case "AtmDt":
		p.playerStatsTracker.SetAtomics(p.parseIntSafe(val))
	case "EPrb":
		p.playerStatsTracker.SetEprobes(p.parseIntSafe(val))
	case "MDis":
		p.playerStatsTracker.SetMineDisr(p.parseIntSafe(val))
	case "Aln":
		p.playerStatsTracker.SetAlignment(p.parseIntSafeWithCommas(val))
	case "Exp":
		p.playerStatsTracker.SetExperience(p.parseIntSafeWithCommas(val))
	case "Corp":
		p.playerStatsTracker.SetCorp(p.parseIntSafe(val))
	case "TWarp":
		if val == "No" {
			p.playerStatsTracker.SetTwarpType(0)
		} else {
			p.playerStatsTracker.SetTwarpType(p.parseIntSafe(val))
		}
	case "PsPrb":
		p.playerStatsTracker.SetPsychicProbe(val == "Yes")
	case "PlScn":
		p.playerStatsTracker.SetPlanetScanner(val == "Yes")
	case "LRS":
		switch val {
		case "None":
			p.playerStatsTracker.SetScanType(0)
		case "Dens":
			p.playerStatsTracker.SetScanType(1)
		case "Holo":
			p.playerStatsTracker.SetScanType(2)
		default:
			p.playerStatsTracker.SetScanType(0)
		}
	case "Ship":
		if len(parts) >= 3 {
			shipNumber := p.parseIntSafe(val)
			shipClass := parts[2]
			p.playerStatsTracker.SetShipNumber(shipNumber)
			p.playerStatsTracker.SetShipClass(shipClass)
		}
	case "Sect":
		// Update current sector from quick stats if available
		sectorNum := p.parseIntSafe(val)
		if sectorNum > 0 {
			p.currentSectorIndex = sectorNum
			p.playerStatsTracker.SetCurrentSector(sectorNum)
		}
	}
}
