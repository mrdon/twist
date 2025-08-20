package streaming

import (
	"strings"
)

// ============================================================================
// DETAILED PRODUCT LINE PARSING (Mirrors TWX Pascal product parsing logic)
// ============================================================================

// parseProductLine handles detailed product line parsing from port data
func (p *TWXParser) parseProductLine(line string) {
	
	// Parse formats:
	// "Fuel Ore     Selling       10,000 units at 100%"
	// "Organics     Buying         5,000 units at 95%"
	// "Equipment    : 1000 buying at 15"
	
	// Determine product type
	productType := p.getProductTypeFromLine(line)
	if productType == -1 {
		return // Not a product line
	}
	
	product := ProductInfo{
		Type: ProductType(productType),
	}
	
	// Parse different formats
	if strings.Contains(line, " units at ") {
		p.parseStandardProductFormat(line, &product)
	} else if strings.Contains(line, " at ") {
		p.parseAlternateProductFormat(line, &product)
	} else {
		return
	}
	
	// Phase 3: Product data is now tracked directly in PortTracker during parsing
}

// getProductTypeFromLine determines the product type from the line
func (p *TWXParser) getProductTypeFromLine(line string) int {
	lineLower := strings.ToLower(line)
	
	if strings.Contains(lineLower, "fuel ore") || strings.Contains(lineLower, "fuelore") {
		return int(ProductFuelOre)
	} else if strings.Contains(lineLower, "organics") {
		return int(ProductOrganics)
	} else if strings.Contains(lineLower, "equipment") {
		return int(ProductEquipment)
	}
	
	return -1 // Not a product line
}

// parseStandardProductFormat parses the standard format
// "Fuel Ore     Selling       10,000 units at 100%"
func (p *TWXParser) parseStandardProductFormat(line string, product *ProductInfo) {
	// Split by whitespace and reassemble
	parts := strings.Fields(line)
	if len(parts) < 6 {
		return
	}
	
	// Find "Buying" or "Selling"
	var statusIndex int = -1
	for i, part := range parts {
		if strings.EqualFold(part, "buying") {
			product.Buying = true
			product.Status = "Buying"
			statusIndex = i
			break
		} else if strings.EqualFold(part, "selling") {
			product.Selling = true
			product.Status = "Selling"
			statusIndex = i
			break
		}
	}
	
	if statusIndex == -1 {
		return
	}
	
	// Get quantity (should be after status)
	if statusIndex+1 < len(parts) {
		quantityStr := parts[statusIndex+1]
		product.Quantity = p.parseIntSafeWithCommas(quantityStr)
	}
	
	// Get percentage (last part should be like "100%")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		if strings.HasSuffix(lastPart, "%") {
			percentStr := lastPart[:len(lastPart)-1]
			product.Percent = p.parseIntSafe(percentStr)
		}
	}
}

// parseAlternateProductFormat parses the alternate format
// "Equipment    : 1000 buying at 15"
func (p *TWXParser) parseAlternateProductFormat(line string, product *ProductInfo) {
	// Find the colon
	colonPos := strings.Index(line, ":")
	if colonPos == -1 {
		return
	}
	
	// Parse after colon
	afterColon := strings.TrimSpace(line[colonPos+1:])
	parts := strings.Fields(afterColon)
	
	if len(parts) < 4 {
		return
	}
	
	// Format: "1000 buying at 15"
	product.Quantity = p.parseIntSafeWithCommas(parts[0])
	
	if strings.EqualFold(parts[1], "buying") {
		product.Buying = true
		product.Status = "Buying"
	} else if strings.EqualFold(parts[1], "selling") {
		product.Selling = true
		product.Status = "Selling"
	}
	
	// Get percentage (after "at")
	if len(parts) >= 4 && strings.EqualFold(parts[2], "at") {
		product.Percent = p.parseIntSafe(parts[3])
	}
}

// processPortLine handles port commerce lines using state-dependent parsing
func (p *TWXParser) processPortLine(line string) {
	// Use the comprehensive state-dependent port parsing from port_parser.go
	p.processLineInPortContext(line)
}

// isProductLine determines if a line contains product information
func (p *TWXParser) isProductLine(line string) bool {
	lineLower := strings.ToLower(line)
	
	// Check for product names
	hasProduct := strings.Contains(lineLower, "fuel ore") ||
		strings.Contains(lineLower, "organics") ||
		strings.Contains(lineLower, "equipment")
	
	if !hasProduct {
		return false
	}
	
	// Check for buying/selling indicators
	hasTrade := strings.Contains(lineLower, "buying") ||
		strings.Contains(lineLower, "selling") ||
		strings.Contains(lineLower, " at ") ||
		strings.Contains(lineLower, "units")
	
	return hasTrade
}

// extractProductInfo extracts basic product information (fallback method)
func (p *TWXParser) extractProductInfo(line string, productType ProductType) {
	product := ProductInfo{
		Type: productType,
	}
	
	// Simple extraction for quantities and percentages
	parts := strings.Fields(line)
	for _, part := range parts {
		// Look for quantities (numbers with commas)
		if strings.Contains(part, ",") || p.isNumeric(part) {
			if num := p.parseIntSafeWithCommas(part); num > 0 {
				product.Quantity = num
			}
		}
		
		// Look for percentages
		if strings.HasSuffix(part, "%") {
			percentStr := part[:len(part)-1]
			if percent := p.parseIntSafe(percentStr); percent > 0 {
				product.Percent = percent
			}
		}
		
		// Look for buying/selling
		if strings.EqualFold(part, "buying") {
			product.Buying = true
			product.Status = "Buying"
		} else if strings.EqualFold(part, "selling") {
			product.Selling = true
			product.Status = "Selling"
		}
	}
	
	// Store if we got useful information
	if product.Quantity > 0 || product.Percent > 0 {
		// Phase 3: Product data is now tracked directly in PortTracker during parsing
	}
}

// isNumeric checks if a string represents a number
func (p *TWXParser) isNumeric(s string) bool {
	if s == "" {
		return false
	}
	
	for _, char := range s {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// Enhanced port data parsing with buy/sell status determination
func (p *TWXParser) parsePortTradeStatus(line string) (buyOre, buyOrg, buyEquip bool) {
	// This mirrors the TWX Pascal logic for determining port buy/sell status
	// from various indicators in the port data
	
	lineLower := strings.ToLower(line)
	
	// Method 1: Direct "Buying" indicator
	if strings.Contains(lineLower, "buying") {
		if strings.Contains(lineLower, "fuel ore") {
			buyOre = true
		}
		if strings.Contains(lineLower, "organics") {
			buyOrg = true
		}
		if strings.Contains(lineLower, "equipment") {
			buyEquip = true
		}
	}
	
	// Method 2: Low percentage indicator (typically buying ports)
	parts := strings.Fields(line)
	for i, part := range parts {
		if strings.HasSuffix(part, "%") {
			percentStr := part[:len(part)-1]
			if percent := p.parseIntSafe(percentStr); percent > 0 && percent < 50 {
				// Low percentage often indicates buying
				if i > 0 {
					prevParts := strings.Join(parts[:i], " ")
					if strings.Contains(strings.ToLower(prevParts), "fuel ore") {
						buyOre = true
					}
					if strings.Contains(strings.ToLower(prevParts), "organics") {
						buyOrg = true
					}
					if strings.Contains(strings.ToLower(prevParts), "equipment") {
						buyEquip = true
					}
				}
			}
		}
	}
	
	return buyOre, buyOrg, buyEquip
}

