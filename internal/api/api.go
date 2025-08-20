package api

import "time"

// Enums for type safety
type ProductType int

const (
	ProductTypeFuelOre ProductType = iota
	ProductTypeOrganics
	ProductTypeEquipment
)

func (pt ProductType) String() string {
	switch pt {
	case ProductTypeFuelOre:
		return "FuelOre"
	case ProductTypeOrganics:
		return "Organics"
	case ProductTypeEquipment:
		return "Equipment"
	default:
		return "Unknown"
	}
}

type ProductStatus int

const (
	ProductStatusNone ProductStatus = iota
	ProductStatusBuying
	ProductStatusSelling
)

func (ps ProductStatus) String() string {
	switch ps {
	case ProductStatusNone:
		return "None"
	case ProductStatusBuying:
		return "Buying"
	case ProductStatusSelling:
		return "Selling"
	default:
		return "Unknown"
	}
}

type PortClass int

const (
	PortClassBBS PortClass = iota + 1 // Buy Buy Sell
	PortClassBSB                      // Buy Sell Buy
	PortClassSBB                      // Sell Buy Buy
	PortClassSSB                      // Sell Sell Buy
	PortClassSBS                      // Sell Buy Sell
	PortClassBSS                      // Buy Sell Sell
	PortClassSSS                      // Sell Sell Sell
	PortClassBBB                      // Buy Buy Buy
	PortClassSTD                      // Stardock
)

func (pc PortClass) String() string {
	switch pc {
	case PortClassBBS:
		return "BBS"
	case PortClassBSB:
		return "BSB"
	case PortClassSBB:
		return "SBB"
	case PortClassSSB:
		return "SSB"
	case PortClassSBS:
		return "SBS"
	case PortClassBSS:
		return "BSS"
	case PortClassSSS:
		return "SSS"
	case PortClassBBB:
		return "BBB"
	case PortClassSTD:
		return "STD"
	default:
		return ""
	}
}

type PortInfo struct {
	SectorID   int           `json:"sector_id"`
	Name       string        `json:"name"`
	Class      int           `json:"class"`
	ClassType  PortClass     `json:"class_type"`
	BuildTime  int           `json:"build_time"`
	Products   []ProductInfo `json:"products"`
	LastUpdate time.Time     `json:"last_update"`
	Dead       bool          `json:"dead"`
}

type ProductInfo struct {
	Type       ProductType   `json:"type"`
	Status     ProductStatus `json:"status"`
	Quantity   int           `json:"quantity"`
	Percentage int           `json:"percentage"`
}

// ProxyAPI defines commands from TUI to Proxy
type ProxyAPI interface {
	// Connection Management
	Disconnect() error
	IsConnected() bool

	// Data Processing (symmetric with OnData)
	SendData(data []byte) error

	// Script Management (Phase 3)
	LoadScript(filename string) error
	StopAllScripts() error
	GetScriptStatus() ScriptStatusInfo

	// Game State Management (Phase 4)
	GetCurrentSector() (int, error)
	GetSectorInfo(sectorNum int) (SectorInfo, error)
	GetPlayerInfo() (PlayerInfo, error)

	// Port Information (Phase 2)
	GetPortInfo(sectorNum int) (*PortInfo, error)

	// Player Statistics
	GetPlayerStats() (*PlayerStatsInfo, error)
}

// TuiAPI defines notifications from Proxy to TUI
//
// CRITICAL: All methods must return immediately (within microseconds) to avoid
// blocking the proxy. Use goroutines for any actual work and queue UI updates
// through tview's QueueUpdateDraw mechanism.
type TuiAPI interface {
	// Connection Events - single callback for all status changes
	OnConnectionStatusChanged(status ConnectionStatus, address string)
	OnConnectionError(err error)

	// Data Events - must return immediately (high frequency calls)
	OnData(data []byte)

	// Script Events (Phase 3)
	OnScriptStatusChanged(status ScriptStatusInfo)
	OnScriptError(scriptName string, err error)

	// Database Events - called when game databases are loaded/unloaded
	OnDatabaseStateChanged(info DatabaseStateInfo)

	// Game State Events (Phase 4.3 - MINIMAL)
	OnCurrentSectorChanged(sectorInfo SectorInfo) // Sector change callback with full sector information

	// Trader and Player Info Events - called when trader data or player stats are updated
	OnTraderDataUpdated(sectorNumber int, traders []TraderInfo) // Trader information captured from sector display
	OnPlayerStatsUpdated(stats PlayerStatsInfo)                 // Player statistics updated from QuickStats or inventory commands

	// Port Events - called when port information is updated
	OnPortUpdated(portInfo PortInfo) // Port information updated from parsing

	// Sector Events - called when sector data is updated (e.g. from etherprobe)
	OnSectorUpdated(sectorInfo SectorInfo) // Sector information updated from parsing or probe data
}

// ConnectionStatus represents the current connection state
type ConnectionStatus int

const (
	ConnectionStatusDisconnected ConnectionStatus = iota
	ConnectionStatusConnecting
	ConnectionStatusConnected
)

func (cs ConnectionStatus) String() string {
	switch cs {
	case ConnectionStatusDisconnected:
		return "disconnected"
	case ConnectionStatusConnecting:
		return "connecting"
	case ConnectionStatusConnected:
		return "connected"
	default:
		return "unknown"
	}
}

// ScriptStatusInfo provides basic script information for Phase 3
type ScriptStatusInfo struct {
	ActiveCount int      `json:"active_count"` // Number of running scripts
	TotalCount  int      `json:"total_count"`  // Total number of loaded scripts
	ScriptNames []string `json:"script_names"` // Names of loaded scripts
}

// PlayerInfo provides basic player information for Phase 4
type PlayerInfo struct {
	Name          string `json:"name"`           // Player name (if available)
	CurrentSector int    `json:"current_sector"` // Current sector location
}

// SectorInfo provides basic sector information for panel display
type SectorInfo struct {
	Number        int    `json:"number"`             // Sector number
	NavHaz        int    `json:"nav_haz"`            // Navigation hazard level
	HasTraders    int    `json:"has_traders"`        // Number of traders present
	Constellation string `json:"constellation"`      // Constellation name
	Beacon        string `json:"beacon"`             // Beacon text
	Warps         []int  `json:"warps"`              // Warp connections to other sectors
	HasPort       bool   `json:"has_port,omitempty"` // True if sector has a port
	Visited       bool   `json:"visited"`            // True only if sector has been actually visited (EtHolo)
}

// DatabaseStateInfo provides information about database loading/unloading
type DatabaseStateInfo struct {
	GameName     string `json:"game_name"`     // Name of the game (e.g., "Trade Wars 2002")
	ServerHost   string `json:"server_host"`   // Server host (e.g., "twgs.geekm0nkey.com")
	ServerPort   string `json:"server_port"`   // Server port (e.g., "23")
	DatabaseName string `json:"database_name"` // Database filename
	IsLoaded     bool   `json:"is_loaded"`     // true when database is loaded, false when unloaded
}

// TraderInfo represents trader information for TUI API
type TraderInfo struct {
	Name      string `json:"name"`      // Trader name
	ShipName  string `json:"ship_name"` // Ship name (e.g., "USS Enterprise")
	ShipType  string `json:"ship_type"` // Ship type (e.g., "Imperial StarShip")
	Fighters  int    `json:"fighters"`  // Number of fighters
	Alignment string `json:"alignment"` // Alignment (Good, Evil, Neutral, etc.)
}

// PlayerStatsInfo represents current player statistics for TUI API
type PlayerStatsInfo struct {
	Turns         int    `json:"turns"`          // Turns remaining
	Credits       int    `json:"credits"`        // Credits
	Fighters      int    `json:"fighters"`       // Fighters
	Shields       int    `json:"shields"`        // Shield strength
	TotalHolds    int    `json:"total_holds"`    // Total cargo holds
	OreHolds      int    `json:"ore_holds"`      // Ore holds
	OrgHolds      int    `json:"org_holds"`      // Organics holds
	EquHolds      int    `json:"equ_holds"`      // Equipment holds
	ColHolds      int    `json:"col_holds"`      // Colonists holds
	Photons       int    `json:"photons"`        // Photon torpedoes
	Armids        int    `json:"armids"`         // Armid mines
	Limpets       int    `json:"limpets"`        // Limpet mines
	GenTorps      int    `json:"gen_torps"`      // Genesis torpedoes
	TwarpType     int    `json:"twarp_type"`     // TransWarp type
	Cloaks        int    `json:"cloaks"`         // Cloaking devices
	Beacons       int    `json:"beacons"`        // Beacons
	Atomics       int    `json:"atomics"`        // Atomic detonators
	Corbomite     int    `json:"corbomite"`      // Corbomite devices
	Eprobes       int    `json:"eprobes"`        // Ether probes
	MineDisr      int    `json:"mine_disr"`      // Mine disruptors
	Alignment     int    `json:"alignment"`      // Alignment value
	Experience    int    `json:"experience"`     // Experience points
	Corp          int    `json:"corp"`           // Corporation number
	ShipNumber    int    `json:"ship_number"`    // Ship number
	ShipClass     string `json:"ship_class"`     // Ship class (e.g., "MerCru")
	PsychicProbe  bool   `json:"psychic_probe"`  // Has psychic probe
	PlanetScanner bool   `json:"planet_scanner"` // Has planet scanner
	ScanType      int    `json:"scan_type"`      // Long range scanner type
	CurrentSector int    `json:"current_sector"` // Current sector number
	PlayerName    string `json:"player_name"`    // Player name
}
