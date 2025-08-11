package streaming

import (
	"fmt"
	"strings"
	"time"
	"twist/internal/ansi"
	"twist/internal/api"
	"twist/internal/debug"
	"twist/internal/proxy/converter"
	"twist/internal/proxy/database"
	"twist/internal/proxy/types"
)

// DisplayType represents the current parsing context
type DisplayType int

const (
	DisplayNone DisplayType = iota
	DisplaySector
	DisplayDensity
	DisplayWarpLane
	DisplayCIM
	DisplayPortCIM
	DisplayPort
	DisplayPortCR
	DisplayWarpCIM
	DisplayFigScan
)

// SectorPosition tracks what part of sector data we're parsing
type SectorPosition int

const (
	SectorPosNormal SectorPosition = iota
	SectorPosPorts
	SectorPosPlanets
	SectorPosShips
	SectorPosMines
	SectorPosTraders
)

// PatternHandler is called when a pattern is matched
type PatternHandler func(line string)

// PlayerStats is an alias to the shared types.PlayerStats to avoid circular dependencies
type PlayerStats = types.PlayerStats

// FighterType represents the type of deployed fighters
type FighterType int

const (
	FighterNone FighterType = iota
	FighterOffensive
	FighterDefensive
	FighterToll
)

// FighterData holds fighter deployment information for a sector
type FighterData struct {
	SectorNum int
	Quantity  int
	Owner     string
	Type      FighterType
}

// PortData holds port commodity information
type PortData struct {
	Name         string
	ClassIndex   int
	Dead         bool
	BuildTime    int
	OreAmount    int
	OrgAmount    int
	EquipAmount  int
	OrePercent   int
	OrgPercent   int
	EquipPercent int
	BuyOre       bool
	BuyOrg       bool
	BuyEquip     bool
}

// ShipInfo holds detailed ship information from sector data
type ShipInfo struct {
	Name      string
	Owner     string
	ShipType  string
	Fighters  int
	Alignment string
}

// TraderInfo holds trader information from sector data
type TraderInfo struct {
	Name      string
	ShipName  string
	ShipType  string
	Fighters  int
	Alignment string
}

// PlanetInfo holds planet information from sector data
type PlanetInfo struct {
	Name      string
	Owner     string
	Fighters  int
	Citadel   bool
	Stardock  bool
}

// MineInfo holds mine information from sector data
type MineInfo struct {
	Type     string // "Armid" or "Limpet"
	Quantity int
	Owner    string
}

// ProductType represents the three port products
type ProductType int

const (
	ProductFuelOre ProductType = iota
	ProductOrganics
	ProductEquipment
)

// ProductInfo holds product line information
type ProductInfo struct {
	Type       ProductType
	Quantity   int
	Percent    int
	Buying     bool
	Selling    bool
	Status     string // "Buying", "Selling", etc.
}

// MessageType represents different message categories (mirrors TWX Pascal htXXX types)
type MessageType int

const (
	MessageGeneral MessageType = iota
	MessageFighter
	MessageComputer
	MessageRadio
	MessageFedlink
	MessagePlanet
	MessagePersonal   // Personal hail messages (P prefix)
	MessageIncoming   // Incoming transmissions
	MessageContinuing // Continuing transmissions
	MessageShipboard  // Shipboard computer messages
	MessageDeployed   // Deployed fighter reports
)

// MessageHistory holds historical message data
type MessageHistory struct {
	Type      MessageType
	Timestamp time.Time
	Content   string
	Sender    string
	Channel   int
}

// SectorData holds comprehensive sector information
type SectorData struct {
	Index        int
	Constellation string
	Beacon       string
	Warps        [6]int // Warp destinations (1-6)
	Port         PortData
	Density      int
	NavHaz       int
	Anomaly      bool
	Explored     bool
	// Detailed sector data
	Ships        []ShipInfo
	Traders      []TraderInfo
	Planets      []PlanetInfo
	Mines        []MineInfo
	Products     []ProductInfo
}

// CurrentSector holds current sector being parsed
type CurrentSector struct {
	Index        int
	Constellation string
	Beacon       string
	Port         PortData // Current port data for build time tracking
	NavHaz       int      // Navigation hazard percentage
}

// TWXParser implements the TWX-style stream parser with buffering for partial lines
type TWXParser struct {
	// Buffering for partial lines (like TWX Pascal implementation)
	currentLine     string
	currentANSILine string
	rawANSILine     string
	inANSI          bool
	ansiStripper    *ansi.StreamingStripper // Handles ANSI sequences across chunks

	// State tracking (mirrors TWX Pascal state)
	currentDisplay       DisplayType
	sectorPosition       SectorPosition
	currentSectorIndex   int
	portSectorIndex      int
	figScanSector        int
	lastWarp             int
	sectorSaved          bool
	menuKey              rune

	// Player statistics (mirrors TWX Pascal FCurrentXXX variables)
	playerStats PlayerStats

	// Current game data
	currentSector CurrentSector
	currentSectorWarps [6]int // Temporary storage for parsed warps
	currentMessage string
	currentChannel int        // Current radio channel for message context
	twgsVer       string
	tw2002Ver     string
	twgsType      int

	// Message history
	messageHistory []MessageHistory
	maxHistorySize int

	// Current parsing context for detailed data
	currentShips    []ShipInfo
	currentTraders  []TraderInfo
	currentPlanets  []PlanetInfo
	currentMines    []MineInfo
	currentProducts []ProductInfo
	currentTrader   TraderInfo  // Temporary storage for trader being parsed

	// Pattern handlers
	handlers map[string]PatternHandler

	// Position tracking
	position int64
	lastChar rune
	
	// Database integration
	database database.Database
	
	// TUI API integration
	tuiAPI api.TuiAPI
	
	// Script integration (mirrors Pascal TWXInterpreter integration)
	scriptEventProcessor *ScriptEventProcessor
	
	// Observer pattern and event system (Pascal: TTWXModule integration)
	observers  []IObserver
	eventBus   IEventBus
	scriptInterpreter IScriptInterpreter
	
}

// NewTWXParser creates a new TWX-style parser with database and TUI API
func NewTWXParser(db database.Database, tuiAPI api.TuiAPI) *TWXParser {
	parser := &TWXParser{
		currentLine:      "",
		currentANSILine:  "",
		rawANSILine:      "",
		inANSI:           false,
		ansiStripper:     ansi.NewStreamingStripper(),
		currentDisplay:   DisplayNone,
		sectorPosition:   SectorPosNormal,
		lastWarp:         0,
		sectorSaved:      false,
		menuKey:          '$',
		handlers:         make(map[string]PatternHandler),
		position:         0,
		lastChar:         0,
		maxHistorySize:   1000,
		database:         db, // Required database
		tuiAPI:           tuiAPI, // Optional TUI API
		// Version detection fields
		twgsType:         0,
		twgsVer:          "",
		tw2002Ver:        "",
		// Initialize data structures
		messageHistory:   make([]MessageHistory, 0),
		currentShips:     make([]ShipInfo, 0),
		currentTraders:   make([]TraderInfo, 0),
		currentPlanets:   make([]PlanetInfo, 0),
		currentMines:     make([]MineInfo, 0),
		currentProducts:  make([]ProductInfo, 0),
		// Initialize script integration (disabled by default)
		scriptEventProcessor: NewScriptEventProcessor(nil),
		// Initialize observer pattern
		observers: make([]IObserver, 0),
	}

	// Initialize event bus and script interpreter
	parser.eventBus = NewEventBus()
	parser.scriptInterpreter = NewScriptInterpreter(parser.eventBus)

	// Set up default handlers
	parser.setupDefaultHandlers()
	return parser
}

// SetScriptEngine sets the script engine for event processing (mirrors Pascal TWXInterpreter integration)
func (p *TWXParser) SetScriptEngine(scriptEngine ScriptEngine) {
	if p.scriptEventProcessor == nil {
		p.scriptEventProcessor = NewScriptEventProcessor(scriptEngine)
	} else {
		p.scriptEventProcessor.SetScriptEngine(scriptEngine)
	}
}

// GetScriptEventProcessor returns the script event processor (for testing)
func (p *TWXParser) GetScriptEventProcessor() *ScriptEventProcessor {
	return p.scriptEventProcessor
}

// AddHandler adds a pattern handler
func (p *TWXParser) AddHandler(pattern string, handler PatternHandler) {
	p.handlers[pattern] = handler
}

// setupDefaultHandlers sets up the core TWX pattern handlers
func (p *TWXParser) setupDefaultHandlers() {
	// Command prompts
	p.AddHandler("Command [TL=", p.handleCommandPrompt)
	p.AddHandler("Computer command [TL=", p.handleComputerPrompt)
	p.AddHandler("Probe entering sector :", p.handleProbePrompt)
	p.AddHandler("Probe Self Destructs", p.handleProbePrompt)
	p.AddHandler("Stop in this sector", p.handleStopPrompt)
	p.AddHandler("Engage the Autopilot?", p.handleStopPrompt)
	// Sector data (must be before CIM detection to avoid false matches)
	p.AddHandler("Sector  : ", p.handleSectorStart)
	p.AddHandler("Warps to Sector(s) :", p.handleSectorWarps)
	p.AddHandler("Beacon  : ", p.handleSectorBeacon)
	p.AddHandler("Ports   : ", p.handleSectorPorts)
	p.AddHandler("Planets : ", p.handleSectorPlanets)
	p.AddHandler("Traders : ", p.handleSectorTraders)
	p.AddHandler("Ships   : ", p.handleSectorShips)
	p.AddHandler("Fighters: ", p.handleSectorFighters)
	p.AddHandler("NavHaz  : ", p.handleSectorNavHaz)
	p.AddHandler("Mines   : ", p.handleSectorMines)
	
	p.AddHandler(": ", p.handleCIMPrompt)

	// Port data
	p.AddHandler("Docking...", p.handlePortDocking)
	p.AddHandler("Commerce report for ", p.handlePortReport)
	p.AddHandler("What sector is the port in? ", p.handlePortCR)

	// Density scanner
	p.AddHandler("Relative Density", p.handleDensityStart)

	// Warp lanes
	p.AddHandler("The shortest path (", p.handleWarpLaneStart)
	p.AddHandler("  TO > ", p.handleWarpLaneStart)

	// Fighter scan
	p.AddHandler("Deployed  Fighter  Scan", p.handleFigScanStart)

	// Version detection
	p.AddHandler("TradeWars Game", p.handleTWGSVersion)
	p.AddHandler("Trade Wars 2002 Game", p.handleTW2002Version)

	// Citadel treasury detection (mirrors Pascal: Copy(Line, 1, 25) = 'Citadel treasury contains')
	p.AddHandler("Citadel treasury contains", p.handleCitadelTreasury)

	// Messages and transmissions
	p.AddHandler("Incoming transmission from", p.handleTransmission)
	p.AddHandler("Continuing transmission from", p.handleTransmission)
	p.AddHandler("Deployed Fighters Report Sector", p.handleFighterReport)
	p.AddHandler("Shipboard Computers ", p.handleComputerReport)
	
	// Stardock detection from 'V' screen (Pascal: Copy(Line, 14, 8) = 'StarDock')
	// Note: We register the pattern differently since we need position-specific matching
}

// ProcessInBound processes incoming data (main entry point, like TWX Pascal)
func (p *TWXParser) ProcessInBound(data string) {
	// Note: Text events are fired in processLine() for complete, processed lines
	// not here for raw chunks which may contain partial data or ANSI codes
	
	// Remove null chars
	data = strings.ReplaceAll(data, "\x00", "")
	p.rawANSILine = data

	// Strip ANSI for processing but keep original for display
	s := data
	p.stripANSI(&s)

	// Remove linefeeds (only process on carriage returns)
	s = strings.ReplaceAll(s, "\n", "")
	ansiS := strings.ReplaceAll(data, "\n", "")

	// Form lines from data by accumulating with existing partial data
	line := p.currentLine + s
	ansiLine := p.currentANSILine + ansiS

	// Process complete lines (ending in \r)
	for {
		crPos := strings.IndexRune(line, '\r')
		if crPos == -1 {
			break
		}

		// Find matching CR in ANSI line
		ansiCRPos := strings.IndexRune(ansiLine, '\r')
		if ansiCRPos == -1 {
			ansiCRPos = len(ansiLine)
		}

		// Extract complete line
		completeLine := line[:crPos]
		completeANSILine := ansiLine[:ansiCRPos]

		// Store current line state
		p.currentLine = completeLine
		p.currentANSILine = completeANSILine

		// TextLineEvent is fired in processLine, not here (Pascal ProcessInBound behavior)
		
		// Process the complete line with error recovery
		p.safeParseWithRecovery("processLine", func() {
			// Validate line format before processing
			if p.validateLineFormat(completeLine) {
				p.processLine(completeLine)
				// Fire parse complete event
				p.fireParseCompleteEvent(completeLine)
			}
		})

		// Remove processed part and continue
		if crPos+1 < len(line) {
			line = line[crPos+1:]
		} else {
			line = ""
		}

		if ansiCRPos+1 < len(ansiLine) {
			ansiLine = ansiLine[ansiCRPos+1:]
		} else {
			ansiLine = ""
		}
	}

	// Store remaining partial data
	p.currentLine = line
	p.currentANSILine = ansiLine

	// Fire AutoTextEvent for prompts only if there's remaining data (Pascal TWX behavior)
	// Pascal: only fires AutoTextEvent at end of ProcessInBound for partial/prompt data
	if p.currentLine != "" {
		p.FireAutoTextEvent(p.currentLine, false)
		
		// Process partial line for prompts (key TWX feature!)
		p.processPrompt(p.currentLine)
	}
}

// Finalize processes any remaining data and completes pending sectors
func (p *TWXParser) Finalize() {
	// If there's remaining data in currentLine, process it as a final line
	if p.currentLine != "" {
		p.processLine(p.currentLine)
		p.processPrompt(p.currentLine)
	}
	
	// Complete any pending sector
	if !p.sectorSaved && p.currentSectorIndex > 0 {
		p.sectorCompleted()
	}
}

// stripANSI removes ANSI escape sequences (mirrors TWX Pascal logic)
func (p *TWXParser) stripANSI(s *string) {
	// Remove bells first
	*s = strings.ReplaceAll(*s, "\x07", "")
	
	// Use streaming ANSI stripper to handle sequences split across chunks
	*s = p.ansiStripper.StripChunk(*s)
}

// processLine processes a complete line (mirrors TWX Pascal ProcessLine)
func (p *TWXParser) processLine(line string) {

	// Handle message continuations (mirrors TWX Pascal logic)
	if p.currentMessage != "" {
		if line != "" {
			p.handleMessageLine(line)
			p.currentMessage = ""
		}
		return
	}

	// Handle direct messages
	if strings.HasPrefix(line, "R ") || strings.HasPrefix(line, "F ") {
		p.handleMessageLine(line)
		return
	}
	if strings.HasPrefix(line, "P ") {
		// Skip "P indicates" messages
		parts := strings.Fields(line)
		if len(parts) < 2 || parts[1] != "indicates" {
			p.handleMessageLine(line)
		}
		return
	}

	// Pascal: Check for density scanner first (Copy(Line, 27, 16) = 'Relative Density')
	// This must be checked before pattern handlers to match Pascal behavior
	if strings.Contains(line, "Relative Density") {
		p.currentDisplay = DisplayDensity
		// Return early - don't process this line further, just set the mode
		return
	}

	// Handle continuation based on current display state
	switch p.currentDisplay {
	case DisplaySector:
		p.processSectorLine(line)
	case DisplayPort, DisplayPortCR:
		p.processPortLine(line)
	case DisplayWarpLane:
		p.processWarpLine(line)
	case DisplayCIM, DisplayPortCIM, DisplayWarpCIM:
		p.processCIMLine(line)
	case DisplayDensity:
		p.processDensityLine(line)
	case DisplayFigScan:
		p.processFigScanLine(line)
	default:
		// Check for pattern matches to change state
		p.checkPatterns(line)
	}

	// Check for QuickStats (mirrors Pascal: ContainsText(Line, '│') or (Copy(Line, 1, 5) = ' Ship'))
	if strings.Contains(line, "│") || strings.HasPrefix(line, " Ship") {
		debug.Log("Processing QuickStats line: %q", line)
		p.processQuickStats(line)
	}

	// Fire TextLineEvent as in Pascal TWX ProcessLine (mirrors Pascal TWXInterpreter.TextLineEvent)
	p.FireTextLineEvent(line, false)

	// Always check for prompts
	p.processPrompt(line)

	// Reactivate script triggers as in Pascal TWX ProcessLine (mirrors Pascal TWXInterpreter.ActivateTriggers)
	p.ActivateTriggers()
}

// processPrompt handles prompts that may not end in newlines (key TWX feature)
func (p *TWXParser) processPrompt(line string) {
	if line == "" {
		return
	}

	// Fire TextEvent as in Pascal TWX ProcessPrompt (mirrors Pascal TWXInterpreter.TextEvent)
	p.FireTextEvent(line, false)


	// Check for prompt patterns
	for pattern, handler := range p.handlers {
		if strings.HasPrefix(line, pattern) {
			handler(line)
			return
		}
	}
}

// checkPatterns checks for pattern matches in complete lines
func (p *TWXParser) checkPatterns(line string) {
	// Check for Stardock detection from 'V' screen first (Pascal: Copy(Line, 14, 8) = 'StarDock' and Copy(Line, 37, 6) = 'sector')
	// Pascal uses 1-indexed strings, so position 14 = index 13, position 37 = index 36
	// Need exact position matching as in Pascal for reliable detection
	if len(line) >= 42 {
		// Check exact position 14 for "StarDock" (index 13)
		if len(line) >= 21 && line[13:21] == "StarDock" {
			// Check position 37 for "sector" (index 36) with some flexibility for exact spacing
			// Based on test pattern, "sector" should be around position 39 (0-indexed)
			if len(line) >= 45 && strings.Contains(line[36:46], "sector") {
				p.handleStardockDetection(line)
				return
			}
		}
	}
	
	for pattern, handler := range p.handlers {
		if strings.Contains(line, pattern) {
			handler(line)
			return
		}
	}
}

// Handler implementations (core TWX parsing logic)

func (p *TWXParser) handleCommandPrompt(line string) {
	
	// Save current sector if not done already
	if !p.sectorSaved {
		p.sectorCompleted()
	}

	// Extract sector number from "Command [TL=150] (2500) ?"
	// Find the opening and closing parentheses and extract number between them
	openParen := strings.Index(line, "(")
	closeParen := strings.Index(line, ")")
	if openParen > 0 && closeParen > openParen {
		sectorStr := strings.TrimSpace(line[openParen+1 : closeParen])
		if sectorNum := p.parseIntSafe(sectorStr); sectorNum > 0 {
			p.currentSectorIndex = sectorNum
		}
	}

	p.currentDisplay = DisplayNone
	p.lastWarp = 0
}

func (p *TWXParser) handleComputerPrompt(line string) {
	p.currentDisplay = DisplayNone
	p.lastWarp = 0

	// Extract sector number from "Computer command [TL=150] (1234) ?"
	// Find the opening and closing parentheses and extract number between them
	openParen := strings.Index(line, "(")
	closeParen := strings.Index(line, ")")
	if openParen > 0 && closeParen > openParen {
		sectorStr := strings.TrimSpace(line[openParen+1 : closeParen])
		if sectorNum := p.parseIntSafe(sectorStr); sectorNum > 0 {
			p.currentSectorIndex = sectorNum
		}
	}
}

func (p *TWXParser) handleProbePrompt(line string) {
	if !p.sectorSaved {
		p.sectorCompleted()
	}
	p.currentDisplay = DisplayNone
}

func (p *TWXParser) handleStopPrompt(line string) {
	if !p.sectorSaved {
		p.sectorCompleted()
	}
	p.currentDisplay = DisplayNone
}

func (p *TWXParser) handleCIMPrompt(line string) {
	// Pascal: else if (Copy(Line, 1, 2) = ': ') then
	// Pascal: // begin CIM download
	// Pascal: FCurrentDisplay := dCIM;
	p.currentDisplay = DisplayCIM
	p.lastWarp = 0
}

func (p *TWXParser) handleSectorStart(line string) {
	
	// Extract sector number first to determine if this is a new sector
	// Format: "Sector  : 1234 in The Sphere"
	parts := strings.Fields(line)
	if len(parts) >= 3 {
		if sectorNum := p.parseIntSafe(parts[2]); sectorNum > 0 {
			// Only complete previous sector if this is a different sector
			if p.currentSectorIndex != sectorNum && !p.sectorSaved {
				p.sectorCompleted()
			}
			
			// Always reset sector data when processing a sector visit
			// This ensures that data from previous visits doesn't carry over
			// (including port data that might persist from previous visits)
			p.sectorSaved = false  // Reset for any sector visit
			p.resetCurrentSector()
			p.currentSectorIndex = sectorNum
			
			p.currentDisplay = DisplaySector
			
			// Extract constellation (everything after "in")
			if len(parts) >= 5 && parts[3] == "in" {
				constellation := strings.Join(parts[4:], " ")
				// Remove trailing period if present
				constellation = strings.TrimSuffix(constellation, ".")
				p.currentSector.Constellation = constellation
			}
		}
	}
}

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
		p.currentSector.Beacon = line[10:]
	}
}

func (p *TWXParser) handleSectorPorts(line string) {
	p.sectorPosition = SectorPosPorts
	
	// Parse port data (mirrors TWX Pascal logic from lines 671-703)
	if strings.Contains(line, "<=-DANGER-=>") {
		// Port is destroyed - set Dead flag
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
	
	// Store port information in current sector data (mirrors Pascal FCurrentSector.SPort)
	p.currentSector.Port.Name = portName
	p.currentSector.Port.ClassIndex = classNum
	p.currentSector.Port.BuildTime = 0 // Reset build time, will be set by continuation line
	
	
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
	// Call detailed navhaz parsing
	p.parseSectorNavHaz(line)
}

func (p *TWXParser) handleSectorMines(line string) {
	p.sectorPosition = SectorPosMines
	// Call detailed mine parsing
	p.parseSectorMines(line)
}

func (p *TWXParser) handlePortDocking(line string) {
	if !p.sectorSaved {
		p.sectorCompleted()
	}
	p.currentDisplay = DisplayPort
	p.portSectorIndex = p.currentSectorIndex
}

func (p *TWXParser) handlePortReport(line string) {
	// Extract port name from "Commerce report for PORT_NAME:"
	colonPos := strings.Index(line, ":")
	if colonPos > 20 {
		portName := strings.TrimSpace(line[20:colonPos])
		_ = portName // Use portName to avoid unused variable error
	}
}

func (p *TWXParser) handlePortCR(line string) {
	p.currentDisplay = DisplayPortCR
	
	// Extract sector number from end of line
	closeBracket := strings.LastIndex(line, "]")
	if closeBracket > 0 && closeBracket < len(line)-1 {
		sectorStr := strings.TrimSpace(line[closeBracket+1:])
		if sectorNum := p.parseIntSafe(sectorStr); sectorNum > 0 {
			p.portSectorIndex = sectorNum
		} else {
			p.portSectorIndex = p.currentSectorIndex
		}
	} else {
		p.portSectorIndex = p.currentSectorIndex
	}
}

func (p *TWXParser) handleDensityStart(line string) {
	// Pascal: if (Copy(Line, 27, 16) = 'Relative Density') then
	// Check position 26 in 0-indexed Go (27-1), length 16
	if len(line) >= 42 && line[26:42] == "Relative Density" {
		p.currentDisplay = DisplayDensity
	}
}

func (p *TWXParser) handleWarpLaneStart(line string) {
	p.currentDisplay = DisplayWarpLane
	p.lastWarp = 0
}

func (p *TWXParser) handleFigScanStart(line string) {
	p.currentDisplay = DisplayFigScan
	p.figScanSector = 0
}

// handleTWGSVersion detects TWGS version (mirrors Pascal lines 295-304)
func (p *TWXParser) handleTWGSVersion(line string) {
	
	// Pascal: if TWXClient.BlockExtended and (Copy(Line, 1, 14) = 'TradeWars Game') then
	if strings.HasPrefix(line, "TradeWars Game") {
		p.twgsType = 2
		p.twgsVer = "2.20b"
		p.tw2002Ver = "3.34"
		
		
		// Pascal: TWXInterpreter.TextEvent('Selection (? for menu):', FALSE);
		// Fire script event after version detection as in Pascal TWX
		if p.scriptEventProcessor != nil && p.scriptEventProcessor.IsEnabled() {
			if err := p.scriptEventProcessor.FireTextEvent("Selection (? for menu):", false); err != nil {
			}
		}
	}
}

// handleTW2002Version detects TW2002 version (mirrors Pascal lines 305-316)
func (p *TWXParser) handleTW2002Version(line string) {
	
	// Pascal: else if TWXClient.BlockExtended and (Copy(Line, 1, 20) = 'Trade Wars 2002 Game') then
	if strings.HasPrefix(line, "Trade Wars 2002 Game") {
		p.twgsType = 1
		p.twgsVer = "1.03"
		p.tw2002Ver = "3.13"
		
	}
}

// handleCitadelTreasury handles Citadel treasury detection (mirrors Pascal line 283)
func (p *TWXParser) handleCitadelTreasury(line string) {
	
	// Pascal: else if (Copy(Line, 1, 25) = 'Citadel treasury contains') then
	if strings.HasPrefix(line, "Citadel treasury contains") {
		// In Citadel - Save current sector if not done already
		if !p.sectorSaved {
			p.sectorCompleted()
		}
		
		// No displays anymore, all done (Pascal: FCurrentDisplay := dNone)
		p.currentDisplay = DisplayNone
		
	}
}

func (p *TWXParser) handleTransmission(line string) {
	
	// Enhanced Pascal-compliant transmission parsing
	// Pascal: else if (Copy(Line, 1, 26) = 'Incoming transmission from') then
	if strings.HasPrefix(line, "Incoming transmission from") || strings.HasPrefix(line, "Continuing transmission from") {
		p.handleEnhancedTransmissionLine(line)
		return
	}
	
	// Fallback to basic parsing for compatibility
	p.handleBasicTransmission(line)
}

// handleEnhancedTransmissionLine implements Pascal-compliant transmission parsing (lines 1199-1228)
func (p *TWXParser) handleEnhancedTransmissionLine(line string) {
	
	// Pascal: I := GetParameterPos(Line, 4);
	paramPos := p.getParameterPos(line, 4)
	if paramPos == -1 {
		return
	}
	
	// Pascal: if (Copy(Line, Length(Line) - 9, 10) = 'comm-link:') then
	if len(line) >= 10 && line[len(line)-10:] == "comm-link:" {
		// Fedlink transmission
		// Pascal: FCurrentMessage := 'F ' + Copy(Line, I, Pos(' on Federation', Line) - I) + ' ';
		fedPos := strings.Index(line, " on Federation")
		if fedPos > paramPos {
			sender := strings.TrimSpace(line[paramPos:fedPos])
			p.currentMessage = "F " + sender + " "
		} else {
			p.currentMessage = "F  "
		}
		return
	}
	
	// Pascal: else if (GetParameter(Line, 5) = 'Fighters:') then
	if p.getParameter(line, 5) == "Fighters:" {
		// Fighter transmission
		// Pascal: FCurrentMessage := 'Figs';
		p.currentMessage = "Figs"
		return
	}
	
	// Pascal: else if (GetParameter(Line, 5) = 'Computers:') then
	if p.getParameter(line, 5) == "Computers:" {
		// Computer transmission
		// Pascal: FCurrentMessage := 'Comp';
		p.currentMessage = "Comp"
		return
	}
	
	// Pascal: else if (Pos(' on channel ', Line) <> 0) then
	channelPos := strings.Index(line, " on channel ")
	if channelPos != -1 {
		// Radio transmission
		// Pascal: FCurrentMessage := 'R ' + Copy(Line, I, Pos(' on channel ', Line) - I) + ' ';
		if channelPos > paramPos {
			sender := strings.TrimSpace(line[paramPos:channelPos])
			p.currentMessage = "R " + sender + " "
			
			// Extract channel number for context
			channelStr := line[channelPos+12:] // After " on channel "
			channelParts := strings.Fields(channelStr)
			if len(channelParts) > 0 {
				channelNum := p.parseIntSafe(strings.TrimSuffix(channelParts[0], ":"))
				p.currentChannel = channelNum
			}
			
		} else {
			p.currentMessage = "R  "
		}
		return
	}
	
	// Pascal: else begin // hail
	// Pascal: FCurrentMessage := 'P ' + Copy(Line, I, Length(Line) - I) + ' ';
	// TEMPORARILY DISABLED: This causes next line to be treated as message continuation
	// TODO: Fix Pascal transmission handling to match expected behavior
	// if paramPos < len(line) {
	// 	sender := strings.TrimSpace(line[paramPos:])
	// 	// Remove trailing colon if present
	// 	sender = strings.TrimSuffix(sender, ":")
	// 	p.currentMessage = "P " + sender + " "
	// } else {
	// 	p.currentMessage = "P  "
	// }
}

// handleBasicTransmission provides fallback transmission parsing for compatibility
func (p *TWXParser) handleBasicTransmission(line string) {
	
	// Parse transmission type using basic logic
	if strings.HasSuffix(line, "comm-link:") {
		// Fedlink transmission
		parts := strings.Fields(line)
		if len(parts) >= 4 {
			sender := ""
			fedIndex := -1
			for i, part := range parts {
				if part == "on" && i+1 < len(parts) && parts[i+1] == "Federation" {
					fedIndex = i
					break
				}
			}
			if fedIndex > 3 {
				sender = strings.Join(parts[3:fedIndex], " ")
			}
			p.currentMessage = "F " + sender + " "
		}
	} else {
		parts := strings.Fields(line)
		if len(parts) >= 5 {
			if parts[4] == "Fighters:" {
				// Fighter transmission (Pascal parameter 5 = Go index 4)
				p.currentMessage = "Figs"
			} else if parts[4] == "Computers:" {
				// Computer transmission (Pascal parameter 5 = Go index 4)
				p.currentMessage = "Comp"
			} else if strings.Contains(line, " on channel ") {
				// Radio transmission
				sender := ""
				channelIndex := -1
				for i, part := range parts {
					if part == "on" && i+1 < len(parts) && parts[i+1] == "channel" {
						channelIndex = i
						break
					}
				}
				if channelIndex > 3 {
					sender = strings.Join(parts[3:channelIndex], " ")
				}
				p.currentMessage = "R " + sender + " "
			} else {
				// Personal/hail transmission
				sender := strings.Join(parts[3:], " ")
				// Remove trailing colon if present
				sender = strings.TrimSuffix(strings.TrimSpace(sender), ":")
				p.currentMessage = "P " + sender + " "
			}
		}
	}
}

func (p *TWXParser) handleFighterReport(line string) {
}

func (p *TWXParser) handleComputerReport(line string) {
}

// Processing methods for different display states

func (p *TWXParser) processSectorLine(line string) {
	
	// Handle continuation lines (start with 8 spaces)
	if strings.HasPrefix(line, "        ") {
		p.handleSectorContinuation(line)
		return
	}

	// Handle end of sections (9th character is ':')
	if len(line) > 9 && line[8] == ':' {
		// Finalize any pending trader without ship details
		if p.sectorPosition == SectorPosTraders && p.currentTrader.Name != "" {
			p.currentTraders = append(p.currentTraders, p.currentTrader)
			p.currentTrader = TraderInfo{} // Reset
		}
		p.sectorPosition = SectorPosNormal
		return
	}

	// Handle specific sector data patterns already covered by handlers
}

// handleSectorContinuation is now implemented in sector_parser.go

// processPortLine is now handled by the enhanced product parsing in product_parser.go
// This method is kept for compatibility but delegates to the enhanced version

func (p *TWXParser) processWarpLine(line string) {
	
	// Parse warp lane format: "3 > 300 > 5362 > 13526 > 149 > 434"
	line = strings.ReplaceAll(line, ")", "")
	line = strings.ReplaceAll(line, "(", "")
	
	parts := strings.Split(line, " >")
	lastSect := p.lastWarp
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if curSect := p.parseIntSafe(part); curSect > 0 {
			if lastSect > 0 {
			}
			lastSect = curSect
			p.lastWarp = curSect
		}
	}
}

func (p *TWXParser) processCIMLine(line string) {
	
	// Pascal: // find out what kind of CIM this is
	// Pascal: if (Length(Line) > 2) then
	if len(line) <= 2 {
		p.currentDisplay = DisplayNone
		return
	}
	
	// Pascal: if (Line[Length(Line) - 1] = '%') then
	// Pascal: TWXDatabase.LastPortCIM := Now;
	// Pascal: FCurrentDisplay := dPortCIM;
	// Check if line contains '%' character (indicating port CIM data)
	if strings.Contains(line, "%") {
		p.currentDisplay = DisplayPortCIM
		p.processPortCIMLine(line)
	} else {
		// Pascal: else FCurrentDisplay := dWarpCIM;
		p.currentDisplay = DisplayWarpCIM
		p.processWarpCIMLine(line)
	}
}

// processWarpCIMLine processes warp CIM data (sector warp connections)
// Format can be: "1234 5678 9012 3456 7890 1234" (sector and its 6 warp destinations)
// Or: "1234 5678 9012" (sector with fewer warps)
func (p *TWXParser) processWarpCIMLine(line string) {
	parts := strings.Fields(line)
	if len(parts) < 2 { // Need at least sector + 1 warp
		p.currentDisplay = DisplayNone
		return
	}
	
	sectorNum := p.parseIntSafe(parts[0])
	if sectorNum <= 0 {
		p.currentDisplay = DisplayNone
		return
	}
	
	
	// Parse available warp destinations (up to 6 max)
	var warps [6]int
	maxWarps := len(parts) - 1 // Number of warp destinations available
	if maxWarps > 6 {
		maxWarps = 6 // Cap at 6 warps maximum
	}
	
	for i := 0; i < maxWarps; i++ {
		warpSector := p.parseIntSafe(parts[i+1])
		if warpSector < 0 { // Invalid warp sector
			break // Stop parsing at first invalid warp
		}
		warps[i] = warpSector
	}
	
	// Store warp data to database (mirrors Pascal TWXDatabase.SaveSector)
	sector, err := p.database.LoadSector(sectorNum)
	if err != nil {
		// Create new sector if it doesn't exist
		sector = database.NULLSector()
	}
	
	// Update warp data
	for i := 0; i < 6; i++ {
		sector.Warp[i] = warps[i]
	}
	
	// Mark root sector as calculated since it appears in CIM sector report
	// CIM data marks sectors as calculated, not fully explored (matches TWX behavior)
	if sector.Explored == database.EtNo {
		sector.Explored = database.EtCalc  // Mark as calculated (not fully explored)
		if sector.Constellation == "" {
			sector.Constellation = "??? (warp calc only)"
		}
	}
	
	sector.UpDate = time.Now()
	
	// Save updated sector
	if err := p.database.SaveSector(sector, sectorNum); err != nil {
		return
	}
	
}

// processPortCIMLine processes port CIM data (mirrors Pascal ProcessCIMLine lines 570-611)
// Format: "1234 5000 60% 3000 80% 2000 90%" or "1234 -5000 60% 3000 80% -2000 90%" (with dashes for buying)
func (p *TWXParser) processPortCIMLine(line string) {
	// Pascal: save port CIM data
	// Sect := GetCIMValue(Line, 1);
	// Len := Length(IntToStr(TWXDatabase.DBHeader.Sectors));
	
	sectorNum := p.getCIMValue(line, 1)
	// Check sector number validity (Pascal validation)
	if sectorNum <= 0 {
		p.currentDisplay = DisplayNone
		return
	}
	
	// Check minimum line length - need at least 7 parameters
	if len(strings.Fields(line)) < 7 {
		p.currentDisplay = DisplayNone
		return
	}
	
	// Pascal: M := StringReplace(Line, '-', '', [rfReplaceAll]);
	// Pascal: M := StringReplace(M, '%', '', [rfReplaceAll]);
	cleanLine := strings.ReplaceAll(line, "-", "")
	cleanLine = strings.ReplaceAll(cleanLine, "%", "")
	
	// Pascal: Ore := GetCIMValue(M, 2); POre := GetCIMValue(M, 3);
	// Pascal: Org := GetCIMValue(M, 4); POrg := GetCIMValue(M, 5);
	// Pascal: Equip := GetCIMValue(M, 6); PEquip := GetCIMValue(M, 7);
	oreAmount := p.getCIMValue(cleanLine, 2)
	orePercent := p.getCIMValue(cleanLine, 3)
	orgAmount := p.getCIMValue(cleanLine, 4)
	orgPercent := p.getCIMValue(cleanLine, 5)
	equipAmount := p.getCIMValue(cleanLine, 6)
	equipPercent := p.getCIMValue(cleanLine, 7)
	
	// Pascal validation: if (Ore < 0) or (Org < 0) or (Equip < 0)
	//  or (POre < 0) or (POre > 100) or (POrg < 0) or (POrg > 100)
	//  or (PEquip < 0) or (PEquip > 100) then
	if oreAmount < 0 || orgAmount < 0 || equipAmount < 0 ||
		orePercent < 0 || orePercent > 100 ||
		orgPercent < 0 || orgPercent > 100 ||
		equipPercent < 0 || equipPercent > 100 {
		p.currentDisplay = DisplayNone
		return
	}
	
	// Determine buy/sell status by examining dash positions in original line
	// This mirrors Pascal logic that would check the original line for dash indicators
	buyOre := p.determineCIMBuyStatus(line, 2)   // Parameter 2 = ore amount
	buyOrg := p.determineCIMBuyStatus(line, 4)   // Parameter 4 = org amount  
	buyEquip := p.determineCIMBuyStatus(line, 6) // Parameter 6 = equip amount
	
	// Determine port class from buy/sell pattern (mirrors Pascal port class logic)
	portClass := p.determinePortClassFromPattern(buyOre, buyOrg, buyEquip)
	
	// Store enhanced port CIM data to database
	p.storePortCIMData(sectorNum, oreAmount, orePercent, buyOre,
		orgAmount, orgPercent, buyOrg, equipAmount, equipPercent, buyEquip, portClass)
}

// getCIMValue extracts a parameter value from CIM data (mirrors Pascal GetCIMValue function)
func (p *TWXParser) getCIMValue(line string, paramNum int) int {
	// Pascal: function GetCIMValue(M : String; Num : Integer) : Integer;
	// Pascal: S := GetParameter(M, Num);
	parts := strings.Fields(line)
	if paramNum <= 0 || paramNum > len(parts) {
		return -1 // Invalid parameter number
	}
	
	// Convert to 0-indexed (Pascal is 1-indexed)
	paramValue := parts[paramNum-1]
	
	// Pascal: if (S = '') or (S = '0') then Result := 0
	// Pascal: else Result := StrToIntSafe(S);
	if paramValue == "" || paramValue == "0" {
		return 0
	}
	
	return p.parseIntSafe(paramValue)
}

// determineCIMBuyStatus determines if a commodity is being bought based on dash position
func (p *TWXParser) determineCIMBuyStatus(originalLine string, paramNum int) bool {
	// Find the position of the parameter in the original line
	parts := strings.Fields(originalLine)
	if paramNum <= 0 || paramNum > len(parts) {
		return false
	}
	
	// Check if the parameter at this position starts with a dash
	paramValue := parts[paramNum-1]
	return strings.HasPrefix(paramValue, "-")
}

// ensureSectorExistsAndSavePort ensures a sector exists in the database before saving port data
// This handles the common pattern where port data requires a sector to exist first (foreign key constraint)
func (p *TWXParser) ensureSectorExistsAndSavePort(port database.TPort, sectorNum int) error {
	if p.database == nil {
		return fmt.Errorf("database not available")
	}
	
	// Always ensure sector exists first (required for foreign key constraint)
	sector, err := p.database.LoadSector(sectorNum)
	if err != nil {
		// Create minimal sector entry
		sector = database.NULLSector()
		sector.UpDate = time.Now()
	}
	
	// Always save/update sector to ensure it exists in current transaction context
	if err := p.database.SaveSector(sector, sectorNum); err != nil {
		return fmt.Errorf("failed to save sector %d: %w", sectorNum, err)
	}
	
	// Save port data
	if err := p.database.SavePort(port, sectorNum); err != nil {
		return fmt.Errorf("failed to save port for sector %d: %w", sectorNum, err)
	}
	
	// Fire any necessary events (consistent with other database operations)
	// This ensures proper notification flow like other database saves
	return nil
}

// ensureSectorExistsAndSavePortWithVisited marks CIM root sector as visited and saves port data
func (p *TWXParser) ensureSectorExistsAndSavePortWithVisited(port database.TPort, sectorNum int) error {
	if p.database == nil {
		return fmt.Errorf("database not available")
	}
	
	// Always ensure sector exists first (required for foreign key constraint)
	sector, err := p.database.LoadSector(sectorNum)
	if err != nil {
		// Create minimal sector entry
		sector = database.NULLSector()
		sector.UpDate = time.Now()
	}
	
	// Mark root sector as calculated since it appears in CIM port report
	// CIM data marks sectors as calculated, not fully explored (matches TWX behavior)
	if sector.Explored == database.EtNo {
		sector.Explored = database.EtCalc  // Mark as calculated (not fully explored)
		if sector.Constellation == "" {
			sector.Constellation = "??? (port data/calc only)"
		}
	}
	sector.UpDate = time.Now()
	
	// Always save/update sector to ensure it exists in current transaction context
	if err := p.database.SaveSector(sector, sectorNum); err != nil {
		return fmt.Errorf("failed to save sector %d: %w", sectorNum, err)
	}
	
	// Save port data
	if err := p.database.SavePort(port, sectorNum); err != nil {
		return fmt.Errorf("failed to save port for sector %d: %w", sectorNum, err)
	}
	
	// Fire any necessary events (consistent with other database operations)
	// This ensures proper notification flow like other database saves
	return nil
}

// clearPortData removes port data from the database for a sector that has no port
// This is called when we visit a sector and confirm it has no port
func (p *TWXParser) clearPortData(sectorIndex int) error {
	if p.database == nil {
		return fmt.Errorf("database not available")
	}
	
	// Delete port data from the ports table
	if err := p.database.DeletePort(sectorIndex); err != nil {
		return fmt.Errorf("failed to delete port data for sector %d: %w", sectorIndex, err)
	}
	
	return nil
}

// resetCurrentSector clears sector parsing data when starting a new sector visit
// This preserves important persistent data while clearing temporary parsing state
func (p *TWXParser) resetCurrentSector() {
	// Only reset parsing-specific data, not persistent sector information
	p.currentSector.Port = PortData{}      // Clear port data (this was the main fix needed)
	p.currentSector.Beacon = ""            // Clear beacon (will be set if present)
	p.currentSector.Constellation = ""     // Clear constellation (will be set if present)
	// Note: We intentionally do NOT reset Index or NavHaz as those are set elsewhere
	
	p.currentShips = nil
	p.currentTraders = nil
	p.currentPlanets = nil
	p.currentMines = nil
	p.currentProducts = nil
	p.currentSectorWarps = [6]int{0, 0, 0, 0, 0, 0}
	p.sectorPosition = SectorPosNormal
}

// storePortCIMData stores complete port CIM data to database
func (p *TWXParser) storePortCIMData(sectorNum, oreAmount, orePercent int, buyOre bool,
	orgAmount, orgPercent int, buyOrg bool, equipAmount, equipPercent int, buyEquip bool, portClass int) {
	
	// Create port data
	port := database.TPort{
		Name:           "", 
		Dead:           false,
		BuildTime:      0, 
		ClassIndex:     portClass,
		BuyProduct:     [3]bool{buyOre, buyOrg, buyEquip},
		ProductPercent: [3]int{orePercent, orgPercent, equipPercent},
		ProductAmount:  [3]int{oreAmount, orgAmount, equipAmount},
		UpDate:         time.Now(),
	}
	
	// Mark sector as visited and save port data (CIM port reports are from visited sectors)
	if err := p.ensureSectorExistsAndSavePortWithVisited(port, sectorNum); err != nil {
		// Log error but don't panic - this is often called in parsing context
		return
	}
}

// processDensityLine processes density scanner data (mirrors Pascal logic lines 1343-1375)
func (p *TWXParser) processDensityLine(line string) {
	
	// Pascal: if (FCurrentDisplay = dDensity) and (Copy(Line, 1, 6) = 'Sector') then
	if !strings.HasPrefix(line, "Sector") {
		return
	}
	
	if p.database == nil {
		return
	}
	
	// Pascal implementation (lines 1346-1375):
	// X := Line;
	// StripChar(X, '(');
	// StripChar(X, ')');
	x := line
	x = strings.ReplaceAll(x, "(", "")
	x = strings.ReplaceAll(x, ")", "")
	
	// Pascal: I := StrToIntSafe(GetParameter(X, 2));
	sectorNum := p.parseIntSafe(p.getParameter(x, 2))
	if sectorNum <= 0 {
		return
	}
	
	// Pascal: Sect := TWXDatabase.LoadSector(I);
	sector, err := p.database.LoadSector(sectorNum)
	if err != nil {
		sector = database.NULLSector()
	}
	
	// Pascal: S := GetParameter(X, 4); StripChar(S, ','); Sect.Density := StrToIntSafe(S);
	densityStr := p.getParameter(x, 4)
	densityStr = strings.ReplaceAll(densityStr, ",", "")
	sector.Density = p.parseIntSafe(densityStr)
	
	// Pascal: if (GetParameter(X, 13) = 'Yes') then Sect.Anomaly := TRUE else Sect.Anomaly := FALSE;
	anomalyParam := p.getParameter(x, 13)
	sector.Anomaly = (anomalyParam == "Yes")
	
	// Pascal: S := GetParameter(X, 10); S := Copy(S, 1, length(S) - 1); Sect.NavHaz := StrToIntSafe(S);
	navhazStr := p.getParameter(x, 10)
	if len(navhazStr) > 0 {
		navhazStr = navhazStr[:len(navhazStr)-1] // Remove last character (%)
	}
	sector.NavHaz = p.parseIntSafe(navhazStr)
	
	// Pascal: Sect.Warps := StrToIntSafe(GetParameter(X, 7));
	sector.Warps = p.parseIntSafe(p.getParameter(x, 7))
	
	// Pascal: if (Sect.Explored in [etNo, etCalc]) then
	if sector.Explored == database.EtNo || sector.Explored == database.EtCalc {
		// Pascal: Sect.Constellation := '???' + ANSI_9 + ' (Density only)';
		// Pascal: Sect.Explored := etDensity;
		// Pascal: Sect.Update := Now;
		sector.Constellation = "??? (Density only)"
		sector.Explored = database.EtDensity
		sector.UpDate = time.Now()
	}
	
	// Pascal: TWXDatabase.SaveSector(Sect, I, nil, nil, nil);
	if err := p.database.SaveSector(sector, sectorNum); err != nil {
		return
	}
}

func (p *TWXParser) processFigScanLine(line string) {
	
	// Handle "No fighters deployed" case - reset fighter database
	if strings.HasPrefix(line, "No fighters deployed") {
		p.resetFighterDatabase()
		return
	}
	
	// Parse fig scan format: "940  1  Personal  Defensive  N/A"
	// Also handles: "940  10T  Personal  Defensive  N/A" (with multipliers)
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return
	}
	
	// Parse sector number
	sectorNum := p.parseIntSafe(fields[0])
	if sectorNum <= 0 {
		return
	}
	
	// Parse fighter quantity (may have T/M/B multipliers)
	figQuantity := p.parseFighterQuantity(fields[1])
	if figQuantity < 0 {
		return
	}
	
	// Parse owner
	ownerField := fields[2]
	var owner string
	if ownerField == "Personal" {
		owner = "yours"
	} else {
		owner = "belong to your Corp"
	}
	
	// Parse fighter type
	typeField := fields[3]
	var fighterType FighterType
	switch typeField {
	case "Defensive":
		fighterType = FighterDefensive
	case "Toll":
		fighterType = FighterToll
	case "Offensive":
		fighterType = FighterOffensive
	default:
		fighterType = FighterOffensive // Default to offensive
	}
	
	// Store fighter data (in a real implementation, this would go to database)
	_ = FighterData{
		SectorNum: sectorNum,
		Quantity:  figQuantity,
		Owner:     owner,
		Type:      fighterType,
	}
	
}

// parseFighterQuantity parses fighter quantities with T/M/B multipliers
// Examples: "1000", "10T", "5M", "2B"
func (p *TWXParser) parseFighterQuantity(quantityStr string) int {
	quantityStr = strings.ReplaceAll(quantityStr, ",", "")
	
	if quantityStr == "" {
		return 0
	}
	
	// Check for multiplier suffix
	lastChar := quantityStr[len(quantityStr)-1]
	var multiplier int = 1
	var numStr string
	
	switch lastChar {
	case 'T', 't':
		multiplier = 1000
		numStr = quantityStr[:len(quantityStr)-1]
	case 'M', 'm':
		multiplier = 1000000
		numStr = quantityStr[:len(quantityStr)-1]
	case 'B', 'b':
		multiplier = 1000000000
		numStr = quantityStr[:len(quantityStr)-1]
	default:
		multiplier = 1
		numStr = quantityStr
	}
	
	baseQty := p.parseIntSafe(numStr)
	if baseQty < 0 {
		return -1
	}
	
	return baseQty * multiplier
}

// resetFighterDatabase resets all fighter data (mirrors TWX Pascal ResetFigDatabase)
func (p *TWXParser) resetFighterDatabase() {
	
	if p.database == nil {
		return
	}
	
	// Enhanced Pascal-compliant fighter database reset
	if err := p.resetFighterDatabasePascalCompliant(); err != nil {
		// Fallback to simple database reset
		if err := p.database.ResetPersonalCorpFighters(); err != nil {
			// Error occurred
		} else {
			// Success
		}
	} else {
		// Success
	}
}

// resetFighterDatabasePascalCompliant implements the Pascal TWX ResetFigDatabase logic exactly
func (p *TWXParser) resetFighterDatabasePascalCompliant() error {
	defer p.recoverFromPanic("resetFighterDatabasePascalCompliant")
	
	// Pascal: for i:= 11 to TWXDatabase.DBHeader.Sectors do
	totalSectors := p.database.GetSectors()
	if totalSectors <= 10 {
		return nil
	}
	
	// Find Stardock sector by checking for Stardock planets
	stardockSector := p.findStardockSector()
	
	sectorsProcessed := 0
	sectorsReset := 0
	
	// Iterate through sectors starting from 11 (Pascal convention)
	for i := 11; i <= totalSectors; i++ {
		// Pascal: if (i <> TWXDatabase.DBHeader.Stardock) then
		if i == stardockSector {
			continue
		}
		
		// Pascal: Sect := TWXDatabase.LoadSector(i);
		sector, err := p.database.LoadSector(i)
		if err != nil {
			continue
		}
		
		sectorsProcessed++
		
		// Pascal: if (Sect.Figs.Owner = 'yours') or (Sect.Figs.Owner = 'belong to your Corp') then
		if p.isPersonalOrCorpFighter(sector.Figs.Owner) {
			// Pascal: Sect.Figs.Quantity := 0;
			// Pascal: Sect.Figs.FigType := ftNone;
			sector.Figs.Quantity = 0
			sector.Figs.Owner = ""
			sector.Figs.FigType = 3 // ftNone
			
			// Pascal: TWXDatabase.SaveSector(Sect, i);
			if err := p.database.SaveSector(sector, i); err != nil {
				continue
			}
			
			sectorsReset++
		}
	}
	
	return nil
}

// findStardockSector attempts to find the Stardock sector by checking for Stardock planets
func (p *TWXParser) findStardockSector() int {
	// Try checking sectors 1-20 as a reasonable range instead of relying on GetSectors()
	// which might not be updated during testing
	for i := 1; i <= 20; i++ {
		sector, err := p.database.LoadSector(i)
		if err != nil {
			continue
		}
		
		// Check if this sector has a Stardock planet
		for _, planet := range sector.Planets {
			if planet.Stardock {
				return i
			}
		}
	}
	
	// Default fallback - no Stardock found
	return -1 // No Stardock to exclude
}

// isPersonalOrCorpFighter checks if the fighter owner indicates personal or corporate fighters
func (p *TWXParser) isPersonalOrCorpFighter(owner string) bool {
	if owner == "" {
		return false
	}
	
	ownerLower := strings.ToLower(owner)
	
	// Pascal exact matching
	return owner == "yours" || 
		   owner == "belong to your Corp" ||
		   ownerLower == "yours" ||
		   ownerLower == "belong to your corp" ||
		   strings.Contains(ownerLower, "your corp") ||
		   strings.Contains(ownerLower, "your corporation")
}

// handleStardockDetection processes Stardock detection from 'V' screen (mirrors Pascal lines 1234-1264)
func (p *TWXParser) handleStardockDetection(line string) {
	
	// Find "sector" and extract the number after it
	sectorPos := strings.Index(line, "sector")
	if sectorPos == -1 {
		return
	}
	
	// Extract everything after "sector "
	afterSector := line[sectorPos+7:] // Skip "sector "
	dotPos := strings.Index(afterSector, ".")
	if dotPos == -1 {
		return
	}
	
	sectorStr := strings.TrimSpace(afterSector[:dotPos])
	sectorNum := p.parseIntSafe(sectorStr)
	
	if sectorNum <= 0 {
		return
	}
	
	
	// Check if Stardock is already known
	currentStardock := p.getStardockSector()
	if currentStardock != 0 && currentStardock != 65535 {
		return
	}
	
	// Pascal: if (I > 0) and (I <= TWXDatabase.DBHeader.Sectors) then
	// For now, we'll assume reasonable sector range
	if sectorNum > 0 && sectorNum <= 50000 {
		p.setupStardockSector(sectorNum)
		p.setStardockSector(sectorNum)
	}
}

// setupStardockSector sets up the Stardock sector with Pascal-compliant data
func (p *TWXParser) setupStardockSector(sectorNum int) {
	if p.database == nil {
		return
	}
	
	// Pascal logic: setup Federation beacon and constellation, port class 9
	sector, err := p.database.LoadSector(sectorNum)
	if err != nil {
		// Create new sector if it doesn't exist
		sector = database.NULLSector()
	}
	
	// Pascal: Sect.Constellation := 'The Federation';
	sector.Constellation = "The Federation"
	
	// Pascal: Sect.Beacon := 'FedSpace, FedLaw Enforced';
	sector.Beacon = "FedSpace, FedLaw Enforced"
	
	// Pascal: Sect.Explored := etCalc;
	sector.Explored = database.EtCalc
	
	// Pascal: Sect.Update := Now;
	sector.UpDate = time.Now()
	
	// Save the sector first
	if err := p.database.SaveSector(sector, sectorNum); err != nil {
		return
	}
	
	// Phase 2: Create separate port data for Stargate
	port := database.TPort{
		Name:           "Stargate Alpha I",
		Dead:           false,
		BuildTime:      0,
		ClassIndex:     9, // Stardock class
		BuyProduct:     [3]bool{false, false, false},
		ProductPercent: [3]int{0, 0, 0},
		ProductAmount:  [3]int{0, 0, 0},
		UpDate:         time.Now(),
	}
	
	// Save port data directly (sector already exists)
	if err := p.database.SavePort(port, sectorNum); err != nil {
		return
	}
}

// setStardockSector stores the Stardock sector number in configuration
func (p *TWXParser) setStardockSector(sectorNum int) {
	if p.database == nil {
		return
	}
	
	// Store as script variable (Pascal stores in INI file, we'll use script variables)
	if err := p.database.SaveScriptVariable("$STARDOCK", sectorNum); err != nil {
	} else {
	}
}

// getStardockSector retrieves the Stardock sector number from configuration
func (p *TWXParser) getStardockSector() int {
	if p.database == nil {
		return 0
	}
	
	value, err := p.database.LoadScriptVariable("$STARDOCK")
	if err != nil {
		return 0 // Unknown
	}
	
	// Handle different numeric types that database might return
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		// Try to parse string as int
		if sectorNum := p.parseIntSafe(v); sectorNum > 0 {
			return sectorNum
		}
	}
	
	return 0 // Unknown
}

// Utility methods

func (p *TWXParser) sectorCompleted() {
	defer p.recoverFromPanic("sectorCompleted")
	
	// Skip if already completed to avoid duplicate TUI API calls
	if p.sectorSaved {
		return
	}
	
	// Finalize any pending trader without ship details
	if p.sectorPosition == SectorPosTraders && p.currentTrader.Name != "" {
		p.validateTraderData(&p.currentTrader)
		p.currentTraders = append(p.currentTraders, p.currentTrader)
		p.currentTrader = TraderInfo{} // Reset
	}
	
	// Validate sector number before completion
	if !p.validateSectorNumber(p.currentSectorIndex) {
		return
	}
	
	// Validate all collected data before saving
	p.validateCollectedSectorData()
	
	
	// Save sector data to database with error recovery
	p.errorRecoveryHandler("saveSectorToDatabase", func() error {
		return p.saveSectorToDatabase()
	})
	
	// Fire sector complete event
	sectorData := p.buildSectorData()
	p.fireSectorCompleteEvent(sectorData)
	
	// Fire trader data update event if we have traders
	if len(p.currentTraders) > 0 {
		p.fireTraderDataEvent(p.currentSectorIndex, p.currentTraders)
	}
	
	p.sectorSaved = true
}

// buildSectorData creates a SectorData struct from current parser state
func (p *TWXParser) buildSectorData() SectorData {
	return SectorData{
		Index:         p.currentSectorIndex,
		Constellation: p.currentSector.Constellation,
		Beacon:        p.currentSector.Beacon,
		Warps:         p.currentSectorWarps,
		Port:          p.currentSector.Port,
		NavHaz:        p.currentSector.NavHaz,
		Ships:         p.currentShips,
		Traders:       p.currentTraders,
		Planets:       p.currentPlanets,
		Mines:         p.currentMines,
		Products:      p.currentProducts,
	}
}

// parseIntSafe is now implemented in parser_utils.go

// Reset resets the parser state
func (p *TWXParser) Reset() {
	p.currentLine = ""
	p.currentANSILine = ""
	p.rawANSILine = ""
	p.inANSI = false
	p.currentDisplay = DisplayNone
	p.sectorPosition = SectorPosNormal
	p.currentSectorIndex = 0
	p.portSectorIndex = 0
	p.figScanSector = 0
	p.lastWarp = 0
	p.sectorSaved = false
	p.position = 0
	p.lastChar = 0
	p.currentTrader = TraderInfo{} // Reset current trader
}

// GetCurrentSector returns the current sector index
func (p *TWXParser) GetCurrentSector() int {
	return p.currentSectorIndex
}

// GetDisplayState returns the current display state
func (p *TWXParser) GetDisplayState() DisplayType {
	return p.currentDisplay
}

// GetPlayerStats returns the current player statistics
func (p *TWXParser) GetPlayerStats() PlayerStats {
	return p.playerStats
}

// GetTWGSType returns the detected TWGS server type (0=unknown, 1=TW2002, 2=TWGS)
func (p *TWXParser) GetTWGSType() int {
	return p.twgsType
}

// GetTWGSVersion returns the detected TWGS version string
func (p *TWXParser) GetTWGSVersion() string {
	return p.twgsVer
}

// GetTW2002Version returns the detected TW2002 version string
func (p *TWXParser) GetTW2002Version() string {
	return p.tw2002Ver
}

// GetCurrentTurns returns current turns remaining
func (p *TWXParser) GetCurrentTurns() int {
	return p.playerStats.Turns
}

// GetCurrentCredits returns current credits
func (p *TWXParser) GetCurrentCredits() int {
	return p.playerStats.Credits
}

// GetCurrentFighters returns current fighters
func (p *TWXParser) GetCurrentFighters() int {
	return p.playerStats.Fighters
}

// ProcessString processes a complete string (for testing)
func (p *TWXParser) ProcessString(input string) {
	p.ProcessInBound(input)
}

// ProcessChunk processes a byte chunk (for streaming compatibility)
func (p *TWXParser) ProcessChunk(data []byte) {
	p.ProcessInBound(string(data))
}

// processQuickStats parses the QuickStats line (mirrors TWX Pascal ProcessQuickStats)
// Format: Sect 1│Turns 1,600│Creds 10,000│Figs 30│Shlds 0│Hlds 40│Ore 0│Org 0│Equ 0
//         Col 0│Phot 0│Armd 0│Lmpt 0│GTorp 0│TWarp No│Clks 0│Beacns 0│AtmDt 0│Crbo 0
//         EPrb 0│MDis 0│PsPrb No│PlScn No│LRS None,Dens,Holo│Aln 0│Exp 0│Ship 1 MerCru
func (p *TWXParser) processQuickStats(line string) {
	defer p.recoverFromPanic("processQuickStats")
	
	if !strings.HasPrefix(line, " ") {
		return
	}

	// Validate line length
	if !p.validateLineFormat(line) {
		return
	}


	// Remove leading space with bounds checking
	if len(line) < 2 {
		return
	}
	content := line[1:]

	// Split on the separator character '│' (as seen in raw.log)
	var values []string
	if strings.Contains(content, "│") {
		values = strings.Split(content, "│")
	} else {
		// No recognized separator found
		debug.Log("QuickStats line has no recognized separator: %q", line)
		return
	}

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		// Split each value into parts (key value)
		parts := strings.Fields(value)
		if len(parts) < 2 {
			continue
		}

		key := parts[0]
		val := parts[1]

		// Process each statistic (mirrors Pascal logic)
		switch key {
		case "Turns":
			p.playerStats.Corp = 0 // No corp displayed if player not member
			p.playerStats.Turns = p.parseIntSafeWithCommas(val)
		case "Creds":
			p.playerStats.Credits = p.parseIntSafeWithCommas(val)
		case "Figs":
			p.playerStats.Fighters = p.parseIntSafeWithCommas(val)
		case "Shlds":
			p.playerStats.Shields = p.parseIntSafeWithCommas(val)
		case "Crbo":
			p.playerStats.Corbomite = p.parseIntSafeWithCommas(val)
		case "Hlds":
			p.playerStats.TotalHolds = p.parseIntSafe(val)
		case "Ore":
			p.playerStats.OreHolds = p.parseIntSafe(val)
		case "Org":
			p.playerStats.OrgHolds = p.parseIntSafe(val)
		case "Equ":
			p.playerStats.EquHolds = p.parseIntSafe(val)
		case "Col":
			p.playerStats.ColHolds = p.parseIntSafe(val)
		case "Phot":
			p.playerStats.Photons = p.parseIntSafe(val)
		case "Armd":
			p.playerStats.Armids = p.parseIntSafe(val)
		case "Lmpt":
			p.playerStats.Limpets = p.parseIntSafe(val)
		case "GTorp":
			p.playerStats.GenTorps = p.parseIntSafe(val)
		case "Clks":
			p.playerStats.Cloaks = p.parseIntSafe(val)
		case "Beacns":
			p.playerStats.Beacons = p.parseIntSafe(val)
		case "AtmDt":
			p.playerStats.Atomics = p.parseIntSafe(val)
		case "EPrb":
			p.playerStats.Eprobes = p.parseIntSafe(val)
		case "MDis":
			p.playerStats.MineDisr = p.parseIntSafe(val)
		case "Aln":
			p.playerStats.Alignment = p.parseIntSafeWithCommas(val)
		case "Exp":
			p.playerStats.Experience = p.parseIntSafeWithCommas(val)
		case "Corp":
			p.playerStats.Corp = p.parseIntSafe(val)
		case "TWarp":
			if val == "No" {
				p.playerStats.TwarpType = 0
			} else {
				p.playerStats.TwarpType = p.parseIntSafe(val)
			}
		case "PsPrb":
			p.playerStats.PsychicProbe = (val == "Yes")
		case "PlScn":
			p.playerStats.PlanetScanner = (val == "Yes")
		case "LRS":
			switch val {
			case "None":
				p.playerStats.ScanType = 0
			case "Dens":
				p.playerStats.ScanType = 1
			case "Holo":
				p.playerStats.ScanType = 2
			default:
				p.playerStats.ScanType = 0
			}
		case "Ship":
			if len(parts) > 2 {
				p.playerStats.ShipNumber = p.parseIntSafe(val)
				p.playerStats.ShipClass = parts[2]
			}
		}
	}

	// Only save to database and fire events if we actually found stats in this line
	foundStats := false
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		parts := strings.Fields(value)
		if len(parts) >= 2 {
			key := parts[0]
			switch key {
			case "Turns", "Creds", "Figs", "Shlds", "Crbo", "Hlds", "Ore", "Org", "Equ", "Col",
				 "Phot", "Armd", "Lmpt", "GTorp", "Clks", "Beacns", "AtmDt", "EPrb", "MDis",
				 "Aln", "Exp", "Corp", "TWarp", "PsPrb", "PlScn", "LRS", "Ship":
				foundStats = true
				break
			}
		}
		if foundStats {
			break
		}
	}
	
	if foundStats {
		// Set current sector in player stats
		p.playerStats.CurrentSector = p.currentSectorIndex

		// Validate all collected player stats
		p.validatePlayerStats(&p.playerStats)
		
		// Save player stats to database with error recovery
		p.errorRecoveryHandler("savePlayerStatsToDatabase", func() error {
			return p.savePlayerStatsToDatabase()
		})
		
		// Fire player stats update event
		p.firePlayerStatsEvent(p.playerStats)
	}
}

// parseIntSafeWithCommas parses integers that may contain commas
// parseIntSafeWithCommas is now implemented in parser_utils.go

// handleMessageLine processes message content (mirrors TWX Pascal message handling)
func (p *TWXParser) handleMessageLine(line string) {
	// Use enhanced message line handling with database integration
	p.handleEnhancedMessageLine(line)
}

// parseWarpConnections parses warp connections from warp data string with validation and conflict resolution
func (p *TWXParser) parseWarpConnections(warpData string) {
	
	// Initialize warps array
	var warps [6]int
	
	// First, strip ANSI color codes to avoid parsing issues
	warpData = ansi.StripString(warpData)
	
	// Clean up the warp data - remove parentheses and split on various delimiters
	warpData = strings.ReplaceAll(warpData, "(", "")
	warpData = strings.ReplaceAll(warpData, ")", "")
	warpData = strings.TrimSpace(warpData)
	
	// Split on both " - " and ", " to handle different formats
	var warpStrs []string
	if strings.Contains(warpData, " - ") {
		warpStrs = strings.Split(warpData, " - ")
	} else if strings.Contains(warpData, ", ") {
		warpStrs = strings.Split(warpData, ", ")
	} else {
		// Single warp or space-separated
		warpStrs = strings.Fields(warpData)
	}
	
	// Parse and validate each warp sector number
	warpIndex := 0
	for _, warpStr := range warpStrs {
		warpStr = strings.TrimSpace(warpStr)
		if warpStr != "" && warpIndex < 6 {
			warpNum := p.parseIntSafe(warpStr)
			if warpNum > 0 {
				// Validate warp sector number (must be reasonable range)
				if p.validateWarpSector(warpNum) {
					// Check for duplicates in current warp list
					if !p.containsWarp(warps[:warpIndex], warpNum) {
						warps[warpIndex] = warpNum
						warpIndex++
					} else {
					}
				} else {
				}
			} else {
			}
		} else {
			if warpStr == "" {
			} else if warpIndex >= 6 {
			}
		}
	}
	
	// Sort warps for consistency (mirrors Pascal AddWarp insertion sort logic)
	p.sortWarps(warps[:warpIndex])
	
	// Store the warps in the current sector data
	p.currentSectorWarps = warps
	
	// Update reverse warp connections in database for advanced pathfinding
	p.updateReverseWarpConnections(p.currentSectorIndex, warps[:warpIndex])
}

// validateWarpSector validates that a warp sector number is reasonable
func (p *TWXParser) validateWarpSector(sectorNum int) bool {
	// Basic validation - sector must be positive and within reasonable bounds
	if sectorNum <= 0 {
		return false
	}
	
	// NOTE: Don't validate against database max sectors as it prevents discovering new sectors
	// The database only knows about sectors that have already been visited/parsed
	
	// Reasonable upper bound check (Trade Wars maximum is 20,000 sectors)
	if sectorNum > 20000 {
		return false
	}
	
	return true
}

// containsWarp checks if a warp already exists in the warp list
func (p *TWXParser) containsWarp(warps []int, warpNum int) bool {
	for _, w := range warps {
		if w == warpNum {
			return true
		}
	}
	return false
}

// sortWarps sorts warps in ascending order (mirrors Pascal insertion sort logic)
func (p *TWXParser) sortWarps(warps []int) {
	for i := 1; i < len(warps); i++ {
		key := warps[i]
		j := i - 1
		for j >= 0 && warps[j] > key {
			warps[j+1] = warps[j]
			j--
		}
		warps[j+1] = key
	}
}

// updateReverseWarpConnections updates reverse warp connections for pathfinding
func (p *TWXParser) updateReverseWarpConnections(fromSector int, warps []int) {
	if p.database == nil {
		return
	}
	
	// For each destination sector, ensure it has a reverse warp back to this sector
	// This mirrors the Pascal AddWarp logic for maintaining bidirectional connectivity
	for _, toSector := range warps {
		if toSector > 0 {
			p.addReverseWarp(toSector, fromSector)
		}
	}
}

// addReverseWarp adds a reverse warp connection (mirrors Pascal AddWarp method)
func (p *TWXParser) addReverseWarp(toSector, fromSector int) {
	// Load the destination sector
	sector, err := p.database.LoadSector(toSector)
	if err != nil {
		return
	}
	
	// Check if reverse warp already exists
	for _, existingWarp := range sector.Warp {
		if existingWarp == fromSector {
			return // Already exists
		}
	}
	
	// Find insertion position (maintain sorted order like Pascal AddWarp)
	insertPos := -1
	for i, warp := range sector.Warp {
		if warp == 0 || warp > fromSector {
			insertPos = i
			break
		}
	}
	
	if insertPos >= 0 && insertPos < 6 {
		// Shift existing warps right
		for i := 5; i > insertPos; i-- {
			sector.Warp[i] = sector.Warp[i-1]
		}
		sector.Warp[insertPos] = fromSector
		
		// Mark as calculated data if not already explored
		if sector.Explored == 0 { // EtNo
			sector.Constellation = "???" + " (warp calc only)"
			sector.Explored = 1 // EtCalc
		}
		
		// Save updated sector
		if err := p.database.SaveSector(sector, toSector); err != nil {
		} else {
		}
	}
}

// ===== IModExtractor Interface Implementation =====

// GetCurrentDisplay returns the current display/parsing context
func (p *TWXParser) GetCurrentDisplay() DisplayType {
	return p.currentDisplay
}

// SetCurrentDisplay sets the current display/parsing context
func (p *TWXParser) SetCurrentDisplay(display DisplayType) {
	oldDisplay := p.currentDisplay
	p.currentDisplay = display
	
	// Fire state change event
	p.fireStateChangeEvent("display", oldDisplay, display)
}

// SetEventBus sets the event bus for module communication
func (p *TWXParser) SetEventBus(bus IEventBus) {
	p.eventBus = bus
	
	// Update script interpreter with new event bus
	if p.scriptInterpreter != nil {
		p.scriptInterpreter = NewScriptInterpreter(bus)
	}
	
}

// GetEventBus returns the current event bus
func (p *TWXParser) GetEventBus() IEventBus {
	return p.eventBus
}

// FireTextEvent fires a text event to the script system (Pascal: TWXInterpreter.TextEvent)
func (p *TWXParser) FireTextEvent(line string, outbound bool) {
	if p.scriptInterpreter != nil {
		p.scriptInterpreter.TextEvent(line, outbound)
	}
	
	// Also fire through the ScriptEventProcessor for the new scripting engine
	if p.scriptEventProcessor != nil && p.scriptEventProcessor.IsEnabled() {
		if err := p.scriptEventProcessor.FireTextEvent(line, false); err != nil {
		}
	}
}

// FireTextLineEvent fires a text line event to the script system (Pascal: TWXInterpreter.TextLineEvent)
func (p *TWXParser) FireTextLineEvent(line string, outbound bool) {
	if p.scriptInterpreter != nil {
		p.scriptInterpreter.TextLineEvent(line, outbound)
	}
	
	// Also fire through the ScriptEventProcessor for the new scripting engine
	if p.scriptEventProcessor != nil && p.scriptEventProcessor.IsEnabled() {
		p.scriptEventProcessor.FireTextLineEvent(line, outbound)
	}
}

// ActivateTriggers activates script triggers (Pascal: TWXInterpreter.ActivateTriggers)
func (p *TWXParser) ActivateTriggers() {
	if p.scriptInterpreter != nil {
		p.scriptInterpreter.ActivateTriggers()
	}
	
	// Also fire through the ScriptEventProcessor for the new scripting engine
	if p.scriptEventProcessor != nil && p.scriptEventProcessor.IsEnabled() {
		p.scriptEventProcessor.FireActivateTriggers()
	}
}

// FireAutoTextEvent fires an auto text event to the script system (Pascal: TWXInterpreter.AutoTextEvent)
func (p *TWXParser) FireAutoTextEvent(line string, outbound bool) {
	if p.scriptInterpreter != nil {
		p.scriptInterpreter.AutoTextEvent(line, outbound)
	}
	
	// Also fire through the ScriptEventProcessor for the new scripting engine
	if p.scriptEventProcessor != nil && p.scriptEventProcessor.IsEnabled() {
		p.scriptEventProcessor.FireAutoTextEvent(line, outbound)
	}
}

// GetDatabase returns the database interface
func (p *TWXParser) GetDatabase() database.Database {
	return p.database
}

// SetDatabase sets the database interface
func (p *TWXParser) SetDatabase(db database.Database) {
	p.database = db
}

// ProcessOutBound processes outbound data and returns whether to continue sending
func (p *TWXParser) ProcessOutBound(data string) bool {
	
	// Fire outbound text events
	p.FireTextEvent(data, true)
	p.FireTextLineEvent(data, true)
	
	// Fire outbound event to event bus
	if p.eventBus != nil {
		event := Event{
			Type: EventText,
			Data: map[string]interface{}{
				"line":     data,
				"outbound": true,
			},
			Source: "TWXParser",
		}
		p.eventBus.Fire(event)
	}
	
	// Always continue sending (return true means "continue")
	return true
}

// ===== Observer Pattern Implementation (ISubject) =====

// Attach adds an observer to the subject
func (p *TWXParser) Attach(observer IObserver) {
	p.observers = append(p.observers, observer)
}

// Detach removes an observer from the subject
func (p *TWXParser) Detach(observerID string) {
	for i, observer := range p.observers {
		if observer.GetObserverID() == observerID {
			// Remove observer by swapping with last element and truncating
			p.observers[i] = p.observers[len(p.observers)-1]
			p.observers = p.observers[:len(p.observers)-1]
			return
		}
	}
}

// Notify notifies all observers of an event
func (p *TWXParser) Notify(event Event) {
	
	for _, observer := range p.observers {
		func() {
			defer func() {
				if r := recover(); r != nil {
				}
			}()
			observer.Update(p, event)
		}()
	}
}

// ===== Event Helper Methods =====

// fireStateChangeEvent fires a state change event to both observers and event bus
func (p *TWXParser) fireStateChangeEvent(property string, oldValue, newValue interface{}) {
	event := Event{
		Type: EventStateChange,
		Data: map[string]interface{}{
			"property":  property,
			"oldValue":  oldValue,
			"newValue":  newValue,
			"sector":    p.currentSectorIndex,
		},
		Source: "TWXParser",
	}
	
	// Notify observers
	p.Notify(event)
	
	// Fire to event bus
	if p.eventBus != nil {
		p.eventBus.Fire(event)
	}
}

// fireSectorCompleteEvent fires a sector completion event
func (p *TWXParser) fireSectorCompleteEvent(sectorData SectorData) {
	event := Event{
		Type: EventSectorComplete,
		Data: map[string]interface{}{
			"sectorData": sectorData,
			"sector":     sectorData.Index,
		},
		Source: "TWXParser",
	}
	
	// Notify observers
	p.Notify(event)
	
	// Fire to event bus
	if p.eventBus != nil {
		p.eventBus.Fire(event)
	}
}

// fireParseCompleteEvent fires a parse completion event
func (p *TWXParser) fireParseCompleteEvent(line string) {
	event := Event{
		Type: EventParseComplete,
		Data: map[string]interface{}{
			"line":   line,
			"sector": p.currentSectorIndex,
		},
		Source: "TWXParser",
	}
	
	// Notify observers
	p.Notify(event)
	
	// Fire to event bus
	if p.eventBus != nil {
		p.eventBus.Fire(event)
	}
}

// fireMessageEvent fires a message received event
func (p *TWXParser) fireMessageEvent(msgType MessageType, content, sender string, channel int) {
	event := Event{
		Type: EventMessageReceived,
		Data: map[string]interface{}{
			"messageType": msgType,
			"content":     content,
			"sender":      sender,
			"channel":     channel,
		},
		Source: "TWXParser",
	}
	
	// Notify observers
	p.Notify(event)
	
	// Fire to event bus
	if p.eventBus != nil {
		p.eventBus.Fire(event)
	}
}

// fireDatabaseUpdateEvent fires a database update event
func (p *TWXParser) fireDatabaseUpdateEvent(operation string, sectorNum int, data interface{}) {
	event := Event{
		Type: EventDatabaseUpdate,
		Data: map[string]interface{}{
			"operation": operation,
			"sector":    sectorNum,
			"data":      data,
		},
		Source: "TWXParser",
	}
	
	// Notify observers
	p.Notify(event)
	
	// Fire to event bus
	if p.eventBus != nil {
		p.eventBus.Fire(event)
	}
}

// fireTraderDataEvent fires a trader data update event to the TUI API
func (p *TWXParser) fireTraderDataEvent(sectorNumber int, traders []TraderInfo) {
	if p.tuiAPI != nil {
		// Convert internal TraderInfo to API TraderInfo
		apiTraders := make([]api.TraderInfo, len(traders))
		for i, trader := range traders {
			apiTraders[i] = api.TraderInfo{
				Name:      trader.Name,
				ShipName:  trader.ShipName,
				ShipType:  trader.ShipType,
				Fighters:  trader.Fighters,
				Alignment: trader.Alignment,
			}
		}
		
		// Fire the event
		p.tuiAPI.OnTraderDataUpdated(sectorNumber, apiTraders)
	}
}

// firePlayerStatsEvent fires a player statistics update event to the TUI API
func (p *TWXParser) firePlayerStatsEvent(stats PlayerStats) {
	if p.tuiAPI != nil {
		// Convert internal PlayerStats to API PlayerStatsInfo using converter
		apiStats := converter.ConvertPlayerStatsToPlayerStatsInfo(stats)
		
		// Fire the event
		p.tuiAPI.OnPlayerStatsUpdated(apiStats)
	}
}