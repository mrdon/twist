package streaming

import (
	"strings"
	"twist/internal/debug"
)

// ============================================================================
// ERROR RECOVERY AND DATA VALIDATION (Mirror TWX Pascal Robustness)
// ============================================================================

// safeSubstring safely extracts a substring with bounds checking
func (p *TWXParser) safeSubstring(s string, start, end int) string {
	if start < 0 || start >= len(s) {
		debug.Log("TWXParser: SafeSubstring start %d out of bounds for string length %d", start, len(s))
		return ""
	}
	if end < start || end > len(s) {
		debug.Log("TWXParser: SafeSubstring end %d invalid for start %d, string length %d", end, start, len(s))
		return s[start:]
	}
	return s[start:end]
}

// safeStringIndex safely finds index with bounds validation
func (p *TWXParser) safeStringIndex(s, substr string) int {
	if s == "" || substr == "" {
		return -1
	}
	return strings.Index(s, substr)
}

// validateSectorNumber ensures sector numbers are within reasonable bounds
func (p *TWXParser) validateSectorNumber(sectorNum int) bool {
	// TWX typically handles sectors 1-20000, but allow flexibility
	if sectorNum < 1 || sectorNum > 50000 {
		debug.Log("TWXParser: Invalid sector number %d - outside reasonable range", sectorNum)
		return false
	}
	return true
}

// validatePercentage ensures percentage values are within 0-100% range
func (p *TWXParser) validatePercentage(pct int) int {
	if pct < 0 {
		debug.Log("TWXParser: Negative percentage %d corrected to 0", pct)
		return 0
	}
	if pct > 100 {
		debug.Log("TWXParser: Percentage %d exceeds 100%%, capped at 100", pct)
		return 100
	}
	return pct
}

// validateFighterCount ensures fighter counts are reasonable
func (p *TWXParser) validateFighterCount(fighters int) int {
	if fighters < 0 {
		debug.Log("TWXParser: Negative fighter count %d corrected to 0", fighters)
		return 0
	}
	// Cap at reasonable maximum (100 billion fighters)
	if fighters > 100000000000 {
		debug.Log("TWXParser: Fighter count %d exceeds reasonable maximum, capped", fighters)
		return 100000000000
	}
	return fighters
}

// validateCredits ensures credit amounts are reasonable
func (p *TWXParser) validateCredits(credits int) int {
	if credits < 0 {
		debug.Log("TWXParser: Negative credits %d corrected to 0", credits)
		return 0
	}
	// Cap at reasonable maximum (100 trillion credits)
	if credits > 100000000000000 {
		debug.Log("TWXParser: Credits %d exceeds reasonable maximum, capped", credits)
		return 100000000000000
	}
	return credits
}

// recoverFromPanic handles panic recovery with proper logging
func (p *TWXParser) recoverFromPanic(operation string) {
	if r := recover(); r != nil {
		debug.Log("TWXParser: PANIC RECOVERED in %s: %v", operation, r)
		// Reset parser state to prevent cascade failures
		p.resetParserState()
	}
}

// resetParserState resets parser to a safe state after errors
func (p *TWXParser) resetParserState() {
	debug.Log("TWXParser: Resetting parser state due to error recovery")
	
	// Reset current parsing context
	p.currentDisplay = DisplayNone
	p.sectorPosition = SectorPosNormal
	p.currentLine = ""
	p.currentANSILine = ""
	p.rawANSILine = ""
	p.inANSI = false
	
	// Clear temporary data structures
	p.currentShips = nil
	p.currentTraders = nil
	p.currentPlanets = nil
	p.currentMines = nil
	p.currentProducts = nil
	p.currentTrader = TraderInfo{}
	
	// Clear message context
	p.currentChannel = 0
	p.currentMessage = ""
	
	debug.Log("TWXParser: Parser state reset completed")
}

// validateLineFormat performs basic line format validation
func (p *TWXParser) validateLineFormat(line string) bool {
	// Check for reasonable line length
	if len(line) > 2000 {
		debug.Log("TWXParser: Line exceeds maximum length: %d characters", len(line))
		return false
	}
	
	// Check for null characters or other problematic content
	if strings.Contains(line, "\x00") {
		debug.Log("TWXParser: Line contains null characters")
		return false
	}
	
	return true
}

// safeParseWithRecovery wraps parsing operations with panic recovery
func (p *TWXParser) safeParseWithRecovery(operation string, parseFunc func()) {
	defer p.recoverFromPanic(operation)
	parseFunc()
}

// validatePortData ensures port data is within reasonable bounds
func (p *TWXParser) validatePortData(port *PortData) {
	if port == nil {
		return
	}
	
	// Validate class index (0-9 are typical)
	if port.ClassIndex < 0 || port.ClassIndex > 20 {
		debug.Log("TWXParser: Port class %d outside typical range, using as-is", port.ClassIndex)
	}
	
	// Validate build time (reasonable range)
	if port.BuildTime < 0 {
		debug.Log("TWXParser: Negative build time %d corrected to 0", port.BuildTime)
		port.BuildTime = 0
	}
	if port.BuildTime > 1000000 {
		debug.Log("TWXParser: Build time %d exceeds reasonable maximum, capped", port.BuildTime)
		port.BuildTime = 1000000
	}
	
	// Validate percentages
	port.OrePercent = p.validatePercentage(port.OrePercent)
	port.OrgPercent = p.validatePercentage(port.OrgPercent)
	port.EquipPercent = p.validatePercentage(port.EquipPercent)
	
	// Validate amounts (non-negative)
	if port.OreAmount < 0 {
		debug.Log("TWXParser: Negative ore amount %d corrected to 0", port.OreAmount)
		port.OreAmount = 0
	}
	if port.OrgAmount < 0 {
		debug.Log("TWXParser: Negative organics amount %d corrected to 0", port.OrgAmount)
		port.OrgAmount = 0
	}
	if port.EquipAmount < 0 {
		debug.Log("TWXParser: Negative equipment amount %d corrected to 0", port.EquipAmount)
		port.EquipAmount = 0
	}
}

// validateShipData ensures ship data is reasonable
func (p *TWXParser) validateShipData(ship *ShipInfo) {
	if ship == nil {
		return
	}
	
	// Validate fighter count
	ship.Fighters = p.validateFighterCount(ship.Fighters)
	
	// Validate name length
	if len(ship.Name) > 200 {
		debug.Log("TWXParser: Ship name truncated from %d to 200 characters", len(ship.Name))
		ship.Name = ship.Name[:200]
	}
	if len(ship.Owner) > 200 {
		debug.Log("TWXParser: Ship owner truncated from %d to 200 characters", len(ship.Owner))
		ship.Owner = ship.Owner[:200]
	}
}

// validateTraderData ensures trader data is reasonable
func (p *TWXParser) validateTraderData(trader *TraderInfo) {
	if trader == nil {
		return
	}
	
	// Validate fighter count
	trader.Fighters = p.validateFighterCount(trader.Fighters)
	
	// Validate name lengths
	if len(trader.Name) > 200 {
		debug.Log("TWXParser: Trader name truncated from %d to 200 characters", len(trader.Name))
		trader.Name = trader.Name[:200]
	}
	if len(trader.ShipName) > 200 {
		debug.Log("TWXParser: Trader ship name truncated from %d to 200 characters", len(trader.ShipName))
		trader.ShipName = trader.ShipName[:200]
	}
}

// validatePlanetData ensures planet data is reasonable
func (p *TWXParser) validatePlanetData(planet *PlanetInfo) {
	if planet == nil {
		return
	}
	
	// Validate fighter count
	planet.Fighters = p.validateFighterCount(planet.Fighters)
	
	// Validate name length
	if len(planet.Name) > 200 {
		debug.Log("TWXParser: Planet name truncated from %d to 200 characters", len(planet.Name))
		planet.Name = planet.Name[:200]
	}
	if len(planet.Owner) > 200 {
		debug.Log("TWXParser: Planet owner truncated from %d to 200 characters", len(planet.Owner))
		planet.Owner = planet.Owner[:200]
	}
	
	// Enhanced validation for planet parsing
	if planet.Name == "" {
		debug.Log("TWXParser: Warning - planet with empty name")
	}
	
	// Validate special type consistency
	if planet.Stardock && planet.Citadel {
		debug.Log("TWXParser: Notice - planet %s marked as both Stardock and Citadel", planet.Name)
	}
	
	// Owner validation and cleanup
	if planet.Owner != "" {
		// Clean up owner names
		planet.Owner = strings.TrimSpace(planet.Owner)
		if strings.HasSuffix(planet.Owner, ",") {
			planet.Owner = strings.TrimSuffix(planet.Owner, ",")
		}
	}
}

// validatePlayerStats ensures player statistics are within reasonable bounds
func (p *TWXParser) validatePlayerStats(stats *PlayerStats) {
	if stats == nil {
		return
	}
	
	// Validate counts and amounts
	stats.Turns = p.validateNonNegative(stats.Turns, "turns")
	stats.Credits = p.validateCredits(stats.Credits)
	stats.Fighters = p.validateFighterCount(stats.Fighters)
	stats.Shields = p.validateNonNegative(stats.Shields, "shields")
	
	// Validate holds
	stats.TotalHolds = p.validateNonNegative(stats.TotalHolds, "total holds")
	stats.OreHolds = p.validateNonNegative(stats.OreHolds, "ore holds")
	stats.OrgHolds = p.validateNonNegative(stats.OrgHolds, "organics holds")
	stats.EquHolds = p.validateNonNegative(stats.EquHolds, "equipment holds")
	stats.ColHolds = p.validateNonNegative(stats.ColHolds, "colonist holds")
	
	// Validate equipment
	stats.Photons = p.validateNonNegative(stats.Photons, "photons")
	stats.Armids = p.validateNonNegative(stats.Armids, "armids")
	stats.Limpets = p.validateNonNegative(stats.Limpets, "limpets")
	stats.GenTorps = p.validateNonNegative(stats.GenTorps, "genesis torpedoes")
	stats.Cloaks = p.validateNonNegative(stats.Cloaks, "cloaks")
	stats.Beacons = p.validateNonNegative(stats.Beacons, "beacons")
	stats.Atomics = p.validateNonNegative(stats.Atomics, "atomic detonators")
	stats.Corbomite = p.validateNonNegative(stats.Corbomite, "corbomite")
	stats.Eprobes = p.validateNonNegative(stats.Eprobes, "ether probes")
	stats.MineDisr = p.validateNonNegative(stats.MineDisr, "mine disruptors")
	
	// Validate experience and alignment (can be negative)
	stats.Experience = p.validateCredits(stats.Experience) // Use same validation as credits
}

// validateNonNegative ensures a value is non-negative
func (p *TWXParser) validateNonNegative(value int, fieldName string) int {
	if value < 0 {
		debug.Log("TWXParser: Negative %s %d corrected to 0", fieldName, value)
		return 0
	}
	return value
}

// safeDivide performs safe division with zero check
func (p *TWXParser) safeDivide(numerator, denominator int) int {
	if denominator == 0 {
		debug.Log("TWXParser: Division by zero avoided, returning 0")
		return 0
	}
	return numerator / denominator
}

// validateWarpData ensures warp data is reasonable
func (p *TWXParser) validateWarpData(warps []int) []int {
	if warps == nil {
		return nil
	}
	
	validWarps := make([]int, 0, len(warps))
	for _, warp := range warps {
		if p.validateSectorNumber(warp) {
			validWarps = append(validWarps, warp)
		}
	}
	
	if len(validWarps) != len(warps) {
		debug.Log("TWXParser: Filtered %d invalid warps, %d valid warps remain", 
			len(warps)-len(validWarps), len(validWarps))
	}
	
	return validWarps
}

// errorRecoveryHandler wraps critical parser operations with comprehensive error handling
func (p *TWXParser) errorRecoveryHandler(operation string, criticalFunc func() error) {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("TWXParser: CRITICAL ERROR in %s - PANIC: %v", operation, r)
			p.resetParserState()
		}
	}()
	
	if err := criticalFunc(); err != nil {
		debug.Log("TWXParser: Error in %s: %v", operation, err)
		// Don't reset state for non-critical errors, just log them
	}
}

// validateMessageData ensures message data is reasonable
func (p *TWXParser) validateMessageData(msg *MessageHistory) {
	if msg == nil {
		return
	}
	
	// Validate content length
	if len(msg.Content) > 10000 {
		debug.Log("TWXParser: Message content truncated from %d to 10000 characters", len(msg.Content))
		msg.Content = msg.Content[:10000]
	}
	
	// Validate sender name
	if len(msg.Sender) > 500 {
		debug.Log("TWXParser: Message sender truncated from %d to 500 characters", len(msg.Sender))
		msg.Sender = msg.Sender[:500]
	}
	
	// Validate channel number (should be within reasonable range)
	if msg.Channel < 0 || msg.Channel > 9999 {
		debug.Log("TWXParser: Channel number %d outside reasonable range", msg.Channel)
	}
}

// validateCollectedSectorData validates all data collected for the current sector
func (p *TWXParser) validateCollectedSectorData() {
	defer p.recoverFromPanic("validateCollectedSectorData")
	
	// Validate port data
	p.validatePortData(&p.currentSector.Port)
	
	// Validate all ships
	for i := range p.currentShips {
		p.validateShipData(&p.currentShips[i])
	}
	
	// Validate all traders
	for i := range p.currentTraders {
		p.validateTraderData(&p.currentTraders[i])
	}
	
	// Validate all planets
	for i := range p.currentPlanets {
		p.validatePlanetData(&p.currentPlanets[i])
	}
	
	// Validate NavHaz percentage
	p.currentSector.NavHaz = p.validatePercentage(p.currentSector.NavHaz)
	
	// Validate constellation and beacon names
	if len(p.currentSector.Constellation) > 500 {
		debug.Log("TWXParser: Constellation name truncated from %d to 500 characters", len(p.currentSector.Constellation))
		p.currentSector.Constellation = p.currentSector.Constellation[:500]
	}
	
	if len(p.currentSector.Beacon) > 500 {
		debug.Log("TWXParser: Beacon name truncated from %d to 500 characters", len(p.currentSector.Beacon))
		p.currentSector.Beacon = p.currentSector.Beacon[:500]
	}
	
	debug.Log("TWXParser: Sector data validation completed for sector %d", p.currentSectorIndex)
}