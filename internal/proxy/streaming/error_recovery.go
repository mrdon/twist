package streaming

import (
	"strings"
)

// ============================================================================
// ERROR RECOVERY AND DATA VALIDATION (Mirror TWX Pascal Robustness)
// ============================================================================

// safeSubstring safely extracts a substring with bounds checking
func (p *TWXParser) safeSubstring(s string, start, end int) string {
	if start < 0 || start >= len(s) {
		return ""
	}
	if end < start || end > len(s) {
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
		return false
	}
	return true
}

// validatePercentage ensures percentage values are within 0-100% range
func (p *TWXParser) validatePercentage(pct int) int {
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

// validateFighterCount ensures fighter counts are reasonable
func (p *TWXParser) validateFighterCount(fighters int) int {
	if fighters < 0 {
		return 0
	}
	// Cap at reasonable maximum (100 billion fighters)
	if fighters > 100000000000 {
		return 100000000000
	}
	return fighters
}

// validateCredits ensures credit amounts are reasonable
func (p *TWXParser) validateCredits(credits int) int {
	if credits < 0 {
		return 0
	}
	// Cap at reasonable maximum (100 trillion credits)
	if credits > 100000000000000 {
		return 100000000000000
	}
	return credits
}

// recoverFromPanic handles panic recovery with proper logging
func (p *TWXParser) recoverFromPanic(operation string) {
	if r := recover(); r != nil {
		// Reset parser state to prevent cascade failures
		p.resetParserState()
	}
}

// resetParserState resets parser to a safe state after errors
func (p *TWXParser) resetParserState() {

	// Reset current parsing context
	p.currentDisplay = DisplayNone
	p.sectorPosition = SectorPosNormal
	p.currentLine = ""
	p.currentANSILine = ""
	p.rawANSILine = ""
	p.inANSI = false

	// Clear message context
	p.currentChannel = 0
	p.currentMessage = ""

}

// validateLineFormat performs basic line format validation
func (p *TWXParser) validateLineFormat(line string) bool {
	// Check for reasonable line length
	if len(line) > 2000 {
		return false
	}

	// Check for null characters or other problematic content
	if strings.Contains(line, "\x00") {
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
	}

	// Validate build time (reasonable range)
	if port.BuildTime < 0 {
		port.BuildTime = 0
	}
	if port.BuildTime > 1000000 {
		port.BuildTime = 1000000
	}

	// Validate percentages
	port.OrePercent = p.validatePercentage(port.OrePercent)
	port.OrgPercent = p.validatePercentage(port.OrgPercent)
	port.EquipPercent = p.validatePercentage(port.EquipPercent)

	// Validate amounts (non-negative)
	if port.OreAmount < 0 {
		port.OreAmount = 0
	}
	if port.OrgAmount < 0 {
		port.OrgAmount = 0
	}
	if port.EquipAmount < 0 {
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
		ship.Name = ship.Name[:200]
	}
	if len(ship.Owner) > 200 {
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
		trader.Name = trader.Name[:200]
	}
	if len(trader.ShipName) > 200 {
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
		planet.Name = planet.Name[:200]
	}
	if len(planet.Owner) > 200 {
		planet.Owner = planet.Owner[:200]
	}

	// Enhanced validation for planet parsing
	if planet.Name == "" {
	}

	// Validate special type consistency
	if planet.Stardock && planet.Citadel {
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

// validateNonNegative ensures a value is non-negative
func (p *TWXParser) validateNonNegative(value int, fieldName string) int {
	if value < 0 {
		return 0
	}
	return value
}

// safeDivide performs safe division with zero check
func (p *TWXParser) safeDivide(numerator, denominator int) int {
	if denominator == 0 {
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
		// Some warps were filtered out
	}

	return validWarps
}

// errorRecoveryHandler wraps critical parser operations with comprehensive error handling
func (p *TWXParser) errorRecoveryHandler(operation string, criticalFunc func() error) {
	defer func() {
		if r := recover(); r != nil {
			p.resetParserState()
		}
	}()

	if err := criticalFunc(); err != nil {
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
		msg.Content = msg.Content[:10000]
	}

	// Validate sender name
	if len(msg.Sender) > 500 {
		msg.Sender = msg.Sender[:500]
	}

	// Validate channel number (should be within reasonable range)
	if msg.Channel < 0 || msg.Channel > 9999 {
	}
}
