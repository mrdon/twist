package streaming

import (
	"strings"
	"twist/internal/api"
	"twist/internal/debug"
)

// ============================================================================
// PORT PARSING LOGIC (Mirrors TWX Pascal port processing in Process.pas)
// ============================================================================

// initializePortData initializes port data structure for the current sector
func (p *TWXParser) initializePortData(portName string) {
	
	// Store port name for database saving
	p.currentPortName = portName
	
	// Clear any existing port data
	// Phase 3: Product data clearing no longer needed with trackers
}

// handlePortCommodity processes commodity lines when in port context
func (p *TWXParser) handlePortCommodity(line string) {
	// Only process commodity lines when we're in port display mode
	if p.currentDisplay != DisplayPortCR && p.currentDisplay != DisplayPort {
		return
	}
	
	debug.Log("PORT: Processing commodity line: '%s' (len=%d)", line, len(line))
	
	// Find the position of '%' character
	percentPos := strings.Index(line, "%")
	debug.Log("PORT: percent found at position %d", percentPos)
	
	// Check if this line has the '%' character at position 33 (TWX pattern)
	if len(line) > 33 && line[32] == '%' {
		debug.Log("PORT: Using TWX pattern parsing (%% at pos 33)")
		p.parsePortCommodityLine(line)
	} else if percentPos >= 0 {
		debug.Log("PORT: Using commodity line parsing with %% at position %d", percentPos)
		p.parsePortCommodityLine(line)
	} else {
		// Fallback to general product line parsing
		if p.isProductLine(line) {
			debug.Log("PORT: Using general product line parsing")
			p.parseProductLine(line)
		} else {
			debug.Log("PORT: Line not recognized as product line")
		}
	}
}

// parsePortCommodityLine parses commodity lines using TWX Pascal patterns
func (p *TWXParser) parsePortCommodityLine(line string) {
	debug.Log("PORT: parsePortCommodityLine called with: '%s'", line)
	
	// TWX Pascal parsing logic:
	// Line := StringReplace(Line, '%', '', [rfReplaceAll]);
	// StatFuel := GetParameter(Line, 3);    // 'Buying' or 'Selling'  
	// QtyFuel := StrToIntSafe(GetParameter(Line, 4));    // Quantity
	// PercFuel := StrToIntSafe(GetParameter(Line, 5));   // Percentage
	
	// Remove '%' character and split into parameters
	cleanLine := strings.ReplaceAll(line, "%", "")
	parts := strings.Fields(cleanLine)
	debug.Log("PORT: cleanLine='%s', parts=%v (len=%d)", cleanLine, parts, len(parts))
	
	if len(parts) < 5 {
		debug.Log("PORT: Not enough parts (%d), skipping", len(parts))
		return
	}
	
	// Determine commodity type from the line
	productType := p.getProductTypeFromLine(line)
	debug.Log("PORT: productType=%d for line: '%s'", productType, line)
	if productType == -1 {
		debug.Log("PORT: Unknown product type, skipping")
		return
	}
	
	// Find the status word ("Selling" or "Buying") position to handle multi-word commodity names
	statusPos := -1
	for i, part := range parts {
		if strings.EqualFold(part, "Selling") || strings.EqualFold(part, "Buying") {
			statusPos = i
			break
		}
	}
	debug.Log("PORT: statusPos=%d", statusPos)
	
	if statusPos == -1 || statusPos+2 >= len(parts) {
		debug.Log("PORT: Invalid status position (%d) or not enough parts after status", statusPos)
		return
	}
	
	// Extract data based on status position
	status := parts[statusPos]           // "Selling" or "Buying"
	quantityStr := parts[statusPos+1]    // Quantity after status
	percentStr := parts[statusPos+2]     // Percentage after quantity
	
	debug.Log("PORT: status='%s', quantityStr='%s', percentStr='%s'", status, quantityStr, percentStr)
	
	// Parse values
	quantity := p.parseIntSafeWithCommas(quantityStr)
	percent := p.parseIntSafe(percentStr)
	isBuying := strings.EqualFold(status, "Buying")
	
	debug.Log("PORT: parsed quantity=%d, percent=%d, isBuying=%v", quantity, percent, isBuying)
	
	// Phase 3: Track product data directly in PortTracker
	if p.portTracker != nil {
		debug.Log("PORT: portTracker exists, updating product data")
		switch ProductType(productType) {
		case ProductFuelOre:
			debug.Log("PORT: Discovered fuel ore data - quantity: %d, percent: %d, buying: %v", quantity, percent, isBuying)
			// Set fuel ore data directly using individual field updates
			p.portTracker.SetFuelOreAmount(quantity).SetFuelOrePercent(percent).SetFuelOreBuying(isBuying)
		case ProductOrganics:
			debug.Log("PORT: Discovered organics data - quantity: %d, percent: %d, buying: %v", quantity, percent, isBuying)
			// Set organics data directly using individual field updates
			p.portTracker.SetOrganicsAmount(quantity).SetOrganicsPercent(percent).SetOrganicsBuying(isBuying)
		case ProductEquipment:
			debug.Log("PORT: Discovered equipment data - quantity: %d, percent: %d, buying: %v", quantity, percent, isBuying)
			// Set equipment data directly using individual field updates
			p.portTracker.SetEquipmentAmount(quantity).SetEquipmentPercent(percent).SetEquipmentBuying(isBuying)
		}
	} else {
		debug.Log("PORT: portTracker is nil, cannot update product data")
	}
}

// processLineInPortContext processes lines when in port context (state-dependent)
func (p *TWXParser) processLineInPortContext(line string) {
	// Only process port-specific lines when in port display mode
	if p.currentDisplay != DisplayPortCR && p.currentDisplay != DisplayPort {
		debug.Log("PORT: processLineInPortContext skipping line (display=%d): '%s'", p.currentDisplay, line)
		return
	}
	
	debug.Log("PORT: processLineInPortContext processing line (display=%d): '%s'", p.currentDisplay, line)
	
	// Check for commodity selection lines ("How many holds of X do you want to buy")
	if strings.Contains(line, "How many holds of") && strings.Contains(line, "do you want to buy") {
		p.parseCurrentCommodityContext(line)
	}
	
	// Check for commodity lines with different patterns
	lineLower := strings.ToLower(line)
	
	// Pattern 1: Standard commodity lines with '%' character
	if strings.Contains(line, "%") {
		if strings.Contains(lineLower, "fuel ore") ||
		   strings.Contains(lineLower, "organics") ||
		   strings.Contains(lineLower, "equipment") {
			p.handlePortCommodity(line)
		}
	}
	
	// Pattern 2: Trading transaction confirmations ("Agreed, X units.")
	if strings.Contains(line, "Agreed,") && strings.Contains(line, "units.") {
		p.parseTradeTransaction(line)
	}
	
	// Pattern 3: Player status updates during trading
	if strings.Contains(line, "You have ") && strings.Contains(line, "credits") {
		p.parsePlayerStatsFromPortLine(line)
	}
	
	// Pattern 4: Experience and promotion notifications
	if strings.Contains(line, "experience point") || strings.Contains(line, "promoted to") {
		p.parseExperienceFromPortLine(line)
	}
	
	// Pattern 5: Turn deduction
	if strings.Contains(line, "turn deducted") || strings.Contains(line, "turns left") {
		p.parseTurnsFromPortLine(line)
	}
	
	// Pattern 5: Command prompt - exit port context
	if strings.Contains(line, "Command [") {
		debug.Log("PORT: Found Command prompt line: '%s'", line)
		if strings.Contains(line, "? : ") {
			debug.Log("PORT: Exiting port context")
			p.exitPortContext()
		}
	}
}

// parsePlayerStatsFromPortLine extracts player stats from port trading lines
func (p *TWXParser) parsePlayerStatsFromPortLine(line string) {
	
	// Example: "You have 374,916 credits and 15 empty cargo holds."
	// Extract credits
	if strings.Contains(line, "credits") {
		parts := strings.Fields(line)
		for i, part := range parts {
			if strings.EqualFold(part, "have") && i+1 < len(parts) {
				creditsStr := strings.ReplaceAll(parts[i+1], ",", "")
				if credits := p.parseIntSafe(creditsStr); credits > 0 {
					// Update credits using straight-sql tracker
					if p.playerStatsTracker == nil {
						p.playerStatsTracker = NewPlayerStatsTracker()
					}
					p.playerStatsTracker.SetCredits(credits)
					
					// Save and fire event with fresh database read
					p.errorRecoveryHandler("savePlayerStatsFromPort", func() error {
						err := p.playerStatsTracker.Execute(p.database.GetDB())
						if err == nil && p.tuiAPI != nil {
							if fullPlayerStats, dbErr := p.database.GetPlayerStatsInfo(); dbErr == nil {
								p.firePlayerStatsEventDirect(fullPlayerStats)
							}
						}
						return err
					})
					
					// Also execute port tracker if it has updates, since port trading might be ending
					if p.portTracker != nil && p.portTracker.HasUpdates() {
						updates := p.portTracker.GetUpdates()
						debug.Log("PORT: Executing port tracker after player stats update (has %d updates)", len(updates))
						debug.Log("PORT: Port tracker updates: %+v", updates)
						p.errorRecoveryHandler("executePortTrackerAfterStats", func() error {
							err := p.portTracker.Execute(p.database.GetDB())
							if err != nil {
								debug.Log("PORT: Failed to execute port tracker: %v", err)
							} else {
								debug.Log("PORT: Successfully executed port tracker after player stats")
							}
							return err
						})
					}
				}
				break
			}
		}
	}
	
	// Extract cargo holds - track actual values, don't guess
	if strings.Contains(line, "cargo holds") {
		parts := strings.Fields(line)
		for i, part := range parts {
			if strings.EqualFold(part, "and") && i+1 < len(parts) {
				holdsStr := parts[i+1]
				if emptyHolds := p.parseIntSafe(holdsStr); emptyHolds >= 0 {
					// Update player stats using straight-sql tracker  
					if p.playerStatsTracker == nil {
						p.playerStatsTracker = NewPlayerStatsTracker()
					}
					
					// Calculate total holds: empty holds + cargo holds
					// First get current cargo to determine total capacity
					if currentStats, err := p.database.GetPlayerStatsInfo(); err == nil {
						totalCargo := currentStats.OreHolds + currentStats.OrgHolds + currentStats.EquHolds
						totalHolds := emptyHolds + totalCargo
						debug.Log("PORT: Calculated holds - empty: %d, cargo: %d, total: %d", emptyHolds, totalCargo, totalHolds)
						p.playerStatsTracker.SetTotalHolds(totalHolds)
					} else {
						// Fallback: if we can't read current stats, assume empty holds = total holds (first time)
						debug.Log("PORT: Fallback holds calculation - assuming empty holds (%d) equals total holds", emptyHolds)
						p.playerStatsTracker.SetTotalHolds(emptyHolds)
					}
				}
				break
			}
		}
	}
}

// parseExperienceFromPortLine extracts experience changes from port activity
func (p *TWXParser) parseExperienceFromPortLine(line string) {
	
	// Example: "For your great trading you receive 2 experience point(s)."
	if strings.Contains(line, "experience point") {
		parts := strings.Fields(line)
		for i, part := range parts {
			if strings.EqualFold(part, "receive") && i+1 < len(parts) {
				expStr := parts[i+1]
				if expGain := p.parseIntSafe(expStr); expGain > 0 {
					// Update experience using straight-sql tracker
					if p.playerStatsTracker == nil {
						p.playerStatsTracker = NewPlayerStatsTracker()
					}
					// Note: Experience should be set to new total, not incremented
					// We need to read current value first, then set new total
					if currentStats, err := p.database.GetPlayerStatsInfo(); err == nil {
						p.playerStatsTracker.SetExperience(currentStats.Experience + expGain)
					}
					
					// Save player stats to database and fire event
					// Execute tracker and fire fresh database event
					p.errorRecoveryHandler("savePlayerStatsFromPort", func() error {
						err := p.playerStatsTracker.Execute(p.database.GetDB())
						if err == nil && p.tuiAPI != nil {
							if fullPlayerStats, dbErr := p.database.GetPlayerStatsInfo(); dbErr == nil {
								p.firePlayerStatsEventDirect(fullPlayerStats)
							}
						}
						return err
					})
				}
				break
			}
		}
	}
}

// parseTurnsFromPortLine extracts turn changes from port activity
func (p *TWXParser) parseTurnsFromPortLine(line string) {
	
	// Example: "One turn deducted, 19993 turns left."
	if strings.Contains(line, "turns left") {
		parts := strings.Fields(line)
		// Find the number right before "turns"
		for i, part := range parts {
			if strings.EqualFold(part, "turns") && i > 0 {
				turnsStr := strings.ReplaceAll(parts[i-1], ",", "")
				if turns := p.parseIntSafe(turnsStr); turns > 0 {
					// Update turns using straight-sql tracker
					if p.playerStatsTracker == nil {
						p.playerStatsTracker = NewPlayerStatsTracker()
					}
					p.playerStatsTracker.SetTurns(turns)
					
					// Save player stats to database and fire event
					// Execute tracker and fire fresh database event
					p.errorRecoveryHandler("savePlayerStatsFromPort", func() error {
						err := p.playerStatsTracker.Execute(p.database.GetDB())
						if err == nil && p.tuiAPI != nil {
							if fullPlayerStats, dbErr := p.database.GetPlayerStatsInfo(); dbErr == nil {
								p.firePlayerStatsEventDirect(fullPlayerStats)
							}
						}
						return err
					})
				}
				break
			}
		}
	}
}

// exitPortContext exits port parsing context and saves port data
func (p *TWXParser) exitPortContext() {
	
	// Phase 3: Port data including class is tracked in PortTracker during parsing
	p.savePortData()
	
	// Reset display mode
	p.currentDisplay = DisplayNone
	
	// Clear product data
	// Phase 3: Product data clearing no longer needed with trackers
}


// savePortData saves port data to the database
func (p *TWXParser) savePortData() {
	
	if p.database == nil || p.portSectorIndex <= 0 {
		return
	}
	
	// Phase 3: Port data is tracked using PortTracker (no intermediate objects needed)
	if p.portTracker != nil {
		p.portTracker.SetName(p.currentPortName)
		debug.Log("PORT: Tracker updated with port name")
		
		// Execute the port tracker to save data to database
		if p.portTracker.HasUpdates() {
			err := p.portTracker.Execute(p.database.GetDB())
			if err != nil {
				debug.Log("PORT: Failed to execute port tracker: %v", err)
			} else {
				debug.Log("PORT: Successfully executed port tracker")
				
				// Fire OnPortUpdated API event with fresh database read
				if p.tuiAPI != nil {
					if portInfo, portErr := p.database.GetPortInfo(p.portSectorIndex); portErr == nil && portInfo != nil {
						debug.Log("PORT: Firing OnPortUpdated for sector %d: %s (Class %d)", p.portSectorIndex, portInfo.Name, portInfo.Class)
						p.tuiAPI.OnPortUpdated(*portInfo)
					} else {
						debug.Log("PORT: Failed to read fresh port info for API event: %v", portErr)
					}
				}
			}
		}
	}
}

// parseCurrentCommodityContext extracts which commodity is currently being traded
func (p *TWXParser) parseCurrentCommodityContext(line string) {
	
	// Example: "How many holds of Fuel Ore do you want to buy [20]?"
	// Iterate through all known product types to find a match
	allProductTypes := []ProductType{ProductFuelOre, ProductOrganics, ProductEquipment}
	
	for _, productType := range allProductTypes {
		productName := p.getProductTypeName(productType)
		if strings.Contains(line, productName) {
			p.currentTradingCommodity = productType
			return
		}
	}
	
}

// parseTradeTransaction processes "Agreed, X units." lines to track purchases
func (p *TWXParser) parseTradeTransaction(line string) {
	
	// Example: "Agreed, 2 units."
	parts := strings.Fields(line)
	
	// Look for pattern: ["Agreed,", "X", "units."]
	for i, part := range parts {
		if strings.EqualFold(part, "units.") && i > 0 {
			// Found "units." - the quantity should be the previous part
			quantityStr := parts[i-1]
			if quantity := p.parseIntSafe(quantityStr); quantity > 0 {
				// Update cargo holds using straight-sql tracker
				if p.playerStatsTracker == nil {
					p.playerStatsTracker = NewPlayerStatsTracker()
				}
				
				// Read current values and increment them
				if currentStats, err := p.database.GetPlayerStatsInfo(); err == nil {
					switch p.currentTradingCommodity {
					case ProductFuelOre:
						p.playerStatsTracker.SetOreHolds(currentStats.OreHolds + quantity)
					case ProductOrganics:
						p.playerStatsTracker.SetOrgHolds(currentStats.OrgHolds + quantity)
					case ProductEquipment:
						p.playerStatsTracker.SetEquHolds(currentStats.EquHolds + quantity)
					}
				}
			}
			break
		}
	}
}

// getPortDataFromTracker gets current port data values to preserve other products when updating one product
func (p *TWXParser) getPortDataFromTracker() ([3]int, [3]int, [3]bool) {
	// Arrays for [fuelore, organics, equipment]
	amounts := [3]int{0, 0, 0}
	percents := [3]int{0, 0, 0}  
	buys := [3]bool{false, false, false}
	
	debug.Log("PORT: getPortDataFromTracker called for sector %d", p.portSectorIndex)
	
	// Try to get existing data from database for this sector
	if p.database != nil && p.portSectorIndex > 0 {
		if portInfo, err := p.database.GetPortInfo(p.portSectorIndex); err == nil && portInfo != nil {
			debug.Log("PORT: Found existing port data with %d products", len(portInfo.Products))
			// Port exists - use current values  
			if len(portInfo.Products) >= 3 {
				amounts[0] = portInfo.Products[0].Quantity
				amounts[1] = portInfo.Products[1].Quantity  
				amounts[2] = portInfo.Products[2].Quantity
				percents[0] = portInfo.Products[0].Percentage
				percents[1] = portInfo.Products[1].Percentage
				percents[2] = portInfo.Products[2].Percentage
				buys[0] = portInfo.Products[0].Status == api.ProductStatusBuying
				buys[1] = portInfo.Products[1].Status == api.ProductStatusBuying
				buys[2] = portInfo.Products[2].Status == api.ProductStatusBuying
				debug.Log("PORT: Current amounts: %v, percents: %v, buys: %v", amounts, percents, buys)
			} else {
				debug.Log("PORT: Port has insufficient products (%d), using defaults", len(portInfo.Products))
			}
		} else {
			debug.Log("PORT: Failed to get port info: %v", err)
		}
	} else {
		debug.Log("PORT: No database or invalid sector index")
	}
	
	return amounts, percents, buys
}

// getProductTypeName returns a string name for a product type
func (p *TWXParser) getProductTypeName(productType ProductType) string {
	switch productType {
	case ProductFuelOre:
		return "Fuel Ore"
	case ProductOrganics:
		return "Organics"
	case ProductEquipment:
		return "Equipment"
	default:
		return "Unknown"
	}
}