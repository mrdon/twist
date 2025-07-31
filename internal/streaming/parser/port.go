package parser

import (
	"strings"
	"twist/internal/database"
)

// PortProcessor handles port-specific parsing logic
type PortProcessor struct {
	ctx   *ParserContext
	utils *ParseUtils
}

// NewPortProcessor creates a new port processor
func NewPortProcessor(ctx *ParserContext) *PortProcessor {
	return &PortProcessor{
		ctx:   ctx,
		utils: NewParseUtils(ctx),
	}
}

// ProcessPortLine processes port-related lines
func (pp *PortProcessor) ProcessPortLine(line string) {
	cleanLine := pp.utils.StripANSI(line)
	
	if strings.Contains(cleanLine, "Trading") {
		pp.processPortHeader(cleanLine)
	} else if strings.Contains(cleanLine, "Fuel Ore") || strings.Contains(cleanLine, "Organics") || strings.Contains(cleanLine, "Equipment") {
		pp.parseProductLine(cleanLine)
	} else if strings.Contains(cleanLine, "Port") {
		pp.processPortInfo(cleanLine)
	}
}

// processPortHeader processes port header information
func (pp *PortProcessor) processPortHeader(line string) {
	// Initialize port data structure
}

// parseProductLine parses product trading information
func (pp *PortProcessor) parseProductLine(line string) {
	var productType database.TProductType
	
	if strings.Contains(line, "Fuel Ore") {
		productType = database.PtFuelOre
	} else if strings.Contains(line, "Organics") {
		productType = database.PtOrganics
	} else if strings.Contains(line, "Equipment") {
		productType = database.PtEquipment
	} else {
		return
	}
	
	
	// Parse product details (amount, price, etc.)
	fields := strings.Fields(line)
	if len(fields) >= 4 {
		// Extract trading information
		pp.extractProductInfo(fields, productType)
	}
}

// extractProductInfo extracts product trading information from fields
func (pp *PortProcessor) extractProductInfo(fields []string, productType database.TProductType) {
	// This would parse the specific format of product lines
	// Example: "Fuel Ore     : 1000 buying at 15"
	for i, field := range fields {
		if field == "buying" && i+2 < len(fields) && fields[i+1] == "at" {
			price := pp.utils.StrToIntSafe(fields[i+2])
			// Process buying price
			_ = price
		} else if field == "selling" && i+2 < len(fields) && fields[i+1] == "for" {
			price := pp.utils.StrToIntSafe(fields[i+2])
			// Process selling price
			_ = price
		}
	}
}

// processPortInfo processes general port information
func (pp *PortProcessor) processPortInfo(line string) {
	// Process general port information
}

// FinalizePortData saves the current port data to database
func (pp *PortProcessor) FinalizePortData() {
	// Save port data to database
}