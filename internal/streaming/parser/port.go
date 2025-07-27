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
	pp.ctx.Logger.Printf("Processing port header: %s", line)
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
	
	pp.ctx.Logger.Printf("Processing product line for %v: %s", productType, line)
	
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
			pp.ctx.Logger.Printf("Product %v buying at price %d", productType, price)
		} else if field == "selling" && i+2 < len(fields) && fields[i+1] == "for" {
			price := pp.utils.StrToIntSafe(fields[i+2])
			pp.ctx.Logger.Printf("Product %v selling for price %d", productType, price)
		}
	}
}

// processPortInfo processes general port information
func (pp *PortProcessor) processPortInfo(line string) {
	pp.ctx.Logger.Printf("Processing port info: %s", line)
}

// FinalizePortData saves the current port data to database
func (pp *PortProcessor) FinalizePortData() {
	pp.ctx.Logger.Printf("Finalizing port data for sector %d", pp.ctx.State.CurrentSectorIndex)
	// Save port data to database
}