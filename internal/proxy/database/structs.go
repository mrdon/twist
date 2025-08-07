package database

import "time"

// Enums matching TWX exactly

// TFighterType matches TWX TFighterType
type TFighterType int

const (
	FtToll TFighterType = iota
	FtDefensive
	FtOffensive
	FtNone
)

// TSectorExploredType matches TWX TSectorExploredType
type TSectorExploredType int

const (
	EtNo TSectorExploredType = iota
	EtCalc
	EtDensity
	EtHolo
)

// TProductType matches TWX TProductType
type TProductType int

const (
	PtFuelOre TProductType = iota
	PtOrganics
	PtEquipment
)

// Core structs matching TWX records exactly

// TSpaceObject matches TWX TSpaceObject record
type TSpaceObject struct {
	Quantity int           `json:"quantity"`
	Owner    string        `json:"owner"`    // string[40] in TWX
	FigType  TFighterType  `json:"fig_type"`
}

// TPort matches TWX TPort record  
type TPort struct {
	Name           string               `json:"name"`           // string[40] in TWX
	Dead           bool                 `json:"dead"`
	BuildTime      int                  `json:"build_time"`     // Byte in TWX
	ClassIndex     int                  `json:"class_index"`    // Byte in TWX
	BuyProduct     [3]bool              `json:"buy_product"`    // array[TProductType] of Boolean
	ProductPercent [3]int               `json:"product_percent"` // array[TProductType] of Byte
	ProductAmount  [3]int               `json:"product_amount"`  // array[TProductType] of Word
	UpDate         time.Time            `json:"update"`
}

// TSector matches TWX TSector record with Phase 2 optimization (port data separated)
type TSector struct {
	// Warp is array[1..6] in TWX, we'll use [6] and handle 1-indexing in code
	Warp          [6]int               `json:"warp"`           
	// SPort removed - now in separate ports table
	NavHaz        int                  `json:"nav_haz"`        // Byte in TWX
	Figs          TSpaceObject         `json:"figs"`
	MinesArmid    TSpaceObject         `json:"mines_armid"`
	MinesLimpet   TSpaceObject         `json:"mines_limpet"`
	Constellation string               `json:"constellation"`  // string[40] in TWX
	Beacon        string               `json:"beacon"`        // string[40] in TWX
	UpDate        time.Time            `json:"update"`
	Anomaly       bool                 `json:"anomaly"`
	Density       int                  `json:"density"`       // LongInt in TWX
	Warps         int                  `json:"warps"`         // Byte in TWX (number of valid warps)
	Explored      TSectorExploredType  `json:"explored"`
	
	// In TWX these are LongInt pointers to linked lists, we'll handle differently
	Ships    []TShip    `json:"ships"`
	Traders  []TTrader  `json:"traders"`
	Planets  []TPlanet  `json:"planets"`
	Vars     []TSectorVar `json:"vars"`     // Sector variables
}

// TTrader matches TWX TTrader record
type TTrader struct {
	Name     string `json:"name"`      // string[40] in TWX
	ShipType string `json:"ship_type"` // string[40] in TWX  
	ShipName string `json:"ship_name"` // string[40] in TWX
	Figs     int    `json:"figs"`      // LongInt in TWX
}

// TShip matches TWX TShip record
type TShip struct {
	Name     string `json:"name"`      // string[40] in TWX
	Owner    string `json:"owner"`     // string[40] in TWX
	ShipType string `json:"ship_type"` // string[40] in TWX  
	Figs     int    `json:"figs"`      // LongInt in TWX
}

// TPlanet matches TWX TPlanet record with parser enhancements
type TPlanet struct {
	Name     string `json:"name"`     // string[40] in TWX
	Owner    string `json:"owner"`    // Enhanced from parser
	Fighters int    `json:"fighters"` // Enhanced from parser
	Citadel  bool   `json:"citadel"`  // Enhanced from parser  
	Stardock bool   `json:"stardock"` // Enhanced from parser
}

// TSectorVar matches TWX TSectorVar record
type TSectorVar struct {
	VarName string `json:"var_name"` // string[10] in TWX
	Value   string `json:"value"`    // string[40] in TWX
}

// TMessageType represents different message categories (matches parser)
type TMessageType int

const (
	TMessageGeneral TMessageType = iota
	TMessageFighter
	TMessageComputer
	TMessageRadio
	TMessageFedlink
	TMessagePlanet
)

// TMessageHistory holds historical message data (matches parser)
type TMessageHistory struct {
	Type      TMessageType `json:"type"`
	Timestamp time.Time    `json:"timestamp"`
	Content   string       `json:"content"`
	Sender    string       `json:"sender"`
	Channel   int          `json:"channel"`
}

// TPlayerStats holds current player statistics (matches parser)
type TPlayerStats struct {
	Turns         int    `json:"turns"`
	Credits       int    `json:"credits"`
	Fighters      int    `json:"fighters"`
	Shields       int    `json:"shields"`
	TotalHolds    int    `json:"total_holds"`
	OreHolds      int    `json:"ore_holds"`
	OrgHolds      int    `json:"org_holds"`
	EquHolds      int    `json:"equ_holds"`
	ColHolds      int    `json:"col_holds"`
	Photons       int    `json:"photons"`
	Armids        int    `json:"armids"`
	Limpets       int    `json:"limpets"`
	GenTorps      int    `json:"gen_torps"`
	TwarpType     int    `json:"twarp_type"`
	Cloaks        int    `json:"cloaks"`
	Beacons       int    `json:"beacons"`
	Atomics       int    `json:"atomics"`
	Corbomite     int    `json:"corbomite"`
	Eprobes       int    `json:"eprobes"`
	MineDisr      int    `json:"mine_disr"`
	Alignment     int    `json:"alignment"`
	Experience    int    `json:"experience"`
	Corp          int    `json:"corp"`
	ShipNumber    int    `json:"ship_number"`
	PsychicProbe  bool   `json:"psychic_probe"`
	PlanetScanner bool   `json:"planet_scanner"`
	ScanType      int    `json:"scan_type"`
	ShipClass     string `json:"ship_class"`
	
	// Current game state (like TWX Database.pas)
	CurrentSector int    `json:"current_sector"`
	PlayerName    string `json:"player_name"`
}

// Helper functions matching TWX behavior

// NULLSector initializes a sector with TWX default values
func NULLSector() TSector {
	return TSector{
		Warp:          [6]int{0, 0, 0, 0, 0, 0},
		// SPort removed - now in separate ports table
		NavHaz:        0,
		Figs:          TSpaceObject{},
		MinesArmid:    TSpaceObject{},
		MinesLimpet:   TSpaceObject{},
		Constellation: "",
		Beacon:        "",
		UpDate:        time.Time{},
		Anomaly:       false,
		Density:       -1, // TWX default for unexplored sectors
		Warps:         0,
		Explored:      EtNo,
		Ships:         []TShip{},
		Traders:       []TTrader{},
		Planets:       []TPlanet{},
		Vars:          []TSectorVar{},
	}
}

// NULLPort initializes a port with TWX default values  
func NULLPort() TPort {
	return TPort{
		Name:           "",
		Dead:           false,
		BuildTime:      0,
		ClassIndex:     -1, // -1 indicates unknown port class
		BuyProduct:     [3]bool{false, false, false},
		ProductPercent: [3]int{0, 0, 0},
		ProductAmount:  [3]int{0, 0, 0},
		UpDate:         time.Time{},
	}
}

// NULLTrader initializes a trader with TWX default values
func NULLTrader() TTrader {
	return TTrader{
		Name:     "",
		ShipType: "",
		ShipName: "",
		Figs:     0,
	}
}

// NULLShip initializes a ship with TWX default values
func NULLShip() TShip {
	return TShip{
		Name:     "",
		Owner:    "",
		ShipType: "",
		Figs:     0,
	}
}

// NULLPlanet initializes a planet with TWX default values
func NULLPlanet() TPlanet {
	return TPlanet{
		Name:     "",
		Owner:    "",
		Fighters: 0,
		Citadel:  false,
		Stardock: false,
	}
}