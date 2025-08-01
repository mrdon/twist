package parser

import (
	"log"
	"regexp"
	"twist/internal/proxy/database"
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

// ParserState holds the current parsing state
type ParserState struct {
	CurrentDisplay     TDisplay
	SectorPosition     TSectorPosition
	CurrentSectorIndex int
	PortSectorIndex    int
	FigScanSector      int
	LastWarp           int
	SectorSaved        bool
	FigScanType        TFigScanType

	// Current parsing data
	CurrentSector *database.TSector
	CurrentShip   *database.TShip
	CurrentTrader *database.TTrader
}

// ParserContext holds shared parsing resources
type ParserContext struct {
	Logger      *log.Logger
	DataLog     *log.Logger
	AnsiPattern *regexp.Regexp
	DB          database.Database
	State       *ParserState
}

// NewParserContext creates a new parser context
func NewParserContext(db database.Database, logger, dataLog *log.Logger) *ParserContext {
	return &ParserContext{
		Logger:      logger,
		DataLog:     dataLog,
		AnsiPattern: regexp.MustCompile(`\x1b\[[0-9;]*[mK]`),
		DB:          db,
		State: &ParserState{
			CurrentDisplay: DNone,
			SectorPosition: SpNormal,
		},
	}
}