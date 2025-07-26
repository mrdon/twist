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

// TSector matches TWX TSector record exactly
type TSector struct {
	// Warp is array[1..6] in TWX, we'll use [6] and handle 1-indexing in code
	Warp          [6]int               `json:"warp"`           
	SPort         TPort                `json:"sport"`
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

// TPlanet matches TWX TPlanet record
type TPlanet struct {
	Name string `json:"name"` // string[40] in TWX
}

// TSectorVar matches TWX TSectorVar record
type TSectorVar struct {
	VarName string `json:"var_name"` // string[10] in TWX
	Value   string `json:"value"`    // string[40] in TWX
}

// Helper functions matching TWX behavior

// NULLSector initializes a sector with TWX default values
func NULLSector() TSector {
	return TSector{
		Warp:          [6]int{0, 0, 0, 0, 0, 0},
		SPort:         NULLPort(),
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
		Name: "",
	}
}