package streaming

import (
	"fmt"
	"strings"
	"time"
	"twist/internal/ansi"
	"twist/internal/api"
	"twist/internal/log"
	"twist/internal/proxy/database"
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

// OrderedPatternHandler holds a pattern and its handler in order
type OrderedPatternHandler struct {
	Pattern string
	Handler PatternHandler
}

// PlayerStats type removed - using api.PlayerStatsInfo directly (straight-sql pattern)

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
	Name     string
	Owner    string
	Fighters int
	Citadel  bool
	Stardock bool
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
	Type     ProductType
	Quantity int
	Percent  int
	Buying   bool
	Selling  bool
	Status   string // "Buying", "Selling", etc.
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
	Index         int
	Constellation string
	Beacon        string
	Warps         [6]int // Warp destinations (1-6)
	Port          PortData
	Density       int
	NavHaz        int
	Anomaly       bool
	Explored      bool
	// Detailed sector data
	Ships    []ShipInfo
	Traders  []TraderInfo
	Planets  []PlanetInfo
	Mines    []MineInfo
	Products []ProductInfo
}

// CurrentSector holds current sector being parsed
type CurrentSector struct {
	Index         int
	Constellation string
	Beacon        string
	Port          PortData // Current port data for build time tracking
	NavHaz        int      // Navigation hazard percentage
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
	currentDisplay          DisplayType
	sectorPosition          SectorPosition
	currentSectorIndex      int
	portSectorIndex         int
	currentPortName         string      // Name of the current port being parsed
	currentTradingCommodity ProductType // Currently trading commodity (port-specific state)
	figScanSector           int
	lastWarp                int
	sectorSaved             bool
	probeMode               bool         // True when parsing probe-discovered sectors, prevents TUI events
	probeDiscoveredSectors  map[int]bool // Track sectors discovered by probes to suppress TUI events
	menuKey                 rune

	// Phase 1: Straight SQL player stats tracker (replaces intermediate objects)
	playerStatsTracker *PlayerStatsTracker

	// Phase 2: Straight SQL sector and collection trackers (replace intermediate objects)
	sectorTracker     *SectorTracker
	sectorCollections *SectorCollections

	// Phase 3: Straight SQL port tracker (replace intermediate objects)
	portTracker *PortTracker

	// Info display parsing state
	infoDisplay InfoDisplay

	// Quick stats parsing state
	quickStatsDisplay QuickStatsDisplay

	// Current game data
	currentSectorWarps [6]int // Temporary storage for parsed warps
	currentMessage     string
	currentChannel     int // Current radio channel for message context
	twgsVer            string
	tw2002Ver          string
	twgsType           int

	// Message history
	messageHistory []MessageHistory
	maxHistorySize int

	// Temporary storage for trader being parsed (minimal intermediate data)
	currentTrader TraderInfo

	// Pattern handlers (ordered slice to ensure deterministic processing)
	handlers []OrderedPatternHandler

	// Position tracking
	position int64
	lastChar rune

	// Database integration
	getDatabaseFunc func() database.Database

	// TUI API integration
	tuiAPI api.TuiAPI

	// Script integration (mirrors Pascal TWXInterpreter integration)
	scriptEventProcessor *ScriptEventProcessor

	// Observer pattern and event system (Pascal: TTWXModule integration)
	observers         []IObserver
	eventBus          IEventBus
	scriptInterpreter IScriptInterpreter
}

// GetDatabase returns the database instance, panicking if it's nil
func (p *TWXParser) GetDatabase() database.Database {
	if p.getDatabaseFunc == nil {
		log.Error("CRITICAL: getDatabaseFunc is nil in TWXParser", "parser", p)
		panic("getDatabaseFunc is nil - proxy initialization failed")
	}
	db := p.getDatabaseFunc()
	if db == nil {
		log.Error("CRITICAL: Database returned nil from getDatabaseFunc", "func", p.getDatabaseFunc)
		panic("Database is nil - game detector should have set this up")
	}
	log.Debug("Database retrieved successfully", "db", db)
	return db
}

// NewTWXParser creates a new TWX-style parser with database accessor and TUI API
func NewTWXParser(getDatabaseFunc func() database.Database, tuiAPI api.TuiAPI) *TWXParser {
	parser := &TWXParser{
		currentLine:            "",
		currentANSILine:        "",
		rawANSILine:            "",
		inANSI:                 false,
		ansiStripper:           ansi.NewStreamingStripper(),
		currentDisplay:         DisplayNone,
		sectorPosition:         SectorPosNormal,
		lastWarp:               0,
		sectorSaved:            false,
		probeDiscoveredSectors: make(map[int]bool),
		menuKey:                '$',
		handlers:               make([]OrderedPatternHandler, 0),
		position:               0,
		lastChar:               0,
		maxHistorySize:         1000,
		getDatabaseFunc:        getDatabaseFunc, // Database accessor
		tuiAPI:                 tuiAPI,          // Optional TUI API
		// Version detection fields
		twgsType:  0,
		twgsVer:   "",
		tw2002Ver: "",
		// Initialize data structures
		messageHistory: make([]MessageHistory, 0),
		// Initialize script integration (disabled by default)
		scriptEventProcessor: NewScriptEventProcessor(nil),
		// Initialize observer pattern
		observers: make([]IObserver, 0),
	}

	// Initialize event bus and script interpreter
	parser.eventBus = NewEventBus()
	parser.scriptInterpreter = NewScriptInterpreter(parser.eventBus)

	// Initialize info display
	parser.initInfoDisplay()

	// Initialize quick stats display
	parser.initQuickStatsDisplay()

	// Set up default handlers
	parser.setupDefaultHandlers()
	parser.setupInfoHandlers()
	parser.setupQuickStatsHandlers()
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
	p.handlers = append(p.handlers, OrderedPatternHandler{
		Pattern: pattern,
		Handler: handler,
	})
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
	p.AddHandler("Sector  :", p.handleSectorStart) // Handle variant without space before colon
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

		// Process the complete line WITHOUT error recovery to see actual error
		// Validate line format before processing
		if p.validateLineFormat(completeLine) {
			p.processLine(completeLine)
			// Fire parse complete event
			p.fireParseCompleteEvent(completeLine)
		}

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
		// Update CURRENTLINE system constant for partial data (prompts) to match TWX Pascal behavior
		// In Pascal TWX: CurrentLine := Line (line 1499), and SCCurrentLine returns TWXExtractor.CurrentLine
		if strings.TrimSpace(p.currentLine) != "" {
			p.UpdateCurrentLine(p.currentLine)
		}

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

	// Complete any pending info display
	if p.infoDisplay.Active {
		log.Info("INFO_PARSER: Finalize() completing active info display")
		p.completeInfoDisplay()
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

	// Pascal TWX pattern matching - check independent patterns that can coexist
	// Based on Process.pas lines 317 and 1338

	// Check command prompt first (Pascal: Copy(Line, 1, 12) = 'Command [TL=')
	if strings.HasPrefix(line, "Command [TL=") {
		p.handleCommandPrompt(line)
	}

	// Check density scanner independently (Pascal: Copy(Line, 27, 16) = 'Relative Density')
	if strings.Contains(line, "Relative Density") {
		p.currentDisplay = DisplayDensity
		// Pascal TWX returns early after setting mode, so we do the same
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
		// Phase 2: Density data now handled through straight-sql trackers
		p.processDensityLineTracker(line)
	case DisplayFigScan:
		p.processFigScanLine(line)
	default:
		// Check for pattern matches to change state
		p.checkPatterns(line)
	}
	// Check for info display end and quick stats end before other processing
	p.checkInfoDisplayEnd(line)
	p.checkQuickStatsEnd(line)

	// Update CURRENTLINE system constant before firing script events (matches TWX Pascal ProcessInBound sequence)
	// Skip empty lines to prevent overwriting meaningful content
	if strings.TrimSpace(line) != "" {
		p.UpdateCurrentLine(line)
	}

	// Fire TextLineEvent as in Pascal TWX ProcessLine (mirrors Pascal TWXInterpreter.TextLineEvent)
	textLineTriggerFired, err := p.FireTextLineEvent(line, false)
	if err != nil {
		log.Error("Error firing TextLineEvent", "error", err, "line", line)
	}

	// If a TextLineTrigger fired, skip Text event processing (waitfor) - matches TWX behavior
	if !textLineTriggerFired {
		// Always check for prompts (this fires TextEvent)
		p.processPrompt(line)
	}

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
	for _, ph := range p.handlers {
		if strings.HasPrefix(line, ph.Pattern) {
			ph.Handler(line)
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

	for _, ph := range p.handlers {
		if strings.Contains(line, ph.Pattern) {
			ph.Handler(line)
			return
		}
	}
}

// Handler implementations (core TWX parsing logic)

func (p *TWXParser) handleCommandPrompt(line string) {
	// Clear all probe state when we get back to command prompt (back to normal player interaction)
	if p.probeMode || len(p.probeDiscoveredSectors) > 0 {
		p.probeMode = false
		p.probeDiscoveredSectors = make(map[int]bool) // Clear all probe-discovered sectors
		log.Info("PROBE: Cleared all probe state (command prompt) - back to normal player mode")
	}

	// Save current sector if not done already
	if !p.sectorSaved {
		p.sectorCompleted()
	}

	// Extract sector number from "Command [TL=150] (2500) ?" or "Command [TL=00:00:01]:[190] (?=Help)? : "
	// Try square brackets first (for new format), then parentheses (for old format)

	// Check for square brackets pattern: [sector_number]
	openBracket := strings.Index(line, "]:[")
	if openBracket > 0 {
		// Look for closing bracket after the sector number
		startPos := openBracket + 3 // Skip "]:["
		closeBracket := strings.Index(line[startPos:], "]")
		if closeBracket > 0 {
			sectorStr := strings.TrimSpace(line[startPos : startPos+closeBracket])
			if sectorNum := p.parseIntSafe(sectorStr); sectorNum > 0 {
				p.currentSectorIndex = sectorNum
				// Only set lastWarp if we don't already have one (avoid resetting during probe sequence)
				if p.lastWarp == 0 {
					p.lastWarp = sectorNum
				}
				// Ensure the current sector exists in the database
				sectorTracker := NewSectorTracker(sectorNum)
				p.errorRecoveryHandler("ensureCurrentSectorExists", func() error {
					return sectorTracker.Execute(p.GetDatabase().GetDB())
				})

				// Update current sector using straight-sql tracker
				if p.playerStatsTracker == nil {
					p.playerStatsTracker = NewPlayerStatsTracker()
				}
				p.playerStatsTracker.SetCurrentSector(sectorNum)
				p.errorRecoveryHandler("savePlayerStatsToDatabase", func() error {
					return p.playerStatsTracker.Execute(p.GetDatabase().GetDB())
				})

				// Fire OnCurrentSectorChanged event for the player's actual current sector
				// This ensures the TUI is notified when the player returns to their actual location
				if p.tuiAPI != nil {
					freshSectorInfo, err := p.GetDatabase().GetSectorInfo(sectorNum)
					if err == nil {
						log.Info("TWX_PARSER: Firing OnCurrentSectorChanged for player's current sector from command prompt", "sector", sectorNum)
						p.tuiAPI.OnCurrentSectorChanged(freshSectorInfo)
					}
				}
			}
		}
	} else {
		// Fall back to parentheses pattern: (sector_number)
		openParen := strings.Index(line, "(")
		closeParen := strings.Index(line, ")")
		if openParen > 0 && closeParen > openParen {
			sectorStr := strings.TrimSpace(line[openParen+1 : closeParen])
			if sectorNum := p.parseIntSafe(sectorStr); sectorNum > 0 {
				p.currentSectorIndex = sectorNum
				// Only set lastWarp if we don't already have one (avoid resetting during probe sequence)
				if p.lastWarp == 0 {
					p.lastWarp = sectorNum
				}
				// Ensure the current sector exists in the database
				sectorTracker := NewSectorTracker(sectorNum)
				p.errorRecoveryHandler("ensureCurrentSectorExists", func() error {
					return sectorTracker.Execute(p.GetDatabase().GetDB())
				})

				// Fire OnCurrentSectorChanged event for the player's actual current sector
				// This ensures the TUI is notified when the player returns to their actual location
				if p.tuiAPI != nil {
					freshSectorInfo, err := p.GetDatabase().GetSectorInfo(sectorNum)
					if err == nil {
						log.Info("TWX_PARSER: Firing OnCurrentSectorChanged for player's current sector from command prompt", "sector", sectorNum)
						p.tuiAPI.OnCurrentSectorChanged(freshSectorInfo)
					}
				}

				// Update current sector using straight-sql tracker
				if p.playerStatsTracker == nil {
					p.playerStatsTracker = NewPlayerStatsTracker()
				}
				p.playerStatsTracker.SetCurrentSector(sectorNum)
				p.errorRecoveryHandler("savePlayerStatsToDatabase", func() error {
					return p.playerStatsTracker.Execute(p.GetDatabase().GetDB())
				})
			}
		}
	}

	p.currentDisplay = DisplayNone
}

func (p *TWXParser) handleComputerPrompt(line string) {
	log.Info("COMPUTER: handleComputerPrompt called, resetting lastWarp to 0", "previous_lastWarp", p.lastWarp)
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
	log.Info("PROBE: handleProbePrompt called", "line", line)

	// Check if this is "Probe entering sector :" to extract target sector
	if strings.Contains(line, "Probe entering sector :") {
		// Set probe mode to prevent TUI sector change events for probe-discovered sectors
		p.probeMode = true
		log.Info("PROBE: Set probe mode to true")
		// Handle concatenated lines - extract just the probe part
		probeStart := strings.Index(line, "Probe entering sector :")
		if probeStart >= 0 {
			probeLine := line[probeStart:]

			// Extract sector number from "Probe entering sector : 510"
			parts := strings.Fields(probeLine)
			log.Info("PROBE: parsed parts", "parts", parts, "length", len(parts))
			if len(parts) >= 5 {
				targetSectorStr := parts[4]
				log.Info("PROBE: extracted target sector string", "target_sector_str", targetSectorStr)
				if targetSector := p.parseIntSafe(targetSectorStr); targetSector > 0 {
					log.Info("PROBE: parsed target sector", "target_sector", targetSector, "last_warp", p.lastWarp)

					// Mark this sector as discovered by probe to suppress TUI events
					p.probeDiscoveredSectors[targetSector] = true
					log.Info("PROBE: Marked sector as probe-discovered", "sector", targetSector)

					// If we have a previous sector (lastWarp), create a one-way warp connection
					if p.lastWarp > 0 && p.lastWarp != targetSector {
						log.Info("PROBE: Creating warp", "from_sector", p.lastWarp, "to_sector", targetSector)
						p.addProbeWarp(p.lastWarp, targetSector)
					} else {
						log.Info("PROBE: Not creating warp", "last_warp", p.lastWarp, "target_sector", targetSector)
					}
					// Update lastWarp to current target sector for next probe movement
					p.lastWarp = targetSector
					log.Info("PROBE: Updated lastWarp", "new_last_warp", p.lastWarp)
				} else {
					log.Info("PROBE: Failed to parse targetSectorStr as int", "target_sector_str", targetSectorStr)
				}
			} else {
				log.Info("PROBE: Not enough parts in probeLine", "probe_line", probeLine)
			}
		}
	}

	// Check if probe self-destructs to clear probe mode
	if strings.Contains(line, "Probe Self Destructs") {
		// Clear probe mode - we're no longer parsing probe data
		p.probeMode = false
		// Don't clear probeDiscoveredSectors here - we want to continue suppressing TUI events
		// for those sectors until the player actually visits them
		log.Info("PROBE: Set probe mode to false (probe self-destructed)")
	}

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
	// Only trigger if line actually STARTS with ": " (not just contains it)
	if !strings.HasPrefix(line, ": ") {
		return // Don't trigger for lines like "Probe entering sector : 274"
	}

	// Pascal: // begin CIM download
	// Pascal: FCurrentDisplay := dCIM;
	log.Info("CIM: handleCIMPrompt called, resetting lastWarp to 0", "previous_lastWarp", p.lastWarp)
	p.currentDisplay = DisplayCIM
	p.lastWarp = 0
}

func (p *TWXParser) handleSectorStart(line string) {
	log.Info("SECTOR: handleSectorStart called", "line", line, "last_warp", p.lastWarp)

	// Extract sector number first to determine if this is a new sector
	// Format: "Sector  : 1234 in The Sphere"
	parts := strings.Fields(line)
	if len(parts) >= 3 {
		if sectorNum := p.parseIntSafe(parts[2]); sectorNum > 0 {
			log.Info("SECTOR: Parsing sector", "sector", sectorNum, "current_sector", p.currentSectorIndex, "last_warp", p.lastWarp)
			// Only complete previous sector if this is a different sector
			if p.currentSectorIndex != sectorNum && !p.sectorSaved {
				log.Info("SECTOR: Completing previous sector", "previous_sector", p.currentSectorIndex)
				p.sectorCompleted()
			}

			// Always reset sector data when processing a sector visit
			// This ensures that data from previous visits doesn't carry over
			// (including port data that might persist from previous visits)
			p.sectorSaved = false // Reset for any sector visit
			log.Info("SECTOR: About to reset current sector", "last_warp", p.lastWarp)
			p.resetCurrentSector()
			log.Info("SECTOR: After reset current sector", "last_warp", p.lastWarp)
			p.currentSectorIndex = sectorNum

			// Phase 2: Initialize straight-sql trackers for new sector
			if p.sectorTracker != nil && p.sectorTracker.HasUpdates() {
				log.Info("SECTOR: Discarding incomplete sector tracker - new sector detected")
			}
			if p.sectorCollections != nil && p.sectorCollections.HasData() {
				log.Info("SECTOR: Discarding incomplete sector collections - new sector detected")
			}

			// Start new discovered field session
			log.Info("SECTOR_TRACKER_LIFECYCLE: Creating new sectorTracker", "sector", sectorNum, "previous_tracker_nil", p.sectorTracker == nil)
			p.sectorTracker = NewSectorTracker(sectorNum)
			p.sectorCollections = NewSectorCollections(sectorNum)
			p.portTracker = NewPortTracker(sectorNum)

			p.currentDisplay = DisplaySector

			// Handle constellation parsing
			p.handleSectorConstellation(parts)
		}
	}
}

func (p *TWXParser) handlePortDocking(line string) {
	if !p.sectorSaved {
		p.sectorCompleted()
	}
	p.currentDisplay = DisplayPort
	p.portSectorIndex = p.currentSectorIndex
}

func (p *TWXParser) handlePortReport(line string) {
	log.Info("PORT: handlePortReport called", "line", line)

	// Set display mode to Port Commerce Report
	p.currentDisplay = DisplayPortCR
	p.portSectorIndex = p.currentSectorIndex

	// Create or recreate portTracker for this port trading session
	if p.portTracker == nil {
		p.portTracker = NewPortTracker(p.currentSectorIndex)
		log.Info("PORT: Created new portTracker for port trading session", "sector", p.currentSectorIndex)
	}

	// Extract port name from "Commerce report for PORT_NAME:"
	colonPos := strings.Index(line, ":")
	if colonPos > 20 {
		portName := strings.TrimSpace(line[20:colonPos])
		log.Info("PORT: Extracted port name", "port_name", portName)

		// Initialize port data for current sector
		p.initializePortData(portName)
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
	log.Info("WARP: handleWarpLaneStart called, resetting lastWarp to 0", "previous_lastWarp", p.lastWarp)
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

	// Handle end of sections (9th character is ':') - but not for sector data lines
	// This should only trigger for actual section endings, not sector data like "Planets : Terra (L)"
	if len(line) > 9 && line[8] == ':' {
		// Skip this logic if it's a known sector data pattern
		sectorDataPatterns := []string{"Planets : ", "Beacon  : ", "NavHaz  : "}
		isSectorData := false
		for _, pattern := range sectorDataPatterns {
			if strings.HasPrefix(line, pattern) {
				isSectorData = true
				break
			}
		}

		if !isSectorData {
			// Finalize any pending trader without ship details
			if p.sectorPosition == SectorPosTraders && p.currentTrader.Name != "" {
				// Phase 4.5: Traders tracked via collection trackers (no intermediate objects)
				p.currentTrader = TraderInfo{} // Reset
			}
			p.sectorPosition = SectorPosNormal
			return
		}
	}

	// Handle specific sector data patterns directly
	// These were previously handled only by event system, but need direct handling for proper state management
	if strings.HasPrefix(line, "Planets : ") {
		p.handleSectorPlanets(line)
		return
	}
	if strings.HasPrefix(line, "Ports   : ") {
		p.handleSectorPorts(line)
		return
	}
	if strings.HasPrefix(line, "Traders : ") {
		p.handleSectorTraders(line)
		return
	}
	if strings.HasPrefix(line, "Ships   : ") {
		p.handleSectorShips(line)
		return
	}
	if strings.HasPrefix(line, "Mines   : ") {
		p.handleSectorMines(line)
		return
	}
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
	sector, err := p.GetDatabase().LoadSector(sectorNum)
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
		sector.Explored = database.EtCalc // Mark as calculated (not fully explored)
		if sector.Constellation == "" {
			sector.Constellation = "??? (warp calc only)"
		}
	}

	sector.UpDate = time.Now()

	// Save updated sector
	if err := p.GetDatabase().SaveSector(sector, sectorNum); err != nil {
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
	log.Info("PORT: ensureSectorExistsAndSavePort called", "sector", sectorNum, "port_name", port.Name, "class", port.ClassIndex)

	// Save port data
	if err := p.GetDatabase().SavePort(port, sectorNum); err != nil {
		return fmt.Errorf("failed to save port for sector %d: %w", sectorNum, err)
	}

	log.Info("PORT: Successfully saved port data", "sector", sectorNum, "port_name", port.Name, "class", port.ClassIndex)

	// Fire any necessary events (consistent with other database operations)
	// This ensures proper notification flow like other database saves
	return nil
}

// ensureSectorExistsAndSavePortWithVisited marks CIM root sector as visited and saves port data
func (p *TWXParser) ensureSectorExistsAndSavePortWithVisited(port database.TPort, sectorNum int) error {
	// Always ensure sector exists first (required for foreign key constraint)
	sector, err := p.GetDatabase().LoadSector(sectorNum)
	if err != nil {
		// Create minimal sector entry
		sector = database.NULLSector()
		sector.UpDate = time.Now()
	}

	// Mark root sector as calculated since it appears in CIM port report
	// CIM data marks sectors as calculated, not fully explored (matches TWX behavior)
	if sector.Explored == database.EtNo {
		sector.Explored = database.EtCalc // Mark as calculated (not fully explored)
		if sector.Constellation == "" {
			sector.Constellation = "??? (port data/calc only)"
		}
	}
	sector.UpDate = time.Now()

	// Always save/update sector to ensure it exists in current transaction context
	if err := p.GetDatabase().SaveSector(sector, sectorNum); err != nil {
		return fmt.Errorf("failed to save sector %d: %w", sectorNum, err)
	}

	// Save port data
	if err := p.GetDatabase().SavePort(port, sectorNum); err != nil {
		return fmt.Errorf("failed to save port for sector %d: %w", sectorNum, err)
	}

	// Fire any necessary events (consistent with other database operations)
	// This ensures proper notification flow like other database saves
	return nil
}

// clearPortData removes port data from the database for a sector that has no port
// This is called when we visit a sector and confirm it has no port
func (p *TWXParser) clearPortData(sectorIndex int) error {
	if p.getDatabaseFunc() == nil {
		return fmt.Errorf("database not available")
	}

	// Delete port data from the ports table
	if err := p.GetDatabase().DeletePort(sectorIndex); err != nil {
		return fmt.Errorf("failed to delete port data for sector %d: %w", sectorIndex, err)
	}

	return nil
}

// resetCurrentSector clears sector parsing data when starting a new sector visit
// This preserves important persistent data while clearing temporary parsing state
func (p *TWXParser) resetCurrentSector() {
	// Only reset parsing-specific data, not persistent sector information
	// Phase 4.5: Port, beacon, constellation now tracked via trackers (no intermediate objects)
	// Note: We intentionally do NOT reset Index or NavHaz as those are set elsewhere

	// Phase 4.5: Intermediate object collections removed - using trackers only
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
	sector, err := p.GetDatabase().LoadSector(sectorNum)
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
	if err := p.GetDatabase().SaveSector(sector, sectorNum); err != nil {
		panic(fmt.Sprintf("Critical database error in processDensityLine SaveSector for sector %d: %v", sectorNum, err))
	}
}

// processDensityLineTracker processes density scanner data using straight-sql tracker approach
func (p *TWXParser) processDensityLineTracker(line string) {
	// Parse density scan format: "Sector  XXXX  ==>           DENSITY  Warps : N    NavHaz :     X%    Anom : Yes/No"
	if !strings.HasPrefix(line, "Sector") || !strings.Contains(line, "==>") {
		return
	}

	if p.getDatabaseFunc() == nil {
		return
	}

	// Extract sector number
	x := line
	x = strings.ReplaceAll(x, "(", "")
	x = strings.ReplaceAll(x, ")", "")

	sectorNum := p.parseIntSafe(p.getParameter(x, 2))
	if sectorNum <= 0 {
		return
	}

	// Check if this is the current sector BEFORE changing any state
	isCurrentSector := (sectorNum == p.currentSectorIndex && p.currentSectorIndex > 0)

	// Initialize tracker for this sector
	var densityTracker *SectorTracker
	if p.sectorTracker == nil || p.currentSectorIndex != sectorNum {
		log.Info("SECTOR_TRACKER_LIFECYCLE: Creating sectorTracker in density parsing", "sector", sectorNum, "previous_sector", p.currentSectorIndex, "tracker_was_nil", p.sectorTracker == nil)
		p.currentSectorIndex = sectorNum
		p.sectorTracker = NewSectorTracker(sectorNum)
		p.sectorCollections = NewSectorCollections(sectorNum)
		p.portTracker = NewPortTracker(sectorNum)
	}

	// For density scans, we should only ADD density data, not overwrite exploration status
	// If this is the same sector that was just completed, create a separate tracker that only sets density fields
	if isCurrentSector {
		// Same sector that was just visited - create separate tracker to preserve existing exploration status
		densityTracker = NewSectorTracker(sectorNum)
	} else {
		// Different sector - use the normal tracker
		densityTracker = p.sectorTracker
	}

	// Parse density (parameter 4, remove commas)
	densityStr := p.getParameter(x, 4)
	densityStr = strings.ReplaceAll(densityStr, ",", "")
	if density := p.parseIntSafe(densityStr); density > 0 {
		densityTracker.SetDensity(density)
	}

	// Parse anomaly (parameter 13: "Yes" or "No")
	anomalyParam := p.getParameter(x, 13)
	densityTracker.SetAnomaly(anomalyParam == "Yes")

	// Parse NavHaz (parameter 10, remove % sign)
	navhazStr := p.getParameter(x, 10)
	if len(navhazStr) > 0 && strings.HasSuffix(navhazStr, "%") {
		navhazStr = navhazStr[:len(navhazStr)-1]
	}
	if navhaz := p.parseIntSafe(navhazStr); navhaz >= 0 {
		densityTracker.SetNavHaz(navhaz)
	}

	// Parse warp count (parameter 7)
	// For current sector: don't update warps (already accurate from sector visit)
	// For different sectors: update warps (this is the only info we have)
	if !isCurrentSector {
		warpCountStr := p.getParameter(x, 7)
		if warpCount := p.parseIntSafe(warpCountStr); warpCount > 0 {
			// For unvisited sectors, set warps count (but not individual warp destinations)
			densityTracker.updates[ColSectorWarps] = warpCount
		}
	}

	// Handle exploration status for density scans
	if isCurrentSector {
		// This is the current sector that was just visited - DO NOT change exploration status
		// The density scan should only ADD density data, not overwrite exploration status
	} else {
		// Different sector - set exploration status based on density scan discovery
		// Check current exploration status first to preserve higher statuses
		currentSector, err := p.GetDatabase().LoadSector(sectorNum)
		var currentExplored database.TSectorExploredType
		if err == nil {
			currentExplored = currentSector.Explored
		}

		// Only set to EtDensity if current status is EtNo or EtCalc (preserve EtHolo)
		if currentExplored == database.EtNo || currentExplored == database.EtCalc {
			densityTracker.SetExplored(int(database.EtDensity))
			densityTracker.SetConstellation("??? (Density only)")
		}
	}

	log.Info("DENSITY: Parsed density scan", "sector", sectorNum, "density", densityStr, "navhaz", navhazStr, "anomaly", anomalyParam)

	// Execute density tracker immediately (standalone updates)
	if densityTracker != nil && densityTracker.HasUpdates() {
		err := densityTracker.Execute(p.GetDatabase().GetDB())
		if err != nil {
			log.Info("DENSITY: Failed to update sector fields", "error", err)
		} else {
			log.Info("DENSITY: Successfully updated sector with density scan data", "sector", sectorNum)
		}
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

	// Enhanced Pascal-compliant fighter database reset
	if err := p.resetFighterDatabasePascalCompliant(); err != nil {
		// Fallback to simple database reset
		if err := p.GetDatabase().ResetPersonalCorpFighters(); err != nil {
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
	totalSectors := p.GetDatabase().GetSectors()
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
		sector, err := p.GetDatabase().LoadSector(i)
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
			if err := p.GetDatabase().SaveSector(sector, i); err != nil {
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
		sector, err := p.GetDatabase().LoadSector(i)
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
	if p.getDatabaseFunc() == nil {
		return
	}

	// Pascal logic: setup Federation beacon and constellation, port class 9
	sector, err := p.GetDatabase().LoadSector(sectorNum)
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
	if err := p.GetDatabase().SaveSector(sector, sectorNum); err != nil {
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
	if err := p.GetDatabase().SavePort(port, sectorNum); err != nil {
		return
	}
}

// setStardockSector stores the Stardock sector number in configuration
func (p *TWXParser) setStardockSector(sectorNum int) {
	// Store as script variable (Pascal stores in INI file, we'll use script variables)
	if err := p.GetDatabase().SaveScriptVariable("$STARDOCK", sectorNum); err != nil {
	} else {
	}
}

// getStardockSector retrieves the Stardock sector number from configuration
func (p *TWXParser) getStardockSector() int {
	if p.getDatabaseFunc() == nil {
		return 0
	}

	value, err := p.GetDatabase().LoadScriptVariable("$STARDOCK")
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

// DEPRECATED: sectorCompleted() is being phased out in favor of specific save functions.
// Instead of relying on this bulk save at sector completion, individual parsers should
// save their specific data immediately using functions like:
// - savePlayerCurrentSector()
// - saveSectorBasicInfo()
// - saveSectorPort()
// - saveProbeWarp()
// This prevents data overwrites and improves performance with targeted SQL updates.
func (p *TWXParser) sectorCompleted() {
	defer p.recoverFromPanic("sectorCompleted")
	log.Info("PROBE: sectorCompleted called", "sector", p.currentSectorIndex, "probe_mode", p.probeMode, "sector_saved", p.sectorSaved)

	// Skip if already completed to avoid duplicate TUI API calls
	if p.sectorSaved {
		log.Info("PROBE: Skipping sectorCompleted - already saved", "sector", p.currentSectorIndex)
		return
	}

	// Skip invalid sector numbers (0 means no sector has been parsed yet)
	if p.currentSectorIndex <= 0 {
		log.Info("PROBE: Skipping sectorCompleted - invalid sector number", "sector", p.currentSectorIndex)
		return
	}

	// Set immediately to prevent race conditions
	p.sectorSaved = true

	// Finalize any pending trader without ship details
	if p.sectorPosition == SectorPosTraders && p.currentTrader.Name != "" {
		p.validateTraderData(&p.currentTrader)
		// Phase 4.5: Traders tracked via collection trackers (no intermediate objects)
		p.currentTrader = TraderInfo{} // Reset
	}

	// Validate sector number before completion
	if !p.validateSectorNumber(p.currentSectorIndex) {
		return
	}

	// Validate all collected data before saving
	// Phase 4.5: Validation removed with intermediate objects

	// Phase 2: Execute trackers for straight-sql approach
	log.Info("SECTOR_TRACKER_LIFECYCLE: About to check sectorTracker", "sector", p.currentSectorIndex, "tracker_nil", p.sectorTracker == nil)
	if p.sectorTracker != nil {
		log.Info("SECTOR_TRACKER_LIFECYCLE: sectorTracker is not nil, proceeding", "sector", p.currentSectorIndex)
		// Set exploration status based on context (before executing)
		if p.probeMode {
			log.Info("SECTOR_TRACKER_LIFECYCLE: About to SetExplored (EtCalc)", "sector", p.currentSectorIndex, "tracker_nil_check", p.sectorTracker == nil)
			if p.sectorTracker != nil {
				p.sectorTracker.SetExplored(int(database.EtCalc)) // Mark as probe data
				log.Info("SECTOR: Setting exploration to EtCalc", "exploration_type", int(database.EtCalc), "sector", p.currentSectorIndex)
			} else {
				log.Error("SECTOR_TRACKER_LIFECYCLE: sectorTracker became nil during EtCalc!", "sector", p.currentSectorIndex)
			}
		} else {
			log.Info("SECTOR_TRACKER_LIFECYCLE: About to SetExplored (EtHolo)", "sector", p.currentSectorIndex, "tracker_nil_check", p.sectorTracker == nil)
			if p.sectorTracker != nil {
				p.sectorTracker.SetExplored(int(database.EtHolo)) // Mark current sector as visited by player
				log.Info("SECTOR: Setting exploration to EtHolo", "exploration_type", int(database.EtHolo), "sector", p.currentSectorIndex)
			} else {
				log.Error("SECTOR_TRACKER_LIFECYCLE: sectorTracker became nil during EtHolo!", "sector", p.currentSectorIndex)
			}
		}

		log.Info("SECTOR_TRACKER_LIFECYCLE: About to check HasUpdates", "sector", p.currentSectorIndex, "tracker_nil_check", p.sectorTracker == nil)
		if p.sectorTracker != nil && p.sectorTracker.HasUpdates() {
			log.Info("SECTOR_TRACKER_LIFECYCLE: About to Execute", "sector", p.currentSectorIndex)
			db := p.GetDatabase().GetDB()
			log.Info("SECTOR_TRACKER_LIFECYCLE: Database connection", "sector", p.currentSectorIndex, "db_nil", db == nil)
			if db == nil {
				log.Error("SECTOR_TRACKER_LIFECYCLE: Database connection is nil!", "sector", p.currentSectorIndex)
			} else {
				err := p.sectorTracker.Execute(db)
				if err != nil {
					log.Info("SECTOR_PARSER: Failed to update sector fields", "error", err)
				}
			}
		} else if p.sectorTracker == nil {
			log.Error("SECTOR_TRACKER_LIFECYCLE: sectorTracker became nil before HasUpdates!", "sector", p.currentSectorIndex)
		}
	} else {
		log.Warn("SECTOR_TRACKER_LIFECYCLE: sectorTracker is nil at start of execution", "sector", p.currentSectorIndex)
	}

	if p.sectorCollections != nil && p.sectorCollections.HasData() {
		err := p.sectorCollections.Execute(p.GetDatabase().GetDB())
		if err != nil {
			log.Info("SECTOR_PARSER: Failed to update sector collections", "error", err)
		}
	}

	// Phase 3: Execute port tracker for straight-sql approach
	if p.portTracker != nil && p.portTracker.HasUpdates() {
		err := p.portTracker.Execute(p.GetDatabase().GetDB())
		if err != nil {
			log.Info("PORT_PARSER: Failed to update port fields", "error", err)
		} else {
			// Phase 3: Fire OnPortUpdated API event with fresh database read
			if p.tuiAPI != nil {
				portInfo, err := p.GetDatabase().GetPortInfo(p.currentSectorIndex)
				if err == nil && portInfo != nil {
					log.Info("PORT_PARSER: Firing OnPortUpdated", "sector", p.currentSectorIndex, "port_name", portInfo.Name, "class", portInfo.Class)
					p.tuiAPI.OnPortUpdated(*portInfo)
				} else {
					log.Info("PORT_PARSER: Failed to read fresh port info for API event", "error", err)
				}
			}
		}
	}

	// Phase 4.5: Sector data and warps saved via tracker execution

	// Phase 3: Port data tracked directly via PortTracker during parsing (no intermediate objects)

	// Phase 2: Exploration status now handled by tracker system above
	// Legacy saveSectorProbeData/saveSectorVisited calls removed

	// Fire TUI current sector change event (but not for probe-discovered sectors or probe mode)
	isProbeDiscovered := p.probeDiscoveredSectors[p.currentSectorIndex]
	shouldSuppressEvent := p.probeMode || isProbeDiscovered
	if p.tuiAPI != nil && !shouldSuppressEvent {
		// Phase 2: Use fresh database read for basic API event
		freshSectorInfo, err := p.GetDatabase().GetSectorInfo(p.currentSectorIndex)
		if err == nil {
			log.Info("TWX_PARSER: Firing OnCurrentSectorChanged [SOURCE: sectorCompleted]", "sector", freshSectorInfo.Number, "probe_mode", p.probeMode, "probe_discovered", isProbeDiscovered)
			p.tuiAPI.OnCurrentSectorChanged(freshSectorInfo)
		} else {
			log.Info("TWX_PARSER: Failed to read fresh sector info for API event", "error", err)
		}
	} else if p.tuiAPI != nil {
		log.Info("TWX_PARSER: Suppressing OnCurrentSectorChanged [SOURCE: sectorCompleted]", "sector", p.currentSectorIndex, "probe_mode", p.probeMode, "probe_discovered", isProbeDiscovered)
	}

	// Phase 4.5: Fire sector events to event bus with fresh data from database
	if p.eventBus != nil {
		event := Event{
			Type: EventSectorComplete,
			Data: map[string]interface{}{
				"sector": p.currentSectorIndex,
			},
			Source: "TWXParser",
		}
		p.eventBus.Fire(event)
		log.Info("TWX_PARSER: Fired EventSectorComplete to event bus", "sector", p.currentSectorIndex)
	}

	// Phase 4.5: Fire sector events to observers
	if len(p.observers) > 0 {
		event := Event{
			Type: EventSectorComplete,
			Data: map[string]interface{}{
				"sector": p.currentSectorIndex,
			},
			Source: "TWXParser",
		}
		p.Notify(event)
		log.Info("TWX_PARSER: Fired EventSectorComplete to observers", "sector", p.currentSectorIndex)
	}

	// Phase 2: Reset trackers for next parsing session
	log.Info("SECTOR_TRACKER_LIFECYCLE: Setting trackers to nil", "sector", p.currentSectorIndex, "tracker_was_nil", p.sectorTracker == nil)
	p.sectorTracker = nil
	p.sectorCollections = nil
	p.portTracker = nil
}

// parseIntSafe is now implemented in parser_utils.go

// Reset resets the parser state
func (p *TWXParser) Reset() {
	log.Info("RESET: Full parser reset called", "previous_lastWarp", p.lastWarp)
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
	log.Info("RESET: Full parser reset completed", "current_lastWarp", p.lastWarp)
}

// GetCurrentSector returns the current sector index
func (p *TWXParser) GetCurrentSector() int {
	return p.currentSectorIndex
}

// GetDisplayState returns the current display state
func (p *TWXParser) GetDisplayState() DisplayType {
	return p.currentDisplay
}

// GetPlayerStats returns the current player statistics from database (straight-sql pattern)
func (p *TWXParser) GetPlayerStats() (*api.PlayerStatsInfo, error) {
	stats, err := p.GetDatabase().GetPlayerStatsInfo()
	return &stats, err
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

// GetCurrentTurns returns current turns from database (straight-sql pattern)
func (p *TWXParser) GetCurrentTurns() int {
	if playerInfo, err := p.GetDatabase().GetPlayerStatsInfo(); err == nil {
		return playerInfo.Turns
	}
	return 0
}

// GetCurrentCredits returns current credits from database (straight-sql pattern)
func (p *TWXParser) GetCurrentCredits() int {
	if playerInfo, err := p.GetDatabase().GetPlayerStatsInfo(); err == nil {
		return playerInfo.Credits
	}
	return 0
}

// GetCurrentFighters returns current fighters from database (straight-sql pattern)
func (p *TWXParser) GetCurrentFighters() int {
	if playerInfo, err := p.GetDatabase().GetPlayerStatsInfo(); err == nil {
		return playerInfo.Fighters
	}
	return 0
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
//
//	Col 0│Phot 0│Armd 0│Lmpt 0│GTorp 0│TWarp No│Clks 0│Beacns 0│AtmDt 0│Crbo 0
//	EPrb 0│MDis 0│PsPrb No│PlScn No│LRS None,Dens,Holo│Aln 0│Exp 0│Ship 1 MerCru

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

	// Phase 2: Record discovered warp fields
	if p.sectorTracker != nil {
		p.sectorTracker.SetWarps(warps)
	}

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
	// For each destination sector, ensure it has a reverse warp back to this sector
	// This mirrors the Pascal AddWarp logic for maintaining bidirectional connectivity
	for _, toSector := range warps {
		if toSector > 0 {
			p.addReverseWarp(toSector, fromSector)
		}
	}
}

// addProbeWarp adds a one-way warp connection discovered by probe movement
func (p *TWXParser) addProbeWarp(fromSector, toSector int) {
	log.Info("PROBE WARP: addProbeWarp called", "from_sector", fromSector, "to_sector", toSector)

	// Phase 4.5: Probe warps saved via sector tracker
	// Create a sector tracker for the fromSector and add the warp
	fromTracker := NewSectorTracker(fromSector)

	// Load existing warps from database to preserve them
	if sectorInfo, err := p.GetDatabase().LoadSector(fromSector); err == nil {
		// Set existing warps plus the new one
		existingWarps := sectorInfo.Warp

		// Find an empty slot and add the new warp
		for i := 0; i < 6; i++ {
			if existingWarps[i] == 0 {
				existingWarps[i] = toSector
				break
			} else if existingWarps[i] == toSector {
				// Warp already exists, no need to add it again
				log.Info("PROBE WARP: Warp already exists", "from_sector", fromSector, "to_sector", toSector)
				return
			}
		}

		fromTracker.SetWarps(existingWarps)
	} else {
		// No existing sector, create new one with just this warp
		newWarps := [6]int{toSector, 0, 0, 0, 0, 0}
		fromTracker.SetWarps(newWarps)
	}

	// Execute the tracker to save the warp
	err := fromTracker.Execute(p.GetDatabase().GetDB())
	if err != nil {
		log.Info("PROBE WARP: Failed to save probe warp", "from_sector", fromSector, "to_sector", toSector, "error", err)
		return
	}
	log.Info("PROBE WARP: Successfully saved probe warp", "from_sector", fromSector, "to_sector", toSector)
}

// addReverseWarp adds a reverse warp connection (mirrors Pascal AddWarp method)
func (p *TWXParser) addReverseWarp(toSector, fromSector int) {
	// Load the destination sector
	sector, err := p.GetDatabase().LoadSector(toSector)
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
		if err := p.GetDatabase().SaveSector(sector, toSector); err != nil {
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
func (p *TWXParser) FireTextLineEvent(line string, outbound bool) (bool, error) {
	if p.scriptInterpreter != nil {
		p.scriptInterpreter.TextLineEvent(line, outbound)
	}

	// Also fire through the ScriptEventProcessor for the new scripting engine
	if p.scriptEventProcessor != nil && p.scriptEventProcessor.IsEnabled() {
		return p.scriptEventProcessor.FireTextLineEvent(line, outbound)
	}

	return false, nil
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

// UpdateCurrentLine updates the CURRENTLINE system constant (mirrors TWX ProcessInBound behavior)
func (p *TWXParser) UpdateCurrentLine(line string) {
	// Update CURRENTLINE through the ScriptEventProcessor
	if p.scriptEventProcessor != nil && p.scriptEventProcessor.IsEnabled() {
		p.scriptEventProcessor.UpdateCurrentLine(line)
	}
}

// ProcessOutBound processes outbound data and returns whether to continue sending
func (p *TWXParser) ProcessOutBound(data string) bool {

	// Fire outbound text events
	p.FireTextEvent(data, true)
	_, err := p.FireTextLineEvent(data, true)
	if err != nil {
		log.Error("Error firing outbound TextLineEvent", "error", err, "data", data)
	}

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
					log.Error("PANIC recovered in observer update", "function", "notifyObservers", "error", r)
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
			"property": property,
			"oldValue": oldValue,
			"newValue": newValue,
			"sector":   p.currentSectorIndex,
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

// firePlayerStatsEventDirect fires a player statistics update event using API PlayerStatsInfo directly
// This is used by the straight-sql pattern where we read fresh data from database
func (p *TWXParser) firePlayerStatsEventDirect(stats api.PlayerStatsInfo) {
	if p.tuiAPI != nil {
		// Fire the event with fresh database data
		p.tuiAPI.OnPlayerStatsUpdated(stats)
	}
}
