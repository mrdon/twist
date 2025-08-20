package streaming

import (
	"strings"
	"twist/internal/debug"
	"twist/internal/proxy/database"
)

// ============================================================================
// PORT PARSING LOGIC (Mirrors TWX Pascal port processing in Process.pas)
// ============================================================================

// initializePortData initializes port data structure for the current sector
func (p *TWXParser) initializePortData(portName string) {
	
	// Store port name for database saving
	p.currentPortName = portName
	
	// Clear any existing port data
	p.ClearProductData()
}

// handlePortCommodity processes commodity lines when in port context
func (p *TWXParser) handlePortCommodity(line string) {
	// Only process commodity lines when we're in port display mode
	if p.currentDisplay != DisplayPortCR && p.currentDisplay != DisplayPort {
		return
	}
	
	
	// Check if this line has the '%' character at position 33 (TWX pattern)
	if len(line) > 33 && line[32] == '%' {
		p.parsePortCommodityLine(line)
	} else {
		// Fallback to general product line parsing
		if p.isProductLine(line) {
			p.parseProductLine(line)
		}
	}
}

// parsePortCommodityLine parses commodity lines using TWX Pascal patterns
func (p *TWXParser) parsePortCommodityLine(line string) {
	
	// TWX Pascal parsing logic:
	// Line := StringReplace(Line, '%', '', [rfReplaceAll]);
	// StatFuel := GetParameter(Line, 3);    // 'Buying' or 'Selling'  
	// QtyFuel := StrToIntSafe(GetParameter(Line, 4));    // Quantity
	// PercFuel := StrToIntSafe(GetParameter(Line, 5));   // Percentage
	
	// Remove '%' character and split into parameters
	cleanLine := strings.ReplaceAll(line, "%", "")
	parts := strings.Fields(cleanLine)
	
	if len(parts) < 5 {
		return
	}
	
	// Determine commodity type from the line
	productType := p.getProductTypeFromLine(line)
	if productType == -1 {
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
	
	if statusPos == -1 || statusPos+2 >= len(parts) {
		return
	}
	
	// Extract data based on status position
	status := parts[statusPos]           // "Selling" or "Buying"
	quantityStr := parts[statusPos+1]    // Quantity after status
	percentStr := parts[statusPos+2]     // Percentage after quantity
	
	// Parse values
	quantity := p.parseIntSafeWithCommas(quantityStr)
	percent := p.parseIntSafe(percentStr)
	isBuying := strings.EqualFold(status, "Buying")
	isSelling := strings.EqualFold(status, "Selling")
	
	// Create product info
	product := ProductInfo{
		Type:     ProductType(productType),
		Status:   status,
		Quantity: quantity,
		Percent:  percent,
		Buying:   isBuying,
		Selling:  isSelling,
	}
	
	// Store the product data
	p.currentProducts = append(p.currentProducts, product)
	
	// If we've collected all 3 standard commodities (Fuel Ore, Organics, Equipment), save port data
	if len(p.currentProducts) >= 3 {
		portClass := p.calculatePortClass()
		p.savePortData(portClass)
	}
}

// processLineInPortContext processes lines when in port context (state-dependent)
func (p *TWXParser) processLineInPortContext(line string) {
	// Only process port-specific lines when in port display mode
	if p.currentDisplay != DisplayPortCR && p.currentDisplay != DisplayPort {
		return
	}
	
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
		if strings.Contains(line, "? : ") {
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
					p.playerStats.Credits = credits
					
					// Save player stats to database and fire event
					p.errorRecoveryHandler("savePlayerStatsFromPort", func() error {
						return p.saveAllPlayerStats()
					})
					p.firePlayerStatsEvent(p.playerStats)
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
					// If this is the first time we see cargo holds, establish total capacity
					if p.playerStats.TotalHolds == 0 {
						// First "You have X credits and Y empty cargo holds" line - Y is total capacity
						p.playerStats.TotalHolds = emptyHolds
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
					p.playerStats.Experience += expGain
					
					// Save player stats to database and fire event
					p.errorRecoveryHandler("savePlayerStatsFromPort", func() error {
						return p.saveAllPlayerStats()
					})
					p.firePlayerStatsEvent(p.playerStats)
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
					p.playerStats.Turns = turns
					
					// Save player stats to database and fire event
					p.errorRecoveryHandler("savePlayerStatsFromPort", func() error {
						return p.saveAllPlayerStats()
					})
					p.firePlayerStatsEvent(p.playerStats)
				}
				break
			}
		}
	}
}

// exitPortContext exits port parsing context and saves port data
func (p *TWXParser) exitPortContext() {
	
	// Calculate port class from buy/sell patterns (TWX Pascal logic)
	portClass := p.calculatePortClass()
	
	// TODO: Save port data to database
	p.savePortData(portClass)
	
	// Reset display mode
	p.currentDisplay = DisplayNone
	
	// Clear product data
	p.ClearProductData()
}

// calculatePortClass determines port class from buy/sell patterns (mirrors TWX Pascal)
func (p *TWXParser) calculatePortClass() int {
	// TWX Pascal logic:
	// if (FCurrentSector.SPort.BuyProduct[ptFuelOre]) then PortClass := 'B' else PortClass := 'S';
	// if (FCurrentSector.SPort.BuyProduct[ptOrganics]) then PortClass := PortClass + 'B' else PortClass := PortClass + 'S';
	// if (FCurrentSector.SPort.BuyProduct[ptEquipment]) then PortClass := PortClass + 'B' else PortClass := PortClass + 'S';
	
	buyOre := false
	buyOrg := false
	buyEquip := false
	
	// Check products to determine buy/sell status
	for _, product := range p.currentProducts {
		switch product.Type {
		case ProductFuelOre:
			buyOre = product.Buying
		case ProductOrganics:
			buyOrg = product.Buying
		case ProductEquipment:
			buyEquip = product.Buying
		}
	}
	
	// Build pattern string
	pattern := ""
	if buyOre {
		pattern += "B"
	} else {
		pattern += "S"
	}
	if buyOrg {
		pattern += "B"
	} else {
		pattern += "S"
	}
	if buyEquip {
		pattern += "B"
	} else {
		pattern += "S"
	}
	
	// Map to class indices (TWX Pascal mapping)
	switch pattern {
	case "BBS":
		return 1 // Class 1
	case "BSB":
		return 2 // Class 2
	case "SBB":
		return 3 // Class 3
	case "SSB":
		return 4 // Class 4
	case "SBS":
		return 5 // Class 5
	case "BSS":
		return 6 // Class 6
	case "SSS":
		return 7 // Class 7
	case "BBB":
		return 8 // Class 8 (Special)
	default:
		return 0 // Unknown
	}
}

// savePortData saves port data to the database
func (p *TWXParser) savePortData(portClass int) {
	
	if p.database == nil || p.portSectorIndex <= 0 {
		return
	}
	
	// Create TPort structure from parsed data
	port := database.TPort{
		Name:       p.currentPortName,
		ClassIndex: portClass,
		Dead:       false,
		BuildTime:  0,
	}
	
	// Phase 3: Track discovered product data using straight-sql tracker
	var amountOre, amountOrg, amountEquip int
	var percentOre, percentOrg, percentEquip int
	var buyOre, buyOrg, buyEquip bool
	
	// Extract product data from parsed products and update tracker
	for _, product := range p.currentProducts {
		switch product.Type {
		case ProductFuelOre:
			port.BuyProduct[database.PtFuelOre] = product.Buying
			port.ProductAmount[database.PtFuelOre] = product.Quantity
			port.ProductPercent[database.PtFuelOre] = product.Percent
			// Track discovered data
			buyOre = product.Buying
			amountOre = product.Quantity
			percentOre = product.Percent
		case ProductOrganics:
			port.BuyProduct[database.PtOrganics] = product.Buying
			port.ProductAmount[database.PtOrganics] = product.Quantity
			port.ProductPercent[database.PtOrganics] = product.Percent
			// Track discovered data
			buyOrg = product.Buying
			amountOrg = product.Quantity
			percentOrg = product.Percent
		case ProductEquipment:
			port.BuyProduct[database.PtEquipment] = product.Buying
			port.ProductAmount[database.PtEquipment] = product.Quantity
			port.ProductPercent[database.PtEquipment] = product.Percent
			// Track discovered data
			buyEquip = product.Buying
			amountEquip = product.Quantity
			percentEquip = product.Percent
		}
	}
	
	// Update port tracker with discovered product data
	if p.portTracker != nil && len(p.currentProducts) > 0 {
		p.portTracker.SetName(p.currentPortName).SetClassIndex(portClass)
		p.portTracker.SetBuyProducts(buyOre, buyOrg, buyEquip)
		p.portTracker.SetProductAmounts(amountOre, amountOrg, amountEquip)
		p.portTracker.SetProductPercents(percentOre, percentOrg, percentEquip)
		debug.Log("PORT: Tracker updated with product data - %d products discovered", len(p.currentProducts))
	}
	
	// Save port to database
	if err := p.database.SavePort(port, p.portSectorIndex); err != nil {
		// Log error for debugging but continue
		debug.Log("Failed to save port data: %v", err)
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
				// Update cargo holds based on what commodity is being traded
				switch p.currentTradingCommodity {
				case ProductFuelOre:
					p.playerStats.OreHolds += quantity
				case ProductOrganics:
					p.playerStats.OrgHolds += quantity
				case ProductEquipment:
					p.playerStats.EquHolds += quantity
				}
			}
			break
		}
	}
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