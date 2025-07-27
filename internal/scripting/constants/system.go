package constants

import (
	"fmt"
	"strconv"
	"time"

	"twist/internal/scripting/types"
)

// SystemConstants manages TWX system constants
type SystemConstants struct {
	gameInterface types.GameInterface
	constants     map[string]*types.Value
}

// NewSystemConstants creates a new system constants manager
func NewSystemConstants(gameInterface types.GameInterface) *SystemConstants {
	sc := &SystemConstants{
		gameInterface: gameInterface,
		constants:     make(map[string]*types.Value),
	}
	
	sc.initializeConstants()
	return sc
}

// GetConstant returns the value of a system constant
func (sc *SystemConstants) GetConstant(name string) (*types.Value, bool) {
	// Check if it's a dynamic constant that needs updating
	sc.updateDynamicConstant(name)
	
	value, exists := sc.constants[name]
	if exists {
		return value.Clone(), true
	}
	return nil, false
}

// initializeConstants initializes all system constants
func (sc *SystemConstants) initializeConstants() {
	// ANSI Color Constants (16 constants)
	sc.constants["ANSI_0"] = types.NewStringValue("\x1b[0;30m")   // Black
	sc.constants["ANSI_1"] = types.NewStringValue("\x1b[0;34m")   // Blue
	sc.constants["ANSI_2"] = types.NewStringValue("\x1b[0;32m")   // Green
	sc.constants["ANSI_3"] = types.NewStringValue("\x1b[0;36m")   // Cyan
	sc.constants["ANSI_4"] = types.NewStringValue("\x1b[0;31m")   // Red
	sc.constants["ANSI_5"] = types.NewStringValue("\x1b[0;35m")   // Magenta
	sc.constants["ANSI_6"] = types.NewStringValue("\x1b[0;33m")   // Brown
	sc.constants["ANSI_7"] = types.NewStringValue("\x1b[0;37m")   // Light Gray
	sc.constants["ANSI_8"] = types.NewStringValue("\x1b[1;30m")   // Dark Gray
	sc.constants["ANSI_9"] = types.NewStringValue("\x1b[1;34m")   // Light Blue
	sc.constants["ANSI_10"] = types.NewStringValue("\x1b[1;32m")  // Light Green
	sc.constants["ANSI_11"] = types.NewStringValue("\x1b[1;36m")  // Light Cyan
	sc.constants["ANSI_12"] = types.NewStringValue("\x1b[1;31m")  // Light Red
	sc.constants["ANSI_13"] = types.NewStringValue("\x1b[1;35m")  // Light Magenta
	sc.constants["ANSI_14"] = types.NewStringValue("\x1b[1;33m")  // Yellow
	sc.constants["ANSI_15"] = types.NewStringValue("\x1b[1;37m")  // White
	
	// Boolean Constants
	sc.constants["TRUE"] = types.NewNumberValue(1)
	sc.constants["FALSE"] = types.NewNumberValue(0)
	
	// System Information Constants
	sc.constants["VERSION"] = types.NewStringValue("3.09")  // TWX version compatibility
	sc.constants["GAME"] = types.NewStringValue("TradeWars 2002")
	sc.constants["GAMENAME"] = types.NewStringValue("TradeWars 2002")
	
	// Dynamic constants (will be updated on access)
	sc.constants["CONNECTED"] = types.NewNumberValue(1)  // Assume connected
	sc.constants["CURRENTLINE"] = types.NewStringValue("")
	sc.constants["CURRENTANSILINE"] = types.NewStringValue("")
	sc.constants["DATE"] = types.NewStringValue("")
	sc.constants["TIME"] = types.NewStringValue("")
	sc.constants["CURRENTSECTOR"] = types.NewNumberValue(1)
	sc.constants["RAWPACKET"] = types.NewStringValue("")
	
	// Game Location Constants
	sc.constants["SECTORS"] = types.NewNumberValue(5000)  // Default game size
	sc.constants["STARDOCK"] = types.NewNumberValue(1)
	sc.constants["ALPHACENTAURI"] = types.NewNumberValue(2)
	sc.constants["RYLOS"] = types.NewNumberValue(3)
	
	// Player Status Constants (will be dynamic in real implementation)
	sc.constants["TURNS"] = types.NewNumberValue(0)
	sc.constants["CREDITS"] = types.NewNumberValue(0)
	sc.constants["FIGHTERS"] = types.NewNumberValue(0)
	sc.constants["SHIELDS"] = types.NewNumberValue(0)
	sc.constants["TOTALHOLDS"] = types.NewNumberValue(0)
	sc.constants["OREHOLDS"] = types.NewNumberValue(0)
	sc.constants["ORGHOLDS"] = types.NewNumberValue(0)
	sc.constants["EQUHOLDS"] = types.NewNumberValue(0)
	sc.constants["COLHOLDS"] = types.NewNumberValue(0)
	sc.constants["EMPTYHOLDS"] = types.NewNumberValue(0)
	sc.constants["PHOTONS"] = types.NewNumberValue(0)
	sc.constants["ARMIDS"] = types.NewNumberValue(0)
	sc.constants["LIMPETS"] = types.NewNumberValue(0)
	sc.constants["GENTORPS"] = types.NewNumberValue(0)
	sc.constants["TWARPTYPE"] = types.NewNumberValue(0)
	sc.constants["CLOAKS"] = types.NewNumberValue(0)
	sc.constants["BEACONS"] = types.NewNumberValue(0)
	sc.constants["ATOMICS"] = types.NewNumberValue(0)
	sc.constants["CORBOMITE"] = types.NewNumberValue(0)
	sc.constants["EPROBES"] = types.NewNumberValue(0)
	sc.constants["MINEDISR"] = types.NewNumberValue(0)
	sc.constants["PSYCHICPROBE"] = types.NewNumberValue(0)
	sc.constants["PLANETSCANNER"] = types.NewNumberValue(0)
	sc.constants["SCANTYPE"] = types.NewNumberValue(0)
	sc.constants["ALIGNMENT"] = types.NewNumberValue(0)
	sc.constants["EXPERIENCE"] = types.NewNumberValue(0)
	sc.constants["CORP"] = types.NewNumberValue(0)
	sc.constants["SHIPNUMBER"] = types.NewNumberValue(0)
	sc.constants["SHIPCLASS"] = types.NewStringValue("Unknown")
	
	// Sector Information Constants (dynamic, based on current sector)
	sc.constants["SECTOR.WARPS"] = types.NewStringValue("")
	sc.constants["SECTOR.WARPCOUNT"] = types.NewNumberValue(0)
	sc.constants["SECTOR.WARPSEIN"] = types.NewStringValue("")
	sc.constants["SECTOR.WARPINCOUNT"] = types.NewNumberValue(0)
	sc.constants["SECTOR.BEACON"] = types.NewStringValue("")
	sc.constants["SECTOR.CONSTELLATION"] = types.NewStringValue("")
	sc.constants["SECTOR.DENSITY"] = types.NewNumberValue(-1)
	sc.constants["SECTOR.NAVHAZ"] = types.NewNumberValue(0)
	sc.constants["SECTOR.EXPLORED"] = types.NewNumberValue(0)
	sc.constants["SECTOR.ANOMALY"] = types.NewNumberValue(0)
	sc.constants["SECTOR.DEADEND"] = types.NewNumberValue(0)
	sc.constants["SECTOR.FIGS.OWNER"] = types.NewStringValue("")
	sc.constants["SECTOR.FIGS.QUANTITY"] = types.NewNumberValue(0)
	sc.constants["SECTOR.FIGS.TYPE"] = types.NewStringValue("")
	sc.constants["SECTOR.MINES.OWNER"] = types.NewStringValue("")
	sc.constants["SECTOR.MINES.QUANTITY"] = types.NewNumberValue(0)
	sc.constants["SECTOR.LIMPETS.OWNER"] = types.NewStringValue("")
	sc.constants["SECTOR.LIMPETS.QUANTITY"] = types.NewNumberValue(0)
	sc.constants["SECTOR.SHIPS"] = types.NewStringValue("")
	sc.constants["SECTOR.TRADERS"] = types.NewStringValue("")
	sc.constants["SECTOR.PLANETS"] = types.NewStringValue("")
	sc.constants["SECTOR.PLANETCOUNT"] = types.NewNumberValue(0)
	sc.constants["SECTOR.SHIPCOUNT"] = types.NewNumberValue(0)
	sc.constants["SECTOR.TRADERCOUNT"] = types.NewNumberValue(0)
	sc.constants["SECTOR.UPDATED"] = types.NewStringValue("")
	
	// Port Information Constants (dynamic, based on current sector)
	sc.constants["PORT.EXISTS"] = types.NewNumberValue(0)
	sc.constants["PORT.NAME"] = types.NewStringValue("")
	sc.constants["PORT.CLASS"] = types.NewNumberValue(0)
	sc.constants["PORT.FUEL"] = types.NewNumberValue(0)
	sc.constants["PORT.ORG"] = types.NewNumberValue(0)
	sc.constants["PORT.EQUIP"] = types.NewNumberValue(0)
	sc.constants["PORT.PERCENTFUEL"] = types.NewNumberValue(0)
	sc.constants["PORT.PERCENTORG"] = types.NewNumberValue(0)
	sc.constants["PORT.PERCENTEQUIP"] = types.NewNumberValue(0)
	sc.constants["PORT.BUILDTIME"] = types.NewNumberValue(0)
	sc.constants["PORT.UPDATED"] = types.NewStringValue("")
	sc.constants["PORT.BUYFUEL"] = types.NewNumberValue(0)
	sc.constants["PORT.BUYORG"] = types.NewNumberValue(0)
	sc.constants["PORT.BUYEQUIP"] = types.NewNumberValue(0)
	
	// Bot System Constants (for multi-bot support)
	sc.constants["ACTIVEBOT"] = types.NewStringValue("Default")
	sc.constants["ACTIVEBOTS"] = types.NewNumberValue(1)
	sc.constants["ACTIVEBOTDIR"] = types.NewStringValue("")
	sc.constants["ACTIVEBOTSCRIPT"] = types.NewStringValue("")
	sc.constants["ACTIVEBOTNAME"] = types.NewStringValue("Default")
	sc.constants["BOTLIST"] = types.NewStringValue("Default")
	sc.constants["GAMEDATA"] = types.NewStringValue("")
	
	// Library System Constants (for advanced scripting)
	for i := 0; i < 20; i++ { // Expand to 20 LIBPARM slots
		sc.constants[fmt.Sprintf("LIBPARM[%d]", i)] = types.NewStringValue("")
	}
	sc.constants["LIBPARMS"] = types.NewStringValue("")
	sc.constants["LIBPARMCOUNT"] = types.NewNumberValue(0)
	sc.constants["LIBSUBSPACE"] = types.NewNumberValue(0)
	sc.constants["LIBSILENT"] = types.NewNumberValue(0)
	sc.constants["LIBMULTILINE"] = types.NewNumberValue(0)
	sc.constants["LIBMSG"] = types.NewStringValue("")
	
	// Additional TWX System Constants
	sc.constants["TWXVERSION"] = types.NewStringValue("3.09")
	sc.constants["TWXBUILD"] = types.NewStringValue("20240726")
	sc.constants["SCRIPTERROR"] = types.NewStringValue("")
	sc.constants["SCRIPTLINE"] = types.NewNumberValue(0)
	sc.constants["DEBUGMODE"] = types.NewNumberValue(0)
	sc.constants["MAXTRIGGERS"] = types.NewNumberValue(1000)
	sc.constants["ACTIVETRIGGERS"] = types.NewNumberValue(0)
	sc.constants["GAMESEEDS"] = types.NewNumberValue(0)
	sc.constants["HOLODETECTOR"] = types.NewNumberValue(0)
	sc.constants["PHOTONDISRUPTOR"] = types.NewNumberValue(0)
	sc.constants["ETHER_PROBE"] = types.NewNumberValue(0)
	sc.constants["GENESIS_TORPEDO"] = types.NewNumberValue(0)
	sc.constants["DENSITYSCANNER"] = types.NewNumberValue(0)
	sc.constants["HOLOSCANNER"] = types.NewNumberValue(0)
	
	// Quick Status Constants
	sc.constants["QUICKSTATS"] = types.NewStringValue("")
	sc.constants["ANSIQUICKSTATS"] = types.NewStringValue("")
	sc.constants["QS"] = types.NewStringValue("")
	sc.constants["QSTAT"] = types.NewStringValue("")
	
	// Login/Authentication Constants
	sc.constants["LICENSENAME"] = types.NewStringValue("")
	sc.constants["LOGINNAME"] = types.NewStringValue("")
	sc.constants["PASSWORD"] = types.NewStringValue("")
}

// updateDynamicConstant updates constants that change based on game state
func (sc *SystemConstants) updateDynamicConstant(name string) {
	switch name {
	case "DATE":
		sc.constants["DATE"] = types.NewStringValue(time.Now().Format("01/02/2006"))
	case "TIME":
		sc.constants["TIME"] = types.NewStringValue(time.Now().Format("15:04:05"))
	case "CURRENTSECTOR":
		if sc.gameInterface != nil {
			currentSector := sc.gameInterface.GetCurrentSector()
			sc.constants["CURRENTSECTOR"] = types.NewNumberValue(float64(currentSector))
		}
	case "SECTOR.WARPS", "SECTOR.WARPCOUNT", "SECTOR.DENSITY", "SECTOR.NAVHAZ", 
		 "SECTOR.EXPLORED", "SECTOR.ANOMALY", "SECTOR.BEACON", "SECTOR.CONSTELLATION":
		sc.updateSectorConstants()
	case "PORT.EXISTS", "PORT.NAME", "PORT.CLASS", "PORT.FUEL", "PORT.ORG", "PORT.EQUIP":
		sc.updatePortConstants()
	}
}

// updateSectorConstants updates sector-related constants
func (sc *SystemConstants) updateSectorConstants() {
	if sc.gameInterface == nil {
		return
	}
	
	currentSector := sc.gameInterface.GetCurrentSector()
	sectorData, err := sc.gameInterface.GetSector(currentSector)
	if err != nil {
		return
	}
	
	// Update sector warps
	warpStr := ""
	warpCount := 0
	for _, warp := range sectorData.Warps {
		if warp > 0 {
			if warpStr != "" {
				warpStr += " "
			}
			warpStr += strconv.Itoa(warp)
			warpCount++
		}
	}
	sc.constants["SECTOR.WARPS"] = types.NewStringValue(warpStr)
	sc.constants["SECTOR.WARPCOUNT"] = types.NewNumberValue(float64(warpCount))
	
	// Update other sector info
	sc.constants["SECTOR.DENSITY"] = types.NewNumberValue(float64(sectorData.Density))
	sc.constants["SECTOR.NAVHAZ"] = types.NewNumberValue(float64(sectorData.NavHaz))
	sc.constants["SECTOR.EXPLORED"] = types.NewNumberValue(float64(sectorData.Explored))
	sc.constants["SECTOR.BEACON"] = types.NewStringValue(sectorData.Beacon)
	sc.constants["SECTOR.CONSTELLATION"] = types.NewStringValue(sectorData.Constellation)
	
	if sectorData.Anomaly {
		sc.constants["SECTOR.ANOMALY"] = types.NewNumberValue(1)
	} else {
		sc.constants["SECTOR.ANOMALY"] = types.NewNumberValue(0)
	}
	
	// Update ship/trader/planet counts
	sc.constants["SECTOR.SHIPCOUNT"] = types.NewNumberValue(float64(len(sectorData.Ships)))
	sc.constants["SECTOR.TRADERCOUNT"] = types.NewNumberValue(float64(len(sectorData.Traders)))
	sc.constants["SECTOR.PLANETCOUNT"] = types.NewNumberValue(float64(len(sectorData.Planets)))
}

// updatePortConstants updates port-related constants
func (sc *SystemConstants) updatePortConstants() {
	if sc.gameInterface == nil {
		return
	}
	
	currentSector := sc.gameInterface.GetCurrentSector()
	sectorData, err := sc.gameInterface.GetSector(currentSector)
	if err != nil {
		return
	}
	
	if sectorData.HasPort {
		sc.constants["PORT.EXISTS"] = types.NewNumberValue(1)
		sc.constants["PORT.NAME"] = types.NewStringValue(sectorData.PortName)
		sc.constants["PORT.CLASS"] = types.NewNumberValue(float64(sectorData.PortClass))
	} else {
		sc.constants["PORT.EXISTS"] = types.NewNumberValue(0)
		sc.constants["PORT.NAME"] = types.NewStringValue("")
		sc.constants["PORT.CLASS"] = types.NewNumberValue(0)
	}
}

// ListConstants returns all available constants
func (sc *SystemConstants) ListConstants() []string {
	constants := make([]string, 0, len(sc.constants))
	for name := range sc.constants {
		constants = append(constants, name)
	}
	return constants
}

// GetConstantCount returns the total number of constants
func (sc *SystemConstants) GetConstantCount() int {
	return len(sc.constants)
}

// UpdateCurrentLine updates the current line constants
func (sc *SystemConstants) UpdateCurrentLine(text string) {
	sc.constants["CURRENTLINE"] = types.NewStringValue(text)
	
	// Strip ANSI codes for CURRENTANSILINE (simple implementation)
	ansiText := text // TODO: Implement proper ANSI stripping
	sc.constants["CURRENTANSILINE"] = types.NewStringValue(ansiText)
	
	// Update raw packet if needed
	sc.constants["RAWPACKET"] = types.NewStringValue(text)
}