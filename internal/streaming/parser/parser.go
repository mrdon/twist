package parser

import (
	"strings"
	"twist/internal/database"
)

// StateManager interface for game state updates (avoids circular import)
type StateManager interface {
	SetCurrentSector(sectorNum int)
	SetPlayerName(name string)
}

// SectorParser is the main parser that coordinates all parsing operations
type SectorParser struct {
	ctx             *ParserContext
	sectorProcessor *SectorProcessor
	portProcessor   *PortProcessor
	utils           *ParseUtils
}

// NewSectorParser creates a new sector parser with database
func NewSectorParser(db database.Database) *SectorParser {
	ctx := NewParserContext(db, nil, nil)
	
	return &SectorParser{
		ctx:             ctx,
		sectorProcessor: NewSectorProcessor(ctx),
		portProcessor:   NewPortProcessor(ctx),
		utils:           NewParseUtils(ctx),
	}
}

// NewSectorParserWithStateManager creates a new sector parser with database and state management
func NewSectorParserWithStateManager(db database.Database, stateManager StateManager) *SectorParser {
	ctx := NewParserContext(db, nil, nil)
	
	return &SectorParser{
		ctx:             ctx,
		sectorProcessor: NewSectorProcessorWithStateManager(ctx, stateManager),
		portProcessor:   NewPortProcessor(ctx),
		utils:           NewParseUtils(ctx),
	}
}

// ProcessLine processes a single line of game text
func (sp *SectorParser) ProcessLine(line string) {
	
	// Determine display state based on line content
	sp.determineDisplayState(line)
	
	// Route to appropriate processor based on current display state
	switch sp.ctx.State.CurrentDisplay {
	case DSector:
		sp.sectorProcessor.ProcessSectorLine(line)
	case DPort, DPortCIM, DPortCR:
		sp.portProcessor.ProcessPortLine(line)
	case DDensity:
		sp.processDensityLine(line)
	case DWarpLane:
		sp.processWarpLine(line)
	case DCIM:
		sp.processCIMLine(line)
	case DFigScan:
		sp.processFigScanLine(line)
	}
}

// determineDisplayState determines the current display state from line content
func (sp *SectorParser) determineDisplayState(line string) {
	cleanLine := sp.utils.StripANSI(line)
	
	// State detection logic
	if strings.Contains(cleanLine, "Sector :") {
		sp.ctx.State.CurrentDisplay = DSector
		sp.ctx.State.SectorPosition = SpNormal
	} else if strings.Contains(cleanLine, "Trading") {
		sp.ctx.State.CurrentDisplay = DPort
	} else if strings.Contains(cleanLine, "Density") {
		sp.ctx.State.CurrentDisplay = DDensity
	} else if strings.Contains(cleanLine, "Computer") {
		sp.ctx.State.CurrentDisplay = DCIM
	} else if strings.Contains(cleanLine, "Fig") {
		sp.ctx.State.CurrentDisplay = DFigScan
	}
	
	// Update sector position within sector display
	if sp.ctx.State.CurrentDisplay == DSector {
		if strings.Contains(cleanLine, "Ports") {
			sp.ctx.State.SectorPosition = SpPorts
		} else if strings.Contains(cleanLine, "Planets") {
			sp.ctx.State.SectorPosition = SpPlanets
		} else if strings.Contains(cleanLine, "Ships") {
			sp.ctx.State.SectorPosition = SpShips
		} else if strings.Contains(cleanLine, "Mines") {
			sp.ctx.State.SectorPosition = SpMines
		} else if strings.Contains(cleanLine, "Traders") {
			sp.ctx.State.SectorPosition = SpTraders
		}
	}
}

// processDensityLine processes density scan information
func (sp *SectorParser) processDensityLine(line string) {
	// Process density scan information
}

// processWarpLine processes warp lane information
func (sp *SectorParser) processWarpLine(line string) {
	// Process warp lane information
}

// processCIMLine processes computer information
func (sp *SectorParser) processCIMLine(line string) {
	// Process computer information
}

// processFigScanLine processes fighter scan information
func (sp *SectorParser) processFigScanLine(line string) {
	// Process fighter scan information
}

// ParseText processes multiple lines of text
func (sp *SectorParser) ParseText(text string) {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			sp.ProcessLine(line)
		}
	}
}

// ProcessData processes raw byte data
func (sp *SectorParser) ProcessData(data []byte) {
	text := string(data)
	sp.ParseText(text)
}