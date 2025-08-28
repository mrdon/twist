package streaming

import (
	"strconv"
	"strings"
)

// isNumeric checks if a string represents a valid number
func isNumeric(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	// Remove commas for number parsing
	s = strings.ReplaceAll(s, ",", "")
	_, err := strconv.Atoi(s)
	return err == nil
}

// ============================================================================
// DETAILED SECTOR DATA PARSING (Mirrors TWX Pascal sector parsing logic)
// ============================================================================

// parseSectorShips handles detailed ship parsing from sector data
func (p *TWXParser) parseSectorShips(line string) {

	// Parse format: "Ships   : Enterprise [Owned by Kirk], w/ 500 ftrs,"
	//               "        (Constitution Class Cruiser)"

	if !strings.HasPrefix(line, "Ships   : ") {
		// Continuation line - handle ship type
		p.handleShipContinuation(line)
		return
	}

	shipInfo := line[10:] // Remove "Ships   : "
	p.parseShipLine(shipInfo)
}

// parseShipLine parses individual ship information (mirrors Pascal logic from lines 671-690)
func (p *TWXParser) parseShipLine(shipInfo string) {
	// Phase 4.5: Ships tracked via collection trackers
	if p.sectorPosition != SectorPosShips {
		p.sectorPosition = SectorPosShips
	}

	// Parse ship name and owner (mirrors Pascal exact logic)
	ship := ShipInfo{}

	// Pascal: I := Pos('[Owned by]', Line);
	// Pascal: FCurrentShip.Name := Copy(Line, 11, I - 12);
	ownedByPos := strings.Index(shipInfo, "[Owned by")
	if ownedByPos > 0 {
		// Extract ship name (everything before [Owned by)
		ship.Name = strings.TrimSpace(shipInfo[:ownedByPos])

		// Extract owner from "[Owned by OWNER]"
		ownerStart := ownedByPos + 10 // After "[Owned by "
		ownerEnd := strings.Index(shipInfo[ownerStart:], "]")
		if ownerEnd > 0 {
			ship.Owner = shipInfo[ownerStart : ownerStart+ownerEnd]
		}
	} else {
		// Fallback: try generic bracket parsing
		bracketStart := strings.Index(shipInfo, "[")
		if bracketStart > 0 {
			ship.Name = strings.TrimSpace(shipInfo[:bracketStart])

			// Extract owner (between [ and ])
			bracketEnd := strings.Index(shipInfo, "]")
			if bracketEnd > bracketStart {
				ownerInfo := shipInfo[bracketStart+1 : bracketEnd]
				if strings.HasPrefix(ownerInfo, "Owned by ") {
					ship.Owner = ownerInfo[9:] // Remove "Owned by "
				} else {
					ship.Owner = ownerInfo
				}
			}
		} else {
			// No brackets found - extract what we can
			commaPos := strings.Index(shipInfo, ",")
			if commaPos > 0 {
				ship.Name = strings.TrimSpace(shipInfo[:commaPos])
			} else {
				ship.Name = strings.TrimSpace(shipInfo)
			}
		}
	}

	// Pascal: I := Pos(', w/', Line);
	// Pascal: S := Copy(Line, I + 5, Pos(' ftrs,', Line) - I - 5);
	// Pascal: StripChar(S, ',');
	// Pascal: FCurrentShip.Figs := StrToIntSafe(S);
	fightersPos := strings.Index(shipInfo, ", w/")
	if fightersPos >= 0 {
		// Find end of fighter count (before ' ftrs')
		ftrsPos := strings.Index(shipInfo, " ftrs")
		if ftrsPos > fightersPos {
			fighterStr := shipInfo[fightersPos+4 : ftrsPos] // After ", w/"
			// Pascal: StripChar(S, ',');
			fighterStr = strings.ReplaceAll(fighterStr, ",", "")

			// Validate fighter count (must be non-negative)
			fighterCount := p.parseIntSafe(fighterStr)
			if fighterCount >= 0 {
				ship.Fighters = fighterCount
			} else {
				ship.Fighters = 0
			}
		}
	}

	// Parse alignment if present (Pascal doesn't show this explicitly, but TWX supports it)
	// Look for alignment indicators like "(Good)", "(Evil)", "(Neutral)"
	if parenStart := strings.Index(shipInfo, "("); parenStart >= 0 {
		parenEnd := strings.Index(shipInfo[parenStart:], ")")
		if parenEnd > 0 {
			alignmentCandidate := strings.TrimSpace(shipInfo[parenStart+1 : parenStart+parenEnd])
			// Check if this looks like an alignment (not a ship type or other info)
			alignmentLower := strings.ToLower(alignmentCandidate)
			if alignmentLower == "good" || alignmentLower == "evil" || alignmentLower == "neutral" ||
				alignmentLower == "outlaw" || alignmentLower == "criminal" {
				ship.Alignment = alignmentCandidate
			}
		}
	}

	// Validate parsed ship data
	if ship.Name == "" {
	}

	if ship.Owner == "" {
	}

	// Phase 4.5: Ships tracked via collection trackers only
	if p.sectorCollections != nil {
		p.sectorCollections.AddShip(ship.Name, ship.Owner, ship.ShipType, ship.Fighters)
	}
}

// handleShipContinuation handles continuation lines for ship data (mirrors Pascal lines 113-117)
func (p *TWXParser) handleShipContinuation(line string) {
	// Phase 4.5: Ship continuation handling with collection trackers
	line = strings.TrimSpace(line)

	// Handle different types of ship continuation lines:
	// 1. New ship entries that appear in continuation lines
	if strings.Contains(line, "[Owned by") {
		p.parseShipLine(line)
		return
	}

	// 2. Ship data that might be split across lines (ship types, alignments, etc.)
	if strings.Contains(line, "(") && strings.Contains(line, ")") {
		// This might be ship type or alignment information
		// For collection tracker approach, we only care about complete ship entries
		// Incomplete ship data spanning lines is handled by parseShipLine when it gets the full line
		return
	}

	// 3. Any other line that looks like ship data
	if strings.Contains(line, " ftrs") || strings.Contains(line, ", w/") {
		// This line contains fighter information, treat as ship line
		p.parseShipLine(line)
		return
	}
}

// parseSectorTraders handles detailed trader parsing from sector data (mirrors Pascal logic)
func (p *TWXParser) parseSectorTraders(line string) {

	// Parse format: "Traders : Captain Kirk, w/ 1,000 ftrs"
	// TWX Pascal logic: lines 713-722 in Process.pas

	if !strings.HasPrefix(line, "Traders : ") {
		return
	}

	// Phase 4.5: Traders tracked via collection trackers
	p.sectorPosition = SectorPosTraders

	// Mirror Pascal logic exactly:
	// I := Pos(', w/', Line);
	// FCurrentTrader.Name := Copy(Line, 11, I - 11);
	// S := Copy(Line, I + 5, Pos(' ftrs', Line) - I - 5);

	traderInfo := line[10:] // Remove "Traders : "

	// Handle multiple trader formats and edge cases
	trader := TraderInfo{}

	// First, parse alignment if present (look for alignment indicators)
	workingInfo := traderInfo
	if parenStart := strings.Index(traderInfo, "("); parenStart >= 0 {
		parenEnd := strings.Index(traderInfo[parenStart:], ")")
		if parenEnd > 0 {
			alignmentCandidate := strings.TrimSpace(traderInfo[parenStart+1 : parenStart+parenEnd])
			alignmentLower := strings.ToLower(alignmentCandidate)
			if alignmentLower == "good" || alignmentLower == "evil" || alignmentLower == "neutral" ||
				alignmentLower == "outlaw" || alignmentLower == "criminal" {
				trader.Alignment = alignmentCandidate
				// Remove alignment from working info for further parsing
				workingInfo = strings.TrimSpace(traderInfo[:parenStart]) + strings.TrimSpace(traderInfo[parenStart+parenEnd+1:])
			}
		}
	}

	fighterPos := strings.Index(workingInfo, ", w/")
	if fighterPos == -1 {
		// No fighter info - just extract trader name
		trader.Name = strings.TrimSpace(workingInfo)
		trader.Fighters = 0
	} else {
		// Extract trader name (from start to ', w/' position)
		trader.Name = strings.TrimSpace(workingInfo[:fighterPos])

		// Extract fighter count (from after ', w/' to ' ftrs')
		fighterStart := fighterPos + 4 // After ", w/"
		ftrsPos := strings.Index(workingInfo, " ftrs")
		if ftrsPos > fighterStart {
			fighterStr := workingInfo[fighterStart:ftrsPos]
			// Strip commas as Pascal does: StripChar(S, ',');
			fighterStr = strings.ReplaceAll(fighterStr, ",", "")
			fighterCount := p.parseIntSafe(fighterStr)

			// Validate fighter count (must be non-negative)
			if fighterCount >= 0 {
				trader.Fighters = fighterCount
			} else {
				trader.Fighters = 0
			}
		}
	}

	// Validate parsed trader data
	if trader.Name == "" {
		return // Skip traders with no name
	}

	// Store in currentTrader for continuation line processing
	p.currentTrader = trader

	// Don't add to list yet - wait for ship details in continuation line
}

// parseSectorPlanets handles detailed planet parsing from sector data (mirrors Pascal logic)
func (p *TWXParser) parseSectorPlanets(line string) {
	// Parse format: "Planets : Terra [Owned by Federation], Stardock"
	// Pascal mirrors TWX Process.pas planet parsing logic

	if !strings.HasPrefix(line, "Planets : ") {
		return
	}

	// Phase 4.5: Planets tracked via collection trackers
	p.sectorPosition = SectorPosPlanets

	planetInfo := line[10:] // Remove "Planets : "

	// Enhanced parsing with Pascal-compliant logic
	p.parsePlanetInfo(planetInfo)
}

// parsePlanetInfo parses planet information with Pascal-compliant logic
func (p *TWXParser) parsePlanetInfo(planetInfo string) {
	// Smarter planet parsing that handles fighter counts correctly
	// First, look for fighter information and extract it
	fighterPos := strings.Index(planetInfo, ", w/")
	var fighterInfo string
	var cleanPlanetInfo string

	if fighterPos >= 0 {
		// Extract fighter info and clean planet info
		ftrsPos := strings.Index(planetInfo, " ftrs")
		if ftrsPos > fighterPos {
			fighterInfo = planetInfo[fighterPos : ftrsPos+5] // Include " ftrs"
			cleanPlanetInfo = planetInfo[:fighterPos] + planetInfo[ftrsPos+5:]
		} else {
			cleanPlanetInfo = planetInfo
		}
	} else {
		cleanPlanetInfo = planetInfo
	}

	// Now split by commas to handle multiple planets
	planets := strings.Split(cleanPlanetInfo, ",")

	for i, planetStr := range planets {
		planetStr = strings.TrimSpace(planetStr)
		if planetStr == "" {
			continue
		}

		planet := PlanetInfo{}

		// Enhanced citadel and stardock detection (mirrors Pascal exact logic)
		planetLower := strings.ToLower(planetStr)

		// Pascal-compliant citadel detection
		// Checks for "citadel" keyword in various positions
		planet.Citadel = p.detectCitadel(planetStr, planetLower)

		// Pascal-compliant stardock detection
		// Stardock can be standalone or part of planet name
		planet.Stardock = p.detectStardock(planetStr, planetLower)

		// Enhanced owner parsing with bracket detection
		p.parsePlanetOwnerAndName(&planet, planetStr)

		// Parse fighter information if present and this is the first planet (fighters typically apply to first planet)
		if i == 0 && fighterInfo != "" {
			p.parsePlanetFightersFromString(&planet, fighterInfo)
		}

		// Validate parsed planet data
		p.validatePlanetData(&planet)

		// Phase 4.5: Planets tracked via collection trackers only
		if p.sectorCollections != nil {
			p.sectorCollections.AddPlanet(planet.Name, planet.Owner, planet.Fighters, planet.Citadel, planet.Stardock)
		}
	}
}

// detectCitadel implements Pascal-compliant citadel detection logic
func (p *TWXParser) detectCitadel(planetStr, planetLower string) bool {
	// Pascal logic: look for "citadel" keyword in planet description
	// Can be in planet name or as modifier
	if strings.Contains(planetLower, "citadel") {
		return true
	}

	// Check for citadel indicators like "Cit" or "CitaDel"
	if strings.Contains(planetLower, " cit ") ||
		strings.HasSuffix(planetLower, " cit") ||
		strings.HasPrefix(planetLower, "cit ") {
		return true
	}

	return false
}

// detectStardock implements Pascal-compliant stardock detection logic
func (p *TWXParser) detectStardock(planetStr, planetLower string) bool {
	// Pascal logic: Stardock can be standalone or part of planet name
	if strings.Contains(planetLower, "stardock") {
		return true
	}

	// Check for stardock abbreviations
	if strings.Contains(planetLower, "sd ") ||
		strings.HasSuffix(planetLower, " sd") ||
		strings.HasPrefix(planetLower, "sd ") {
		return true
	}

	return false
}

// parsePlanetOwnerAndName extracts planet name and owner with Pascal-compliant bracket parsing
func (p *TWXParser) parsePlanetOwnerAndName(planet *PlanetInfo, planetStr string) {
	// Enhanced bracket parsing (mirrors Pascal owner extraction)
	bracketStart := strings.Index(planetStr, "[")
	if bracketStart > 0 {
		// Extract planet name (before brackets)
		planet.Name = strings.TrimSpace(planetStr[:bracketStart])

		// Extract owner information (between [ and ])
		bracketEnd := strings.Index(planetStr, "]")
		if bracketEnd > bracketStart {
			ownerInfo := planetStr[bracketStart+1 : bracketEnd]

			// Pascal-compliant owner parsing
			if strings.HasPrefix(ownerInfo, "Owned by ") {
				planet.Owner = strings.TrimSpace(ownerInfo[9:]) // Remove "Owned by "
			} else if strings.HasPrefix(ownerInfo, "owned by ") {
				planet.Owner = strings.TrimSpace(ownerInfo[9:]) // Case insensitive
			} else {
				// Direct owner name (no "Owned by" prefix)
				planet.Owner = strings.TrimSpace(ownerInfo)
			}

		}
	} else {
		// No brackets - entire string is planet name
		// Remove special flags from name for cleaner parsing
		cleanName := planetStr

		// Remove trailing indicators that aren't part of the name
		if planet.Stardock && strings.HasSuffix(strings.ToLower(cleanName), "stardock") {
			// If the entire thing is "Stardock", keep it as name
			if !strings.EqualFold(cleanName, "stardock") {
				// Remove "Stardock" from end if it's appended to another name
				cleanName = strings.TrimSpace(strings.TrimSuffix(cleanName, "Stardock"))
				cleanName = strings.TrimSpace(strings.TrimSuffix(cleanName, "stardock"))
			}
		}

		planet.Name = strings.TrimSpace(cleanName)
	}

	// Fallback for empty names
	if planet.Name == "" {
		if planet.Stardock {
			planet.Name = "Stardock"
		} else if planet.Citadel {
			planet.Name = "Citadel"
		} else {
			planet.Name = "Unknown Planet"
		}
	}
}

// parsePlanetFighters extracts fighter information from planet data
func (p *TWXParser) parsePlanetFighters(planet *PlanetInfo, planetStr string) {
	// Look for fighter information in planet string
	// Format: "Planet [Owner], w/ 1,000 ftrs" or similar
	fightersPos := strings.Index(planetStr, ", w/")
	if fightersPos >= 0 {
		// Find end of fighter count
		ftrsPos := strings.Index(planetStr, " ftrs")
		if ftrsPos > fightersPos {
			fighterStr := planetStr[fightersPos+4 : ftrsPos] // After ", w/"
			// Strip commas as Pascal does
			fighterStr = strings.ReplaceAll(fighterStr, ",", "")

			fighterCount := p.parseIntSafe(fighterStr)
			if fighterCount >= 0 {
				planet.Fighters = fighterCount
			} else {
			}
		}
	}
}

// parsePlanetFightersFromString extracts fighter information from a pre-extracted fighter string
func (p *TWXParser) parsePlanetFightersFromString(planet *PlanetInfo, fighterInfo string) {
	// fighterInfo format: ", w/ 1,000 ftrs"
	if strings.HasPrefix(fighterInfo, ", w/") && strings.HasSuffix(fighterInfo, " ftrs") {
		// Extract the number part
		fighterStr := fighterInfo[4 : len(fighterInfo)-5] // Remove ", w/" and " ftrs"
		fighterStr = strings.TrimSpace(fighterStr)

		// Strip commas as Pascal does
		fighterStr = strings.ReplaceAll(fighterStr, ",", "")

		fighterCount := p.parseIntSafe(fighterStr)
		if fighterCount >= 0 {
			planet.Fighters = fighterCount
		} else {
		}
	}
}

// parseSectorMines handles detailed mine parsing from sector data
func (p *TWXParser) parseSectorMines(line string) {

	// Parse format: "Mines   : 100 Limpet Mines (belong to Kirk)"
	//           or: "Mines   : 50 Armid Mines, 25 Limpet Mines (belong to Spock)"

	if !strings.HasPrefix(line, "Mines   : ") {
		return
	}

	// Clear previous mines
	// Phase 4.5: Mines tracked via database directly (no intermediate collection)
	p.sectorPosition = SectorPosMines

	mineInfo := line[10:] // Remove "Mines   : "

	// Extract owner from parentheses
	owner := ""
	if parenStart := strings.Index(mineInfo, "("); parenStart >= 0 {
		parenEnd := strings.Index(mineInfo, ")")
		if parenEnd > parenStart {
			ownerInfo := mineInfo[parenStart+1 : parenEnd]
			if strings.HasPrefix(ownerInfo, "belong to ") {
				owner = ownerInfo[10:] // Remove "belong to "
			} else {
				owner = ownerInfo
			}
			// Remove owner info for parsing
			mineInfo = strings.TrimSpace(mineInfo[:parenStart])
		}
	}

	// Split by commas to handle multiple mine types
	mineTypes := strings.Split(mineInfo, ",")

	for _, mineStr := range mineTypes {
		mineStr = strings.TrimSpace(mineStr)
		if mineStr == "" {
			continue
		}

		mine := MineInfo{Owner: owner}

		// Parse quantity and type (e.g., "100 Limpet Mines")
		parts := strings.Fields(mineStr)
		if len(parts) >= 3 {
			mine.Quantity = p.parseIntSafeWithCommas(parts[0])
			mine.Type = parts[1] // "Armid" or "Limpet"
		}

		// Phase 4.5: Mines tracked directly to database (no intermediate collection)
	}
}

// parseSectorFighters handles detailed fighter parsing from sector data
func (p *TWXParser) parseSectorFighters(line string) {

	// Parse format: "Fighters: 2,500 (belong to Kirk) [Defensive]"

	if !strings.HasPrefix(line, "Fighters: ") {
		return
	}

	fighterInfo := line[10:] // Remove "Fighters: "

	// Extract quantity
	parts := strings.Fields(fighterInfo)
	if len(parts) == 0 {
		return
	}

	quantity := p.parseIntSafeWithCommas(parts[0])

	// Extract owner
	owner := ""
	if parenStart := strings.Index(fighterInfo, "("); parenStart >= 0 {
		parenEnd := strings.Index(fighterInfo, ")")
		if parenEnd > parenStart {
			ownerInfo := fighterInfo[parenStart+1 : parenEnd]
			if strings.HasPrefix(ownerInfo, "belong to ") {
				owner = ownerInfo[10:] // Remove "belong to "
			} else {
				owner = ownerInfo
			}
		}
	}

	// Extract fighter type from brackets
	fighterType := FighterDefensive // Default
	if bracketStart := strings.Index(fighterInfo, "["); bracketStart >= 0 {
		bracketEnd := strings.Index(fighterInfo, "]")
		if bracketEnd > bracketStart {
			typeStr := strings.ToLower(fighterInfo[bracketStart+1 : bracketEnd])
			switch typeStr {
			case "offensive":
				fighterType = FighterOffensive
			case "defensive":
				fighterType = FighterDefensive
			case "toll":
				fighterType = FighterToll
			}
		}
	}

	// Store fighter data
	fighterData := FighterData{
		SectorNum: p.currentSectorIndex,
		Quantity:  quantity,
		Owner:     owner,
		Type:      fighterType,
	}

	// Use fighterData (placeholder - would store to DB in full implementation)
	_ = fighterData

	// Add to message history
	p.addToHistory(MessageFighter, line, owner, 0)
}

// parseSectorNavHaz handles navigation hazard parsing
func (p *TWXParser) parseSectorNavHaz(line string) {
	defer p.recoverFromPanic("parseSectorNavHaz")

	// Parse format: "NavHaz  : 5% (10)"

	if !strings.HasPrefix(line, "NavHaz  : ") {
		return
	}

	// Safe substring extraction with bounds checking
	if len(line) <= 10 {
		return
	}
	navHazInfo := line[10:] // Remove "NavHaz  : "

	// Extract percentage and store in current sector
	percentPos := strings.Index(navHazInfo, "%")
	if percentPos > 0 {
		percentStr := strings.TrimSpace(navHazInfo[:percentPos])
		navHazPercent := p.parseIntSafe(percentStr)

		// Validate NavHaz percentage (must be 0-100)
		navHazPercent = p.validatePercentage(navHazPercent)

		// Store NavHaz percentage in current sector data
		// Phase 2: NavHaz tracked via SectorTracker
		if p.sectorTracker != nil {
			p.sectorTracker.SetNavHaz(navHazPercent)
		}
	}

	// Extract actual count from parentheses (for logging/validation)
	if parenStart := strings.Index(navHazInfo, "("); parenStart >= 0 {
		parenEnd := strings.Index(navHazInfo, ")")
		if parenEnd > parenStart {
			countStr := navHazInfo[parenStart+1 : parenEnd]
			actualCount := p.parseIntSafe(countStr)
			// Use actualCount (placeholder - would validate/log in full implementation)
			_ = actualCount
		}
	}
}

// Enhanced sector continuation handling (mirrors Pascal logic from lines 775-844)
func (p *TWXParser) handleSectorContinuation(line string) {
	// Pascal: if (Copy(Line, 1, 8) = '        ') then
	if !strings.HasPrefix(line, "        ") {
		return
	}

	// Enhanced continuation line handling based on current sector position
	switch p.sectorPosition {
	case SectorPosShips:
		p.handleShipContinuation(line)
	case SectorPosPorts:
		p.handlePortContinuation(line)
	case SectorPosTraders:
		p.handleTraderContinuation(line)
	case SectorPosPlanets:
		p.handlePlanetContinuation(line)
	case SectorPosMines:
		p.handleMineContinuation(line)
	default:
	}
}

// handleTraderContinuation handles trader continuation lines (mirrors Pascal lines 795-818)
func (p *TWXParser) handleTraderContinuation(line string) {

	// Pascal logic:
	// if (GetParameter(Line, 1) = 'in') then
	firstParam := p.getParameter(line, 1)
	if firstParam == "in" {
		// Ship info for current trader - parse ship details
		// Pascal logic:
		// I := GetParameterPos(Line, 2);
		// NewTrader^.ShipName := Copy(Line, I, Pos('(', Line) - I - 1);
		// I := Pos('(', Line);
		// NewTrader^.ShipType := Copy(Line, I + 1, Pos(')', Line) - I - 1);

		if p.currentTrader.Name == "" {
			return
		}

		// Find the position of the second parameter (ship name starts here)
		// The line format is: "        in ShipName (ShipType)"
		shipInfoStart := strings.Index(line, "in ") + 3 // After "in "
		if shipInfoStart < 3 {
			// Add trader without ship details
			p.finalizeCurrentTrader()
			return
		}

		shipInfo := strings.TrimSpace(line[shipInfoStart:])

		// Extract ship name (before the opening parenthesis)
		parenStart := strings.Index(shipInfo, "(")
		if parenStart > 0 {
			// Copy the currentTrader and add ship details
			trader := p.currentTrader
			trader.ShipName = strings.TrimSpace(shipInfo[:parenStart])

			// Extract ship type (between parentheses)
			parenEnd := strings.Index(shipInfo, ")")
			if parenEnd > parenStart {
				trader.ShipType = shipInfo[parenStart+1 : parenEnd]
			}

			// Validate completed trader data
			p.validateTraderData(&trader)

			// Phase 4.5: Traders tracked via collection trackers only
			if p.sectorCollections != nil {
				p.sectorCollections.AddTrader(trader.Name, trader.ShipName, trader.ShipType, trader.Fighters)
			}

			// Reset currentTrader to prevent duplicate addition in sectorCompleted
			p.currentTrader = TraderInfo{}
		} else {
			// No parentheses found, but still extract ship name
			trader := p.currentTrader
			trader.ShipName = strings.TrimSpace(shipInfo)

			// Look for alignment in ship info if no ship type parentheses
			if alignStart := strings.Index(shipInfo, "["); alignStart >= 0 {
				alignEnd := strings.Index(shipInfo[alignStart:], "]")
				if alignEnd > 0 {
					alignmentCandidate := strings.TrimSpace(shipInfo[alignStart+1 : alignStart+alignEnd])
					alignmentLower := strings.ToLower(alignmentCandidate)
					if alignmentLower == "good" || alignmentLower == "evil" || alignmentLower == "neutral" ||
						alignmentLower == "outlaw" || alignmentLower == "criminal" {
						trader.Alignment = alignmentCandidate
						// Remove alignment from ship name
						trader.ShipName = strings.TrimSpace(shipInfo[:alignStart])
					}
				}
			}

			// Validate completed trader data
			p.validateTraderData(&trader)

			// Phase 4.5: Traders tracked via collection trackers only
			if p.sectorCollections != nil {
				p.sectorCollections.AddTrader(trader.Name, trader.ShipName, trader.ShipType, trader.Fighters)
			}

			// Reset currentTrader to prevent duplicate addition in sectorCompleted
			p.currentTrader = TraderInfo{}
		}
	} else {
		// New trader on continuation line
		// Mirror same logic as parseSectorTraders but for continuation line

		// Finalize any pending trader first
		if p.currentTrader.Name != "" {
			p.finalizeCurrentTrader()
		}

		// Extract trader info from continuation line
		traderInfo := strings.TrimSpace(line[8:]) // Skip 8 spaces

		fighterPos := strings.Index(traderInfo, ", w/")
		if fighterPos == -1 {
			// No fighter info - just extract trader name
			trader := TraderInfo{
				Name:     strings.TrimSpace(traderInfo),
				Fighters: 0,
			}
			p.currentTrader = trader
			return
		}

		trader := TraderInfo{}

		// Extract trader name (from start to ', w/' position)
		trader.Name = strings.TrimSpace(traderInfo[:fighterPos])

		// Extract fighter count (from after ', w/' to ' ftrs')
		fighterStart := fighterPos + 4 // After ", w/"
		ftrsPos := strings.Index(traderInfo, " ftrs")
		if ftrsPos > fighterStart {
			fighterStr := traderInfo[fighterStart:ftrsPos]
			// Strip commas as Pascal does: StripChar(S, ',');
			fighterStr = strings.ReplaceAll(fighterStr, ",", "")
			fighterCount := p.parseIntSafe(fighterStr)

			// Validate fighter count (must be non-negative)
			if fighterCount >= 0 {
				trader.Fighters = fighterCount
			} else {
				trader.Fighters = 0
			}
		}

		// Validate trader name
		if trader.Name == "" {
			return
		}

		// Store as current trader for potential ship details
		p.currentTrader = trader
	}
}

// finalizeCurrentTrader adds the current trader to the list if it has valid data
func (p *TWXParser) finalizeCurrentTrader() {
	if p.currentTrader.Name != "" {
		p.validateTraderData(&p.currentTrader)
		// Phase 4.5: Traders tracked via collection trackers (no intermediate objects)
		p.currentTrader = TraderInfo{} // Reset
	}
}

// handlePlanetContinuation handles planet continuation lines (mirrors Pascal lines 819-822)
func (p *TWXParser) handlePlanetContinuation(line string) {
	// Pascal logic: NewPlanet^.Name := Copy(Line, 11, length(Line) - 10);
	// Enhanced to match Pascal exact behavior

	if len(line) <= 8 {
		return
	}

	// Pascal: Copy(Line, 11, length(Line) - 10)
	// In Pascal, indices are 1-based, so position 11 = Go index 10
	// Length - 10 means take all characters except first 10
	// But we also need to handle the 8 spaces at the start properly
	planetInfo := strings.TrimSpace(line[8:]) // Remove 8 spaces at start
	if planetInfo != "" {
		// Use the same enhanced parsing logic as main planet parsing
		p.parsePlanetInfo(planetInfo)
	}
}

// handleMineContinuation handles mine continuation lines
func (p *TWXParser) handleMineContinuation(line string) {
	// Pascal logic for second mine type (usually Limpet)

	// Extract owner from parentheses if present
	owner := ""
	if parenStart := strings.Index(line, "("); parenStart >= 0 {
		parenEnd := strings.Index(line, ")")
		if parenEnd > parenStart {
			ownerInfo := line[parenStart+1 : parenEnd]
			if strings.HasPrefix(ownerInfo, "belong to ") {
				owner = ownerInfo[10:] // Remove "belong to "
			}
		}
	}

	// Parse mine info similar to main mine parsing
	parts := strings.Fields(line)
	if len(parts) >= 3 {
		mine := MineInfo{Owner: owner}
		mine.Quantity = p.parseIntSafeWithCommas(parts[0])
		mine.Type = parts[1] // "Armid" or "Limpet"

		// Phase 4.5: Mines tracked directly to database (no intermediate collection)
	}
}

// handlePortContinuation handles port-specific continuation lines (mirrors Pascal lines 785-786)
func (p *TWXParser) handlePortContinuation(line string) {

	// Pascal: FCurrentSector.SPort.BuildTime := StrToIntSafe(GetParameter(Line, 4))
	// However, the actual format varies, so we need to be flexible in parsing

	buildTime := -1

	// Try Pascal GetParameter(Line, 4) first for exact compatibility
	param4 := p.getParameter(line, 4)
	if param4 != "" && isNumeric(param4) {
		pascalParam4 := p.parseIntSafe(param4)
		buildTime = pascalParam4
	} else {
		// Fall back to flexible parsing for common formats
		fields := strings.Fields(line)
		for i, field := range fields {
			// Look for numeric values that could be build time
			if isNumeric(field) {
				val := p.parseIntSafe(field)
				if val >= 0 && val <= 9999 { // Reasonable build time range
					// Check if this looks like a build time context
					if i > 0 && (strings.Contains(strings.ToLower(fields[i-1]), "build") ||
						strings.Contains(strings.ToLower(fields[i-1]), "time")) {
						buildTime = val
						break
					}
					// Also check if previous field contains ":"
					if i > 0 && strings.HasSuffix(fields[i-1], ":") {
						buildTime = val
						break
					}
				}
			}
		}
	}

	if buildTime >= 0 {
		// Store build time in current sector's port data
		// Phase 3: Port build time tracked via PortTracker
		if p.portTracker != nil {
			p.portTracker.SetBuildTime(buildTime)
		}
	} else {
	}
}

// parseEnhancedPortInfo handles enhanced port parsing with detailed class and trade pattern analysis
func (p *TWXParser) parseEnhancedPortInfo(line string) {
	// Parse port data (mirrors TWX Pascal logic)
	if strings.Contains(line, "<=-DANGER-=>") {
		// Port is destroyed
		return
	}

	if len(line) <= 10 {
		return
	}

	portInfo := line[10:] // Remove "Ports   : "

	// Extract port name (before ", Class")
	classPos := strings.Index(portInfo, ", Class")
	if classPos > 0 {
		portName := strings.TrimSpace(portInfo[:classPos])

		// Extract and parse class information
		classInfo := portInfo[classPos:]
		portClass := p.parsePortClass(classInfo)

		// Parse buy/sell indicators and trade pattern
		if strings.Contains(classInfo, "(") && strings.Contains(classInfo, ")") {
			start := strings.Index(classInfo, "(")
			end := strings.Index(classInfo, ")")
			if end > start {
				tradePattern := classInfo[start+1 : end]

				// Determine actual port class from trade pattern if not explicit
				if portClass == 0 {
					portClass = p.classFromTradePattern(tradePattern)
				}
			}
		}

		// Store port information
		p.storePortInfo(portName, portClass, portInfo)
	}
}

// storePortInfo stores parsed port information
func (p *TWXParser) storePortInfo(name string, class int, fullInfo string) {
	// This would typically store to a database or data structure
	// For now, we'll just log the structured information

	// In a full implementation, this would:
	// 1. Create/update port record in database
	// 2. Parse trade status from various indicators
	// 3. Store build time and other metadata
	// 4. Update sector data with port reference
}

// ============================================================================
// SECTOR PARSING METHODS (moved from twx_parser.go for clean separation)
// ============================================================================

func (p *TWXParser) handleSectorWarps(line string) {
	// Parse warp data from line like "Warps to Sector(s) :  (8247) - 18964"
	if len(line) > 20 {
		warpData := line[20:] // Remove "Warps to Sector(s) :"
		p.parseWarpConnections(warpData)
	}

	// Don't complete sector here - warps are not always the last item!
	// Sector display continues with other data like ports, traders, etc.
}

func (p *TWXParser) handleSectorBeacon(line string) {
	if len(line) > 10 {
		beacon := line[10:]
		// Phase 2: Beacon tracked via SectorTracker
		if p.sectorTracker != nil {
			p.sectorTracker.SetBeacon(beacon)
		}
	}
}

func (p *TWXParser) handleSectorPorts(line string) {
	p.sectorPosition = SectorPosPorts

	// Parse port data (mirrors TWX Pascal logic from lines 671-703)
	if strings.Contains(line, "<=-DANGER-=>") {
		// Port is destroyed - set Dead flag
		if p.portTracker != nil {
			p.portTracker.SetDead(true)
		}
		return
	}

	if len(line) <= 10 {
		return
	}

	portInfo := line[10:] // Remove "Ports   : "

	// Extract port name (before ", Class")
	classPos := strings.Index(portInfo, ", Class")
	if classPos <= 0 {
		return
	}

	portName := strings.TrimSpace(portInfo[:classPos])

	// Extract class number (Pascal: StrToIntSafe(Copy(Line, Pos(', Class', Line) + 8, 1)))
	classNum := 0
	if classPos+8 < len(portInfo) {
		classStr := string(portInfo[classPos+8])
		classNum = p.parseIntSafe(classStr)
	}

	// Parse buy/sell indicators from end of line (Pascal logic: lines 685-698)
	// Format: "Port Name, Class 1 Port BBS" (last 3 chars indicate buy/sell)
	buyOre := false
	buyOrg := false
	buyEquip := false

	if len(portInfo) >= 3 {
		// Get last 3 characters for trade pattern
		tradePattern := portInfo[len(portInfo)-3:]

		// Pascal logic: if (Line[length(Line) - 3] = 'B')
		if len(tradePattern) >= 3 {
			buyOre = (tradePattern[0] == 'B')
			buyOrg = (tradePattern[1] == 'B')
			buyEquip = (tradePattern[2] == 'B')
		}
	}

	// Determine port class from buy/sell pattern if not explicit (mirrors Pascal logic)
	if classNum == 0 {
		classNum = p.determinePortClassFromPattern(buyOre, buyOrg, buyEquip)
	}

	// Phase 3: Store port information using straight-sql tracker
	if p.portTracker != nil {
		p.portTracker.SetName(portName).SetClassIndex(classNum).SetBuildTime(0)
		p.portTracker.SetBuyProducts(buyOre, buyOrg, buyEquip)
		// debug.Info("PORT: Tracker updated", "name", portName, "class", classNum, "buy_ore", buyOre, "buy_org", buyOrg, "buy_equip", buyEquip)
	}
}

// determinePortClassFromPattern determines port class from buy/sell pattern (mirrors Pascal logic)
func (p *TWXParser) determinePortClassFromPattern(buyOre, buyOrg, buyEquip bool) int {
	// Mirror Pascal logic from ProcessPortLine (lines 1055-1062)
	// BBS = Class 1, BSB = Class 2, SBB = Class 3, etc.

	if buyOre && buyOrg && !buyEquip {
		return 1 // BBS
	} else if buyOre && !buyOrg && buyEquip {
		return 2 // BSB
	} else if !buyOre && buyOrg && buyEquip {
		return 3 // SBB
	} else if !buyOre && !buyOrg && buyEquip {
		return 4 // SSB
	} else if !buyOre && buyOrg && !buyEquip {
		return 5 // SBS
	} else if buyOre && !buyOrg && !buyEquip {
		return 6 // BSS
	} else if !buyOre && !buyOrg && !buyEquip {
		return 7 // SSS
	} else if buyOre && buyOrg && buyEquip {
		return 8 // BBB
	}

	return 0 // Unknown pattern
}

func (p *TWXParser) handleSectorPlanets(line string) {
	p.sectorPosition = SectorPosPlanets
	// Call detailed planet parsing
	p.parseSectorPlanets(line)
}

func (p *TWXParser) handleSectorTraders(line string) {
	p.sectorPosition = SectorPosTraders
	// Call detailed trader parsing
	p.parseSectorTraders(line)
}

func (p *TWXParser) handleSectorShips(line string) {
	p.sectorPosition = SectorPosShips
	// Call detailed ship parsing
	p.parseSectorShips(line)
}

func (p *TWXParser) handleSectorFighters(line string) {
	// Call detailed fighter parsing
	p.parseSectorFighters(line)
}

func (p *TWXParser) handleSectorNavHaz(line string) {
	p.sectorPosition = SectorPosNormal
	// Call detailed navhaz parsing
	p.parseSectorNavHaz(line)
}

func (p *TWXParser) handleSectorMines(line string) {
	p.sectorPosition = SectorPosMines
	// Call detailed mine parsing
	p.parseSectorMines(line)
}

func (p *TWXParser) handleSectorConstellation(parts []string) {
	// Extract constellation (everything after "in")
	if len(parts) >= 5 && parts[3] == "in" {
		constellation := strings.Join(parts[4:], " ")
		// Remove trailing period if present
		constellation = strings.TrimSuffix(constellation, ".")
		// Remove exploration status suffixes like "(unexplored)"
		constellation = strings.TrimSuffix(constellation, " (unexplored)")
		// Phase 2: Constellation tracked via SectorTracker
		if p.sectorTracker != nil {
			p.sectorTracker.SetConstellation(constellation)
		}
	}
}
