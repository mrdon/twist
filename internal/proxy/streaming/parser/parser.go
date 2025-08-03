package parser

import (
	"strings"
	"twist/internal/debug"
	"twist/internal/proxy/database"
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
	lineBuffer      string // Buffer for partial lines across chunks
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
	
	// Check for current sector detection from command prompts FIRST
	sp.checkForCurrentSectorPrompt(line)
	
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

// checkForCurrentSectorPrompt detects current sector from command prompts
// Based on TWX pattern: "Command [TL=00:00:00]:[sectornum] (?=Help)? :"
func (sp *SectorParser) checkForCurrentSectorPrompt(line string) {
	cleanLine := sp.utils.StripANSI(line)
	
	// Look for command prompt pattern: "Command [TL=xx:xx:xx]:[sector] (?=Help)? :"
	if strings.Contains(cleanLine, "Command [TL=") {
		debug.Log("Parser: Found command prompt pattern: %q", cleanLine)
		
		// Find the sector number between brackets after the time
		// Pattern: Command [TL=00:00:00]:[18964] (?=Help)? :
		startIdx := strings.Index(cleanLine, "]:[")
		if startIdx != -1 {
			startIdx += 3 // Skip "]:[" 
			endIdx := strings.Index(cleanLine[startIdx:], "]")
			if endIdx != -1 {
				sectorStr := cleanLine[startIdx : startIdx+endIdx]
				debug.Log("Parser: Raw sector string extracted: %q", sectorStr)
				sectorNum := sp.utils.StrToIntSafe(sectorStr)
				
				debug.Log("Parser: Extracted current sector from command prompt: %d", sectorNum)
				
				if sectorNum > 0 && sp.sectorProcessor != nil && sp.sectorProcessor.stateManager != nil {
					debug.Log("Parser: Setting current sector to %d via state manager", sectorNum)
					sp.sectorProcessor.stateManager.SetCurrentSector(sectorNum)
				} else {
					debug.Log("Parser: Cannot set sector - sectorNum: %d, sectorProcessor: %v, stateManager: %v", 
						sectorNum, sp.sectorProcessor != nil, sp.sectorProcessor != nil && sp.sectorProcessor.stateManager != nil)
				}
			} else {
				debug.Log("Parser: Could not find closing ] bracket in: %q", cleanLine[startIdx:])
			}
		} else {
			debug.Log("Parser: Could not find ]:[  pattern in: %q", cleanLine)
		}
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

// ProcessData processes raw byte data with buffering for partial lines
func (sp *SectorParser) ProcessData(data []byte) {
	text := string(data)
	
	// Add to line buffer
	sp.lineBuffer += text
	
	// Process complete lines (ending with \n or \r)
	for {
		// Look for line endings
		nlIdx := strings.Index(sp.lineBuffer, "\n")
		crIdx := strings.Index(sp.lineBuffer, "\r")
		
		// Find earliest line ending
		endIdx := -1
		if nlIdx >= 0 && crIdx >= 0 {
			if nlIdx < crIdx {
				endIdx = nlIdx
			} else {
				endIdx = crIdx
			}
		} else if nlIdx >= 0 {
			endIdx = nlIdx
		} else if crIdx >= 0 {
			endIdx = crIdx
		}
		
		if endIdx >= 0 {
			// Extract complete line
			line := sp.lineBuffer[:endIdx]
			sp.lineBuffer = sp.lineBuffer[endIdx+1:] // Skip the line ending
			
			// Process the complete line
			if strings.TrimSpace(line) != "" {
				sp.ProcessLine(line)
			}
		} else {
			// No complete line found, but check if we have a command prompt pattern
			// Command prompts often end with ": " and don't have newlines
			if strings.Contains(sp.lineBuffer, "Command [TL=") && strings.HasSuffix(strings.TrimSpace(sp.lineBuffer), ": ") {
				// This looks like a complete command prompt, process it
				debug.Log("Parser: Processing buffered command prompt: %q", sp.lineBuffer)
				sp.ProcessLine(sp.lineBuffer)
				sp.lineBuffer = "" // Clear buffer after processing
			}
			break // No more complete lines to process
		}
	}
}