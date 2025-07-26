package streaming

import (
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"twist/internal/database"
)

// TWX Display State Machine - matching TWX TDisplay enum
type TDisplay int

const (
	DNone TDisplay = iota
	DSector
	DDensity
	DWarpLane
	DCIM
	DPortCIM
	DPort
	DPortCR
	DWarpCIM
	DFigScan
)

// TWX Sector Position - matching TWX TSectorPosition enum
type TSectorPosition int

const (
	SpNormal TSectorPosition = iota
	SpPorts
	SpPlanets
	SpShips
	SpMines
	SpTraders
)

// Fig Scan Type - matching TWX TFigScanType enum
type TFigScanType int

const (
	FstPersonal TFigScanType = iota
	FstCorp
)

// SectorParser handles parsing of sector information from game text
type SectorParser struct {
	logger      *log.Logger
	dataLog     *log.Logger
	ansiPattern *regexp.Regexp // For stripping ANSI codes

	// TWX State Machine fields (matching TWX TModExtractor)
	currentDisplay     TDisplay
	sectorPosition     TSectorPosition
	currentSectorIndex int
	portSectorIndex    int
	figScanSector      int
	lastWarp           int
	sectorSaved        bool
	figScanType        TFigScanType

	// Current parsing data
	currentSector *database.TSector
	currentShip   *database.TShip
	currentTrader *database.TTrader

	// Database
	db database.Database
}

// Remove the patterns struct - we'll use simple string matching like TWX

// NewSectorParser creates a new sector parser with database
func NewSectorParser(db database.Database) *SectorParser {
	// Set up debug logging
	logFile, err := os.OpenFile("twist_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	// Set up data logging
	dataLogFile, err := os.OpenFile("data.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open data log file: %v", err)
	}

	logger := log.New(logFile, "[SECTOR_PARSER] ", log.LstdFlags|log.Lshortfile)
	dataLog := log.New(dataLogFile, "", log.LstdFlags)

	return &SectorParser{
		logger:         logger,
		dataLog:        dataLog,
		ansiPattern:    regexp.MustCompile(`\x1b\[[0-9;]*[mK]`), // ANSI escape sequences
		currentDisplay: DNone,
		sectorPosition: SpNormal,
		db:             db,
	}
}

// stripANSI removes ANSI escape sequences from text
func (sp *SectorParser) stripANSI(text string) string {
	return sp.ansiPattern.ReplaceAllString(text, "")
}

// strToIntSafe converts string to int, returning 0 on error (like TWX)
func (sp *SectorParser) strToIntSafe(s string) int {
	if val, err := strconv.Atoi(s); err == nil {
		return val
	}
	return 0
}

// getParameter gets the nth parameter from a space-separated string (1-indexed like TWX)
func (sp *SectorParser) getParameter(line string, paramNum int) string {
	fields := strings.Fields(line)
	if paramNum > 0 && paramNum <= len(fields) {
		return fields[paramNum-1]
	}
	return ""
}

// stripChar removes all instances of a character from a string (like TWX)
func (sp *SectorParser) stripChar(line *string, char rune) {
	*line = strings.ReplaceAll(*line, string(char), "")
}

// sectorCompleted saves the current sector if not already saved (matching TWX)
func (sp *SectorParser) sectorCompleted() {
	if sp.currentSector != nil && !sp.sectorSaved {
		// Save to database
		if err := sp.db.SaveSector(*sp.currentSector, sp.currentSectorIndex); err != nil {
			sp.logger.Printf("Failed to save sector %d: %v", sp.currentSectorIndex, err)
		} else {
			sp.logger.Printf("Saved sector %d to database", sp.currentSectorIndex)
		}

		// Also log to data.log for compatibility
		sp.logSectorInfoTWX(sp.currentSector)
		sp.sectorSaved = true
	}
}

// processSectorLine processes a single line using TWX-style parsing with position tracking
func (sp *SectorParser) processSectorLine(line string) {
	// Beacon line: "Beacon  : FedSpace, FedLaw Enforced"
	if len(line) >= 10 && line[:10] == "Beacon  : " {
		sp.currentSector.Beacon = line[10:]
		sp.logger.Printf("Found beacon for sector %d: %s", sp.currentSectorIndex, sp.currentSector.Beacon)
		sp.dataLog.Printf("PARSED_BEACON: Sector=%d, Beacon='%s'", sp.currentSectorIndex, sp.currentSector.Beacon)
	} else if len(line) >= 10 && line[:10] == "Ports   : " {
		// Port line: "Ports   : Sol, Class 0 (Special)" or "Ports   : <=-DANGER-=>"
		if strings.Contains(line, "<=-DANGER-=>") {
			// Port is destroyed (like TWX does)
			sp.currentSector.SPort.Dead = true
			sp.currentSector.SPort.Name = ""
			sp.currentSector.SPort.ClassIndex = -1
			sp.logger.Printf("Found destroyed port in sector %d", sp.currentSectorIndex)
			sp.dataLog.Printf("PARSED_PORT: Sector=%d, Name='', Dead=true, Class=-1", sp.currentSectorIndex)
		} else {
			// Port is alive
			sp.currentSector.SPort.Dead = false
			sp.currentSector.SPort.BuildTime = 0

			if strings.Contains(line, ", Class") {
				classPos := strings.Index(line, ", Class")
				sp.currentSector.SPort.Name = line[10:classPos]

				// Extract class index  
				classStr := sp.getParameter(line[classPos:], 3)
				sp.currentSector.SPort.ClassIndex = sp.strToIntSafe(classStr)

				// Extract BBS status from end of line (like TWX: BBB = Buying all three)
				lineEnd := line[len(line)-3:]
				sp.currentSector.SPort.BuyProduct[0] = lineEnd[0:1] == "B" // Fuel Ore
				sp.currentSector.SPort.BuyProduct[1] = lineEnd[1:2] == "B" // Organics  
				sp.currentSector.SPort.BuyProduct[2] = lineEnd[2:3] == "B" // Equipment
			} else {
				sp.currentSector.SPort.Name = line[10:]
			}
			sp.logger.Printf("Found port in sector %d: %s (Class %d)", sp.currentSectorIndex, sp.currentSector.SPort.Name, sp.currentSector.SPort.ClassIndex)
			sp.dataLog.Printf("PARSED_PORT: Sector=%d, Name='%s', Dead=false, Class=%d, BuyProducts=[%t,%t,%t]", 
				sp.currentSectorIndex, sp.currentSector.SPort.Name, sp.currentSector.SPort.ClassIndex,
				sp.currentSector.SPort.BuyProduct[0], sp.currentSector.SPort.BuyProduct[1], sp.currentSector.SPort.BuyProduct[2])
		}
		sp.sectorPosition = SpPorts
	} else if len(line) >= 10 && line[:10] == "Planets : " {
		// Planet line: "Planets : (M) Terra"
		planetName := line[10:]
		planet := database.TPlanet{Name: planetName}
		sp.currentSector.Planets = append(sp.currentSector.Planets, planet)
		sp.sectorPosition = SpPlanets
		sp.logger.Printf("Found planet in sector %d: %s", sp.currentSectorIndex, planetName)
		sp.dataLog.Printf("PARSED_PLANET: Sector=%d, Name='%s'", sp.currentSectorIndex, planetName)
	} else if len(line) >= 10 && line[:10] == "Traders : " {
		// Trader line: "Traders : Gypsy in Class 0 (Cadet)'s  (Armored Mercenary)  [1,000 Fuel Ore]  [*  *] [with 50 ftrs]"
		sp.parseTraderLine(line)
		sp.sectorPosition = SpTraders
	} else if len(line) >= 10 && line[:10] == "Ships   : " {
		// Ship line: "Ships   : (Nicks) in a Class 6 (Corellian Corvette)  [*  *] [with 25 ftrs, shlds down]"
		sp.parseShipLine(line)
		sp.sectorPosition = SpShips
	} else if len(line) >= 10 && line[:10] == "Fighters: " {
		// Fighter line: "Fighters: 800 belong to your Corp"
		sp.parseFighterLine(line)
	} else if len(line) >= 10 && line[:10] == "Mines   : " {
		// Mine line: "Mines   : 5 Armid mines (Mines belong to the Admiral)"
		sp.parseMineLine(line)
		sp.sectorPosition = SpMines
	} else if len(line) >= 10 && line[:10] == "NavHaz  : " {
		// Navigation hazard: "NavHaz  : 5%"
		navHazStr := sp.getParameter(line, 3)
		if strings.HasSuffix(navHazStr, "%") {
			navHazStr = navHazStr[:len(navHazStr)-1]
		}
		sp.currentSector.NavHaz = sp.strToIntSafe(navHazStr)
		sp.logger.Printf("Found nav hazard for sector %d: %d%%", sp.currentSectorIndex, sp.currentSector.NavHaz)
	} else if len(line) >= 20 && line[:20] == "Warps to Sector(s) :" {
		// Warp line: "Warps to Sector(s) :  (2) - (3) - (4) - (5) - (6) - (7)"
		workLine := line
		sp.stripChar(&workLine, '(')
		sp.stripChar(&workLine, ')')

		// Extract warps using GetParameter like TWX (into array[6])
		sp.currentSector.Warp[0] = sp.strToIntSafe(sp.getParameter(workLine, 5))
		sp.currentSector.Warp[1] = sp.strToIntSafe(sp.getParameter(workLine, 7))
		sp.currentSector.Warp[2] = sp.strToIntSafe(sp.getParameter(workLine, 9))
		sp.currentSector.Warp[3] = sp.strToIntSafe(sp.getParameter(workLine, 11))
		sp.currentSector.Warp[4] = sp.strToIntSafe(sp.getParameter(workLine, 13))
		sp.currentSector.Warp[5] = sp.strToIntSafe(sp.getParameter(workLine, 15))

		// Count valid warps (like TWX does)
		warpCount := 0
		for i := 0; i < 6; i++ {
			if sp.currentSector.Warp[i] > 0 {
				warpCount++
			} else {
				break
			}
		}
		sp.currentSector.Warps = warpCount

		sp.logger.Printf("Found warps for sector %d: %v (count: %d)", sp.currentSectorIndex, sp.currentSector.Warp, warpCount)
		sp.dataLog.Printf("PARSED_WARPS: Sector=%d, Warps=%v, Count=%d", sp.currentSectorIndex, sp.currentSector.Warp[:warpCount], warpCount)

		// Sector is complete when we see warps (like TWX)
		sp.sectorCompleted()
	} else if len(line) >= 8 && line[:8] == "        " {
		// Multi-line continuation (8 spaces) - continue from last occurrence (like TWX)
		sp.processContinuationLine(line)
	} else if len(line) >= 9 && line[8:9] == ":" {
		// Generic header line detected, reset position
		sp.sectorPosition = SpNormal
	} else {
		// Process continuation lines based on current position
		sp.processContinuationLine(line)
	}
}

// ParseText processes text data using TWX ProcessLine state machine
func (sp *SectorParser) ParseText(text string) {
	sp.logger.Printf("Parsing text for game data: %d chars", len(text))

	lines := strings.Split(text, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Use TWX-style ProcessLine state machine
		sp.ProcessLine(line)
	}
}

// logSectorInfoTWX writes sector information to data log (for backward compatibility)
func (sp *SectorParser) logSectorInfoTWX(sector *database.TSector) {
	// Convert warp array to slice for logging
	var warps []int
	for i := 0; i < sector.Warps; i++ {
		warps = append(warps, sector.Warp[i])
	}

	sp.dataLog.Printf("SECTOR_DATA: Number=%d, Warps=%v, NavHaz=%d%%, Constellation='%s', Beacon='%s', Density=%d, Anomaly=%t, Port=%t, PortName='%s'",
		sp.currentSectorIndex,
		warps,
		sector.NavHaz,
		sector.Constellation,
		sector.Beacon,
		sector.Density,
		sector.Anomaly,
		sector.SPort.Name != "",
		sector.SPort.Name,
	)

	sp.logger.Printf("Logged sector data for sector %d", sp.currentSectorIndex)
}

// ProcessLine processes a single line using TWX state machine logic
func (sp *SectorParser) ProcessLine(line string) {
	// Strip ANSI codes first (like TWX does)
	cleanLine := sp.stripANSI(line)

	// Route to appropriate handler based on current display state
	if len(cleanLine) >= 19 && cleanLine[:19] == "The shortest path (" {
		// Computer plotted course
		sp.currentDisplay = DWarpLane
		sp.lastWarp = 0
		sp.processWarpLine(cleanLine)
	} else if len(cleanLine) >= 7 && cleanLine[:7] == "  TO > " {
		// Computer plotted course continuation
		sp.currentDisplay = DWarpLane
		sp.processWarpLine(cleanLine)
	} else if sp.currentDisplay == DWarpLane {
		sp.processWarpLine(cleanLine)
	} else if len(cleanLine) >= 42 && cleanLine[26:42] == "Relative Density" {
		// Density scanner being used
		sp.currentDisplay = DDensity
		sp.processDensityLine(cleanLine)
	} else if sp.currentDisplay == DDensity && len(cleanLine) >= 6 && cleanLine[:6] == "Sector" {
		sp.processDensityLine(cleanLine)
	} else if len(cleanLine) >= 26 && cleanLine[:26] == "Incoming transmission from" {
		// Transmission with ANSI off (TWX pattern)
		sp.processTransmissionLine(cleanLine)
	} else if len(cleanLine) >= 28 && cleanLine[:28] == "Continuing transmission from" {
		// Continuing transmission (TWX pattern)
		sp.processTransmissionLine(cleanLine)
	} else if len(cleanLine) >= 22 && len(cleanLine) >= 43 && cleanLine[13:21] == "StarDock" && cleanLine[36:42] == "sector" {
		// StarDock detection from 'V' screen (TWX pattern)
		sp.processStarDockLine(cleanLine)
	} else if len(cleanLine) >= 40 && cleanLine[17:40] == "Deployed  Fighter  Scan" {
		// Fighter scan started
		sp.currentDisplay = DFigScan
		sp.figScanSector = 0
		sp.processFigScanLine(cleanLine)
	} else if sp.currentDisplay == DFigScan {
		sp.processFigScanLine(cleanLine)
	} else if len(cleanLine) >= 10 && cleanLine[:10] == "Sector  : " {
		// Sector display started
		sp.sectorCompleted() // Save previous sector if any

		sp.currentDisplay = DSector
		sp.sectorSaved = false

		// Parse sector header
		sectorNum := sp.strToIntSafe(sp.getParameter(cleanLine, 3))
		sector := database.NULLSector()
		sp.currentSector = &sector
		sp.currentSectorIndex = sectorNum

		// Extract constellation (everything after "in")
		if strings.Contains(cleanLine, " in ") {
			inPos := strings.Index(cleanLine, " in ")
			sp.currentSector.Constellation = strings.TrimSpace(cleanLine[inPos+4:])
		}

		// Set exploration level and timestamp
		sp.currentSector.Explored = database.EtHolo // Viewing sector = holo scan
		sp.currentSector.UpDate = time.Now()

		sp.logger.Printf("Started processing sector %d in %s", sectorNum, sp.currentSector.Constellation)
		sp.dataLog.Printf("PARSING_STARTED: Sector=%d, Constellation='%s', Explored=%d",
			sectorNum, sp.currentSector.Constellation, int(sp.currentSector.Explored))
	} else if sp.currentDisplay == DSector {
		sp.processSectorLine(cleanLine)
	} else if len(cleanLine) >= 10 && cleanLine[:10] == "Docking..." {
		// Port display started
		sp.sectorCompleted() // Save current sector
		sp.currentDisplay = DPort
		sp.portSectorIndex = sp.currentSectorIndex
		if sp.portSectorIndex > 0 {
			loadedSector, err := sp.db.LoadSector(sp.portSectorIndex)
			if err == nil {
				sp.currentSector = &loadedSector
			}
		}
		sp.sectorSaved = false
		sp.processPortLine(cleanLine)
	} else if sp.currentDisplay == DPort {
		sp.processPortLine(cleanLine)
	} else if len(cleanLine) >= 28 && cleanLine[:28] == "What sector is the port in? " {
		// Computer port report
		sp.currentDisplay = DPortCR
		// Extract sector number from prompt
		rightBracket := strings.Index(cleanLine, "]")
		if rightBracket != -1 && len(cleanLine) > rightBracket+1 {
			sp.portSectorIndex = sp.strToIntSafe(cleanLine[rightBracket+1:])
		} else {
			sp.portSectorIndex = sp.currentSectorIndex
		}
		if sp.portSectorIndex > 0 {
			loadedSector, err := sp.db.LoadSector(sp.portSectorIndex)
			if err == nil {
				sp.currentSector = &loadedSector
			}
		}
		sp.processPortLine(cleanLine)
	} else if sp.currentDisplay == DPortCR {
		sp.processPortLine(cleanLine)
	} else if len(cleanLine) >= 21 && cleanLine[:21] == "Computer command [TL=" {
		// Computer prompt - kill all displays
		sp.currentDisplay = DNone
		sp.lastWarp = 0
	} else if len(cleanLine) >= 2 && cleanLine[:2] == ": " {
		// CIM prompt detection
		if sp.currentDisplay != DCIM {
			sp.currentDisplay = DNone
		}
		sp.lastWarp = 0
	} else if strings.Contains(cleanLine, "Computer Information Matrix") {
		// CIM download started
		sp.currentDisplay = DCIM
		sp.processCIMLine(cleanLine)
	} else if sp.currentDisplay == DCIM {
		// Determine CIM type based on content
		if len(cleanLine) > 2 && cleanLine[len(cleanLine)-1] == '%' {
			sp.currentDisplay = DPortCIM
		} else {
			sp.currentDisplay = DWarpCIM
		}
		sp.processCIMLine(cleanLine)
	} else if sp.currentDisplay == DWarpCIM || sp.currentDisplay == DPortCIM {
		sp.processCIMLine(cleanLine)
	} else if strings.Contains(cleanLine, "♫") || (len(cleanLine) >= 5 && cleanLine[:5] == " Ship") {
		// QuickStats line (contains special character or starts with " Ship")
		sp.processQuickStats(cleanLine)
	} else if len(cleanLine) >= 14 && cleanLine[:14] == "TradeWars Game" {
		// TWX: TradeWars Game Server version detection
		sp.logger.Printf("Detected TWGS v2.20b: %s", cleanLine)
	} else if len(cleanLine) >= 20 && cleanLine[:20] == "Trade Wars 2002 Game" {
		// TWX: Trade Wars 2002 Game Server detection  
		sp.logger.Printf("Detected TW2002 Game Server: %s", cleanLine)
	} else if len(cleanLine) >= 23 && cleanLine[:23] == "Probe entering sector :" {
		// TWX: Probe entering sector - save current sector
		sp.sectorCompleted()
		sp.logger.Printf("Probe entering sector detected")
	} else if len(cleanLine) >= 20 && cleanLine[:20] == "Probe Self Destructs" {
		// TWX: Probe self destruct - save current sector
		sp.sectorCompleted() 
		sp.logger.Printf("Probe self destruct detected")
	} else if len(cleanLine) >= 25 && cleanLine[:25] == "Citadel treasury contains" {
		// TWX: In Citadel - save current sector
		sp.sectorCompleted()
		sp.logger.Printf("Citadel treasury detected")
	} else if len(cleanLine) >= 20 && cleanLine[:20] == "No fighters deployed" {
		// TWX: Reset fighter database when no fighters deployed
		sp.logger.Printf("No fighters deployed - resetting fighter data")
	} else if len(cleanLine) >= 2 && (cleanLine[:2] == "R " || cleanLine[:2] == "F ") {
		// TWX: Message handling (Radio/Federation messages)
		sp.logger.Printf("Message: %s", cleanLine)
	} else if len(cleanLine) >= 2 && cleanLine[:2] == "P " {
		// TWX: Personal message handling
		if sp.getParameter(cleanLine, 2) != "indicates" {
			sp.logger.Printf("Personal message: %s", cleanLine)
		}
	} else if len(cleanLine) >= 31 && cleanLine[:31] == "Deployed Fighters Report Sector" {
		// TWX: Deployed fighters report
		sp.logger.Printf("Fighter report: %s", cleanLine[18:])
	} else if len(cleanLine) >= 20 && cleanLine[:20] == "Shipboard Computers " {
		// TWX: Shipboard computer activity
		sp.logger.Printf("Computer activity: %s", cleanLine[20:])
	}

	// Check for end of display conditions
	if strings.Contains(cleanLine, "Command [TL=") ||
		len(cleanLine) >= 19 && cleanLine[:19] == "Stop in this sector" ||
		len(cleanLine) >= 21 && cleanLine[:21] == "Engage the Autopilot?" {
		sp.currentDisplay = DNone
		sp.sectorPosition = SpNormal
	}
}

// processWarpLine processes computer plotted course lines
func (sp *SectorParser) processWarpLine(line string) {
	// Warp line format: "3 > 300 > 5362 > 13526 > 149 > 434"
	// Add bidirectional warps between consecutive sectors

	lastSect := sp.lastWarp

	// Clean up the line
	workLine := line
	sp.stripChar(&workLine, '(')
	sp.stripChar(&workLine, ')')

	// Split on ' >' to get sector numbers
	sectors := strings.Split(workLine, " >")

	for _, sectorStr := range sectors {
		sectorStr = strings.TrimSpace(sectorStr)
		if sectorStr == "" || sectorStr == ">" {
			continue
		}

		curSect := sp.strToIntSafe(sectorStr)
		if curSect < 1 {
			// Invalid sector number, abort processing
			sp.logger.Printf("Invalid sector number in warp line: %s", sectorStr)
			return
		}

		// Add bidirectional warp between last and current sector
		if lastSect > 0 {
			sp.addWarp(lastSect, curSect)
			sp.addWarp(curSect, lastSect) // Bidirectional
		}

		lastSect = curSect
		sp.lastWarp = curSect
	}

	sp.logger.Printf("Processed warp line with %d sectors", len(sectors))
}

// addWarp adds a warp to a sector if it doesn't already exist
func (sp *SectorParser) addWarp(sectorNum, warpTo int) {
	// Load the sector
	sector, err := sp.db.LoadSector(sectorNum)
	if err != nil {
		// Create new sector if not found
		sector = database.NULLSector()
	}

	// Check if warp already exists
	for i := 0; i < 6; i++ {
		if sector.Warp[i] == warpTo {
			return // Warp already exists
		}
	}

	// Find position to insert warp (sorted order)
	insertPos := 6 // Default to end (no space)
	for i := 0; i < 6; i++ {
		if sector.Warp[i] == 0 || sector.Warp[i] > warpTo {
			insertPos = i
			break
		}
	}

	// Insert warp at position, shifting others if needed
	if insertPos < 6 {
		// Shift warps to make room
		for i := 5; i > insertPos; i-- {
			if i > 0 {
				sector.Warp[i] = sector.Warp[i-1]
			}
		}

		// Insert new warp
		sector.Warp[insertPos] = warpTo

		// Update warp count
		warpCount := 0
		for i := 0; i < 6; i++ {
			if sector.Warp[i] > 0 {
				warpCount++
			}
		}
		sector.Warps = warpCount

		// Save the updated sector
		if err := sp.db.SaveSector(sector, sectorNum); err != nil {
			sp.logger.Printf("Failed to save warp data for sector %d: %v", sectorNum, err)
		} else {
			sp.logger.Printf("Added warp %d -> %d", sectorNum, warpTo)
			sp.dataLog.Printf("PARSED_WARP_ADDITION: FromSector=%d, ToSector=%d, NewWarpCount=%d",
				sectorNum, warpTo, warpCount)
		}
	}
}

// processDensityLine processes density scanner output
func (sp *SectorParser) processDensityLine(line string) {
	// Density scanner output format:
	// "Sector (1) :    1,200  (3 warps)  5% Navhaz  Anomaly: Yes"
	if len(line) >= 6 && line[:6] == "Sector" {
		// Clean up the line for parsing
		workLine := line
		sp.stripChar(&workLine, '(')
		sp.stripChar(&workLine, ')')

		// Extract sector number (parameter 2 after stripping parentheses)
		sectorNum := sp.strToIntSafe(sp.getParameter(workLine, 2))
		if sectorNum <= 0 {
			sp.logger.Printf("Invalid sector number in density line: %s", line)
			return
		}

		// Load existing sector or create new one
		sector, err := sp.db.LoadSector(sectorNum)
		if err != nil {
			// Create new sector if not found
			sector = database.NULLSector()
		}

		// Extract density (parameter 4, remove comma)
		densityStr := sp.getParameter(workLine, 4)
		sp.stripChar(&densityStr, ',')
		sector.Density = sp.strToIntSafe(densityStr)

		// Extract warp count (parameter 7)
		sector.Warps = sp.strToIntSafe(sp.getParameter(workLine, 7))

		// Extract nav hazard (parameter 10, remove % sign)
		navHazStr := sp.getParameter(workLine, 10)
		if len(navHazStr) > 0 && navHazStr[len(navHazStr)-1:] == "%" {
			navHazStr = navHazStr[:len(navHazStr)-1]
		}
		sector.NavHaz = sp.strToIntSafe(navHazStr)

		// Extract anomaly status (parameter 13)
		anomalyStr := sp.getParameter(workLine, 13)
		sector.Anomaly = (anomalyStr == "Yes")

		// Set exploration level if not already explored or calculated
		if sector.Explored == database.EtNo || sector.Explored == database.EtCalc {
			sector.Constellation = "??? (Density only)"
			sector.Explored = database.EtDensity
			sector.UpDate = time.Now()
		}

		// Save the updated sector
		if err := sp.db.SaveSector(sector, sectorNum); err != nil {
			sp.logger.Printf("Failed to save density data for sector %d: %v", sectorNum, err)
		} else {
			sp.logger.Printf("Saved density data for sector %d: density=%d, warps=%d, navhaz=%d%%, anomaly=%t",
				sectorNum, sector.Density, sector.Warps, sector.NavHaz, sector.Anomaly)
			sp.dataLog.Printf("PARSED_DENSITY: Sector=%d, Density=%d, Warps=%d, NavHaz=%d%%, Anomaly=%t, Explored=%d",
				sectorNum, sector.Density, sector.Warps, sector.NavHaz, sector.Anomaly, int(sector.Explored))
		}
	} else {
		// Header line or other density scanner output
		sp.logger.Printf("Density scanner header: %s", line)
	}
}

// processFigScanLine processes fighter scan output
func (sp *SectorParser) processFigScanLine(line string) {
	// Fighter scan format: "Sector 1234:  800 belong to your Corp"
	// TODO: Implement complete fighter scan parsing from TWX ProcessFigScanLine
	if strings.Contains(line, "Sector ") && strings.Contains(line, ":") {
		// Extract sector number and fighter info
		parts := strings.Split(line, ":")
		if len(parts) >= 2 {
			sectorPart := strings.TrimSpace(parts[0])
			figPart := strings.TrimSpace(parts[1])

			// Extract sector number
			sectorNumStr := strings.TrimPrefix(sectorPart, "Sector ")
			sectorNum := sp.strToIntSafe(sectorNumStr)

			sp.logger.Printf("Fighter scan for sector %d: %s", sectorNum, figPart)
		}
	} else {
		sp.logger.Printf("Fighter scan header: %s", line)
	}
}

// processPortLine processes port display lines
func (sp *SectorParser) processPortLine(line string) {
	// Commerce report header: "Commerce report for Sol:"
	if len(line) >= 20 && line[:20] == "Commerce report for " {
		// Extract port name (between position 20 and first colon)
		colonPos := strings.Index(line, ":")
		if colonPos != -1 {
			sp.currentSector.SPort.Name = line[20:colonPos]
			sp.logger.Printf("Found port name: %s", sp.currentSector.SPort.Name)
		}
	} else if len(line) >= 8 && line[:8] == "Fuel Ore" && len(line) >= 33 && line[32:33] == "%" {
		// Fuel Ore line: "Fuel Ore    Selling      1,000    100%"
		sp.parseProductLine(line, database.PtFuelOre)
	} else if len(line) >= 8 && line[:8] == "Organics" && len(line) >= 33 && line[32:33] == "%" {
		// Organics line: "Organics    Buying         750     95%"  
		sp.parseProductLine(line, database.PtOrganics)
	} else if len(line) >= 9 && line[:9] == "Equipment" && len(line) >= 33 && line[32:33] == "%" {
		// Equipment line: "Equipment   Selling        500    120%"
		sp.parseProductLine(line, database.PtEquipment)

		// All products seen, finalize port data
		sp.finalizePortData()
	}
}

// parseProductLine parses product information from port reports
func (sp *SectorParser) parseProductLine(line string, productType database.TProductType) {
	// Remove % symbols for parsing
	cleanLine := strings.ReplaceAll(line, "%", "")

	// Extract product status, quantity, and percentage
	var status string
	var qty, percent int

	switch productType {
	case database.PtFuelOre:
		// "Fuel Ore    Selling      1,000    100"
		status = sp.getParameter(cleanLine, 3)
		qty = sp.strToIntSafe(strings.ReplaceAll(sp.getParameter(cleanLine, 4), ",", ""))
		percent = sp.strToIntSafe(sp.getParameter(cleanLine, 5))
	case database.PtOrganics:
		// "Organics    Buying         750     95"
		status = sp.getParameter(cleanLine, 2)
		qty = sp.strToIntSafe(strings.ReplaceAll(sp.getParameter(cleanLine, 3), ",", ""))
		percent = sp.strToIntSafe(sp.getParameter(cleanLine, 4))
	case database.PtEquipment:
		// "Equipment   Selling        500    120"
		status = sp.getParameter(cleanLine, 2)
		qty = sp.strToIntSafe(strings.ReplaceAll(sp.getParameter(cleanLine, 3), ",", ""))
		percent = sp.strToIntSafe(sp.getParameter(cleanLine, 4))
	}

	// Set buy/sell status
	if status == "Buying" {
		sp.currentSector.SPort.BuyProduct[productType] = true
	} else {
		sp.currentSector.SPort.BuyProduct[productType] = false
	}

	// Set quantity and percentage
	sp.currentSector.SPort.ProductAmount[productType] = qty
	sp.currentSector.SPort.ProductPercent[productType] = percent

	sp.logger.Printf("Port product %d: %s %d at %d%%",
		int(productType), status, qty, percent)
	sp.dataLog.Printf("PARSED_PORT_PRODUCT: Sector=%d, ProductType=%d, Status='%s', Quantity=%d, Percent=%d%%",
		sp.portSectorIndex, int(productType), status, qty, percent)
}

// finalizePortData completes port processing and determines class
func (sp *SectorParser) finalizePortData() {
	// Timestamp the port data
	sp.currentSector.SPort.UpDate = time.Now()

	// Determine port class if unknown (-1)
	if sp.currentSector.SPort.ClassIndex <= 0 {
		// Build port class string: B=Buying, S=Selling
		var portClass string

		if sp.currentSector.SPort.BuyProduct[database.PtFuelOre] {
			portClass += "B"
		} else {
			portClass += "S"
		}

		if sp.currentSector.SPort.BuyProduct[database.PtOrganics] {
			portClass += "B"
		} else {
			portClass += "S"
		}

		if sp.currentSector.SPort.BuyProduct[database.PtEquipment] {
			portClass += "B"
		} else {
			portClass += "S"
		}

		// Map class string to index (matching TWX logic)
		switch portClass {
		case "BBS":
			sp.currentSector.SPort.ClassIndex = 1
		case "BSB":
			sp.currentSector.SPort.ClassIndex = 2
		case "SBB":
			sp.currentSector.SPort.ClassIndex = 3
		case "SSB":
			sp.currentSector.SPort.ClassIndex = 4
		case "SBS":
			sp.currentSector.SPort.ClassIndex = 5
		case "BSS":
			sp.currentSector.SPort.ClassIndex = 6
		case "SSS":
			sp.currentSector.SPort.ClassIndex = 7
		case "BBB":
			sp.currentSector.SPort.ClassIndex = 8
		}

		sp.logger.Printf("Determined port class: %s -> %d", portClass, sp.currentSector.SPort.ClassIndex)
	}

	// If this is a previously unseen sector, mark as calculated
	if sp.currentSector.Explored == database.EtNo {
		sp.currentSector.Constellation = "??? (port data/calc only)"
		sp.currentSector.Explored = database.EtCalc
	}

	// Save the updated sector
	if err := sp.db.SaveSector(*sp.currentSector, sp.portSectorIndex); err != nil {
		sp.logger.Printf("Failed to save port sector %d: %v", sp.portSectorIndex, err)
	} else {
		sp.logger.Printf("Saved port data for sector %d", sp.portSectorIndex)
	}
}

// processCIMLine processes Computer Information Matrix lines
func (sp *SectorParser) processCIMLine(line string) {
	// CIM line formats depend on display type:
	// Port CIM: "1234    Sol                 BBS   100%  95%  120%"
	// Warp CIM: "1234     3     5     7     0     0     0"
	// TODO: Implement complete CIM parsing from TWX ProcessCIMLine

	if len(line) > 0 && line[0] >= '0' && line[0] <= '9' {
		// Looks like a data line starting with sector number
		sectorNum := sp.strToIntSafe(sp.getParameter(line, 1))

		if strings.Contains(line, "%") {
			// Port CIM line
			sp.logger.Printf("Port CIM data for sector %d: %s", sectorNum, line)
		} else {
			// Warp CIM line
			sp.logger.Printf("Warp CIM data for sector %d: %s", sectorNum, line)
		}
	} else {
		// Header or other CIM content
		sp.logger.Printf("CIM header: %s", line)
	}
}

// parseTraderLine parses trader information from sector display
func (sp *SectorParser) parseTraderLine(line string) {
	// Trader line format: "Traders : Gypsy in Class 0 (Cadet)'s  (Armored Mercenary)  [1,000 Fuel Ore]  [*  *] [with 50 ftrs]"
	if sp.currentTrader == nil {
		sp.currentTrader = &database.TTrader{}
	}

	// Extract trader name (everything after ": " and before " in ")
	content := line[10:] // Skip "Traders : "
	inPos := strings.Index(content, " in ")
	if inPos != -1 {
		sp.currentTrader.Name = content[:inPos]

		// Extract ship class and type
		remaining := content[inPos+4:]
		if strings.Contains(remaining, "(") && strings.Contains(remaining, ")") {
			// Extract class - "Class X (Type)"
			classStart := strings.Index(remaining, "Class ")
			if classStart != -1 {
				classSection := remaining[classStart:]
				parenStart := strings.Index(classSection, "(")
				parenEnd := strings.Index(classSection, ")")
				if parenStart != -1 && parenEnd != -1 {
					sp.currentTrader.ShipType = classSection[parenStart+1 : parenEnd]
				}
			}
		}

		// Extract fighters if present
		if strings.Contains(remaining, "with ") && strings.Contains(remaining, " ftrs") {
			ftrStart := strings.Index(remaining, "with ") + 5
			ftrEnd := strings.Index(remaining[ftrStart:], " ftrs")
			if ftrEnd != -1 {
				ftrStr := remaining[ftrStart : ftrStart+ftrEnd]
				sp.stripChar(&ftrStr, ',')
				sp.currentTrader.Figs = sp.strToIntSafe(ftrStr)
			}
		}
	}

	// Add trader to current sector
	sp.currentSector.Traders = append(sp.currentSector.Traders, *sp.currentTrader)
	sp.logger.Printf("Found trader in sector %d: %s (%s) - %d fighters",
		sp.currentSectorIndex, sp.currentTrader.Name, sp.currentTrader.ShipType, sp.currentTrader.Figs)
	sp.dataLog.Printf("PARSED_TRADER: Sector=%d, Name='%s', ShipType='%s', ShipName='%s', Fighters=%d",
		sp.currentSectorIndex, sp.currentTrader.Name, sp.currentTrader.ShipType, sp.currentTrader.ShipName, sp.currentTrader.Figs)
}

// parseTraderContinuation handles trader continuation lines (TWX: GetParameter(Line, 1) = 'in')
func (sp *SectorParser) parseTraderContinuation(line string) {
	// Line format: "in ShipName (ShipType) [*  *] [with 50 ftrs]"
	if sp.currentTrader == nil {
		sp.currentTrader = &database.TTrader{}
	}

	// Extract ship name and type from continuation
	if strings.HasPrefix(line, "in ") {
		content := line[3:] // Skip "in "
		if parenPos := strings.Index(content, "("); parenPos != -1 {
			sp.currentTrader.ShipName = strings.TrimSpace(content[:parenPos])

			if endParen := strings.Index(content[parenPos:], ")"); endParen != -1 {
				sp.currentTrader.ShipType = content[parenPos+1 : parenPos+endParen]
			}
		}

		// Extract fighter count
		if strings.Contains(content, "with ") && strings.Contains(content, " ftrs") {
			ftrStart := strings.Index(content, "with ") + 5
			ftrEnd := strings.Index(content[ftrStart:], " ftrs")
			if ftrEnd != -1 {
				ftrStr := content[ftrStart : ftrStart+ftrEnd]
				sp.stripChar(&ftrStr, ',')
				sp.currentTrader.Figs = sp.strToIntSafe(ftrStr)
			}
		}

		// Complete trader and add to sector
		sp.currentSector.Traders = append(sp.currentSector.Traders, *sp.currentTrader)
		sp.logger.Printf("Completed trader in sector %d: %s in %s (%s) - %d fighters",
			sp.currentSectorIndex, sp.currentTrader.Name, sp.currentTrader.ShipName, sp.currentTrader.ShipType, sp.currentTrader.Figs)
	}
}

// parseShipLine parses ship information from sector display  
func (sp *SectorParser) parseShipLine(line string) {
	// Ship line format from TWX: "Ships   : ShipName [Owned by] OwnerName, w/ 123 ftrs"
	// Or: "Ships   : (Nicks) in a Class 6 (Corellian Corvette)  [*  *] [with 25 ftrs, shlds down]"
	if sp.currentShip == nil {
		sp.currentShip = &database.TShip{}
	}

	content := line[10:] // Skip "Ships   : "

	// TWX-style parsing: look for [Owned by] pattern first
	if strings.Contains(content, "[Owned by]") {
		// Extract ship name (before [Owned by])
		ownedByPos := strings.Index(content, "[Owned by]")
		sp.currentShip.Name = strings.TrimSpace(content[:ownedByPos])

		// Extract owner name (after [Owned by])
		afterOwned := content[ownedByPos+10:] // Skip "[Owned by]"
		if commaPos := strings.Index(afterOwned, ","); commaPos != -1 {
			sp.currentShip.Owner = strings.TrimSpace(afterOwned[:commaPos])
		} else {
			sp.currentShip.Owner = strings.TrimSpace(afterOwned)
		}

		// Extract fighter count if present ("w/ 123 ftrs")
		if strings.Contains(afterOwned, " w/ ") && strings.Contains(afterOwned, " ftrs") {
			ftrStart := strings.Index(afterOwned, " w/ ") + 4
			ftrEnd := strings.Index(afterOwned, " ftrs")
			if ftrStart < ftrEnd {
				ftrStr := afterOwned[ftrStart:ftrEnd]
				sp.stripChar(&ftrStr, ',') // Remove commas
				sp.currentShip.Figs = sp.strToIntSafe(ftrStr)
			}
		}
	} else {
		// Legacy format: Extract ship owner name (between parentheses)
		if strings.Contains(content, "(") && strings.Contains(content, ")") {
			parenStart := strings.Index(content, "(")
			parenEnd := strings.Index(content, ")")
			if parenStart != -1 && parenEnd != -1 {
				sp.currentShip.Owner = content[parenStart+1 : parenEnd]
			}
		}

		// Extract ship class and type
		if strings.Contains(content, " in a Class ") {
			classStart := strings.Index(content, " in a Class ") + 12
			remaining := content[classStart:]
			if strings.Contains(remaining, "(") && strings.Contains(remaining, ")") {
				parenStart := strings.Index(remaining, "(")
				parenEnd := strings.Index(remaining, ")")
				if parenStart != -1 && parenEnd != -1 {
					sp.currentShip.ShipType = remaining[parenStart+1 : parenEnd]
				}
			}
		}
	}

	// Extract fighters if present
	if strings.Contains(content, "with ") && strings.Contains(content, " ftrs") {
		ftrStart := strings.Index(content, "with ") + 5
		ftrEnd := strings.Index(content[ftrStart:], " ftrs")
		if ftrEnd != -1 {
			ftrStr := content[ftrStart : ftrStart+ftrEnd]
			sp.stripChar(&ftrStr, ',')
			sp.currentShip.Figs = sp.strToIntSafe(ftrStr)
		}
	}

	// Add ship to current sector
	sp.currentSector.Ships = append(sp.currentSector.Ships, *sp.currentShip)
	sp.logger.Printf("Found ship in sector %d: Name='%s', Owner='%s', Type='%s' - %d fighters",
		sp.currentSectorIndex, sp.currentShip.Name, sp.currentShip.Owner, sp.currentShip.ShipType, sp.currentShip.Figs)
	sp.dataLog.Printf("PARSED_SHIP: Sector=%d, Name='%s', Owner='%s', Type='%s', Fighters=%d",
		sp.currentSectorIndex, sp.currentShip.Name, sp.currentShip.Owner, sp.currentShip.ShipType, sp.currentShip.Figs)
}

func (sp *SectorParser) parseFighterLine(line string) {
	content := line[10:]
	qtyStr := sp.getParameter(content, 1)
	sp.stripChar(&qtyStr, ',')
	sp.currentSector.Figs.Quantity = sp.strToIntSafe(qtyStr)

	sp.currentSector.Figs.FigType = database.FtNone
	if strings.Contains(content, "[Toll]") {
		sp.currentSector.Figs.FigType = database.FtToll
	} else if strings.Contains(content, "[Defensive]") {
		sp.currentSector.Figs.FigType = database.FtDefensive
	} else if strings.Contains(content, "[Offensive]") {
		sp.currentSector.Figs.FigType = database.FtOffensive
	}

	ownerStart := strings.Index(content, " ") + 1
	ownerText := content[ownerStart:]
	ownerText = strings.ReplaceAll(ownerText, "[Toll] ", "")
	ownerText = strings.ReplaceAll(ownerText, "[Defensive] ", "")
	ownerText = strings.ReplaceAll(ownerText, "[Offensive] ", "")
	sp.currentSector.Figs.Owner = strings.TrimSpace(ownerText)

	if sp.currentSector.Figs.Owner == "Personal" {
		sp.currentSector.Figs.Owner = "yours"
	}

	figTypeStr := "None"
	switch sp.currentSector.Figs.FigType {
	case database.FtToll:
		figTypeStr = "Toll"
	case database.FtDefensive:
		figTypeStr = "Defensive"
	case database.FtOffensive:
		figTypeStr = "Offensive"
	}

	sp.logger.Printf("Found fighters in sector %d: %d %s (%s)",
		sp.currentSectorIndex, sp.currentSector.Figs.Quantity, figTypeStr, sp.currentSector.Figs.Owner)
	sp.dataLog.Printf("PARSED_FIGHTERS: Sector=%d, Quantity=%d, Type=%s, Owner='%s'",
		sp.currentSectorIndex, sp.currentSector.Figs.Quantity, figTypeStr, sp.currentSector.Figs.Owner)
}

// parseMineLine parses mine information using TWX exact logic
func (sp *SectorParser) parseMineLine(line string) {
	// Mine line format: "Mines   : 5 Armid mines (Mines belong to the Admiral)"
	// TWX logic: GetParameter(Line, 6) = 'Armid)' to detect mine type
	sp.sectorPosition = SpMines

	// Extract mine quantity (parameter 3 like TWX)
	qtyStr := sp.getParameter(line, 3)
	sp.stripChar(&qtyStr, ',')
	quantity := sp.strToIntSafe(qtyStr)

	// Extract owner (TWX: Copy(Line, GetParameterPos(Line, 7) + 1, length(Line) - I))
	ownerStart := sp.getParameterPos(line, 7)
	if ownerStart > 0 && ownerStart < len(line) {
		owner := line[ownerStart:]
		// Remove parentheses if present
		owner = strings.TrimPrefix(owner, "(")
		owner = strings.TrimSuffix(owner, ")")

		// TWX logic: check parameter 6 for mine type detection
		param6 := sp.getParameter(line, 6)
		if param6 == "Armid)" {
			// Armid mines
			sp.currentSector.MinesArmid.Quantity = quantity
			sp.currentSector.MinesArmid.Owner = owner
			sp.logger.Printf("Found Armid mines in sector %d: %d (%s)",
				sp.currentSectorIndex, quantity, owner)
			sp.dataLog.Printf("PARSED_MINES: Sector=%d, Type=Armid, Quantity=%d, Owner='%s'",
				sp.currentSectorIndex, quantity, owner)
		} else {
			// Limpet mines (default)
			sp.currentSector.MinesLimpet.Quantity = quantity
			sp.currentSector.MinesLimpet.Owner = owner
			sp.logger.Printf("Found Limpet mines in sector %d: %d (%s)",
				sp.currentSectorIndex, quantity, owner)
			sp.dataLog.Printf("PARSED_MINES: Sector=%d, Type=Limpet, Quantity=%d, Owner='%s'",
				sp.currentSectorIndex, quantity, owner)
		}
	}
}

// getParameterPos returns the starting position of the nth parameter (1-indexed like TWX)
func (sp *SectorParser) getParameterPos(line string, paramNum int) int {
	fields := strings.Fields(line)
	if paramNum > 0 && paramNum <= len(fields) {
		// Find the position of the paramNum-th field in the original line
		fieldIndex := 0
		for i, field := range fields {
			if i == paramNum-1 {
				return strings.Index(line, field) + fieldIndex
			}
			fieldIndex = strings.Index(line[fieldIndex:], field) + fieldIndex + len(field)
		}
	}
	return -1
}

// processTransmissionLine handles transmission parsing (TWX pattern)
func (sp *SectorParser) processTransmissionLine(line string) {
	// TWX logic: GetParameterPos(Line, 4) for message start position
	messageStart := sp.getParameterPos(line, 4)
	if messageStart > 0 && messageStart < len(line) {
		// Check for comm-link ending (TWX pattern)
		if strings.HasSuffix(line, "comm-link:") {
			// Extract sender name before "on Federation"
			if strings.Contains(line, " on Federation") {
				senderEnd := strings.Index(line, " on Federation")
				if senderEnd > messageStart {
					sender := line[messageStart:senderEnd]
					sp.logger.Printf("Found comm-link transmission from: %s", sender)
					// TWX stores this as current message for further processing
				}
			}
		} else {
			// Extract message content after parameter 4
			message := strings.TrimSpace(line[messageStart:])
			sp.logger.Printf("Found transmission: %s", message)
		}
	}
}

// processStarDockLine handles StarDock detection from 'V' screen (TWX pattern)
func (sp *SectorParser) processStarDockLine(line string) {
	// TWX captures StarDock from the 'V' Screen
	// Beacon & Constellation are assumed, but will be updated when sector is finally visited
	sp.logger.Printf("Found StarDock reference: %s", line)

	// Extract sector number if present in the line
	// This is logged for future reference but not immediately processed
	// as the actual sector data will come when visiting the sector
}

// processContinuationLine handles continuation lines based on current sector position
func (sp *SectorParser) processContinuationLine(line string) {
	// Trim leading spaces for 8-space continuation lines
	trimmed := strings.TrimLeft(line, " ")

	// Handle multi-line content based on current position (like TWX)
	switch sp.sectorPosition {
	case SpPorts:
		// Port build time continuation data (like TWX GetParameter(Line, 4))
		buildTime := sp.strToIntSafe(sp.getParameter(line, 4))
		if buildTime > 0 {
			sp.currentSector.SPort.BuildTime = buildTime
			sp.logger.Printf("Found port build time in sector %d: %d", sp.currentSectorIndex, buildTime)
		}
	case SpPlanets:
		// Additional planet data on continuation lines
		if trimmed != "" {
			planetName := trimmed
			planet := database.TPlanet{Name: planetName}
			sp.currentSector.Planets = append(sp.currentSector.Planets, planet)
			sp.logger.Printf("Found additional planet in sector %d: %s", sp.currentSectorIndex, planetName)
		}
	case SpTraders:
		// Additional trader data (TWX: GetParameter(Line, 1) = 'in')
		if strings.HasPrefix(trimmed, "in ") || strings.Contains(trimmed, " in ") {
			// Continue working on trader - ship name and type
			sp.parseTraderContinuation(trimmed)
		} else if trimmed != "" {
			// New trader on continuation line
			sp.parseTraderLine("Traders : " + trimmed)
		}
	case SpShips:
		// Additional ship data or new ships on continuation lines
		if strings.Contains(trimmed, " in ") || strings.Contains(trimmed, "[Owned by]") {
			// New ship on continuation line
			sp.parseShipLine("Ships   : " + trimmed)
		}
	case SpMines:
		// Additional mine data (TWX: Limpet mines continuation)
		if strings.Contains(trimmed, "Limpet") {
			sp.currentSector.MinesLimpet.Quantity = sp.strToIntSafe(sp.getParameter(line, 2))
			sp.currentSector.MinesLimpet.Owner = strings.Join(strings.Fields(line)[5:], " ")
			sp.logger.Printf("Found Limpet mines in sector %d: %d owned by %s",
				sp.currentSectorIndex, sp.currentSector.MinesLimpet.Quantity, sp.currentSector.MinesLimpet.Owner)
		}
	default:
		// Normal position - no special handling
		break
	}
}

// processQuickStats processes player status QuickStats lines  
func (sp *SectorParser) processQuickStats(line string) {
	// QuickStats format: " Turns 150♫Credits 50,000♫Ship Imperial Starship♫..."
	// TODO: Implement complete QuickStats parsing from TWX ProcessQuickStats
	if len(line) > 0 && line[0] == ' ' {
		// Parse QuickStats data (contains ♫ separators)
		content := line[1:] // Remove leading space
		if strings.Contains(content, "♫") {
			parts := strings.Split(content, "♫")
			for _, part := range parts {
				if strings.Contains(part, " ") {
					// Extract key-value pairs
					fields := strings.Fields(part)
					if len(fields) >= 2 {
						key := fields[0]
						value := strings.Join(fields[1:], " ")
						sp.logger.Printf("QuickStats: %s = %s", key, value)
					}
				}
			}
		}
	} else {
		sp.logger.Printf("QuickStats header: %s", line)
	}
}

// ProcessData processes a chunk of data for sector information
func (sp *SectorParser) ProcessData(data []byte) {
	text := string(data)
	sp.ParseText(text)
}
