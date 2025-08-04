package streaming

import (
	"time"
	"twist/internal/proxy/database"
)

// converters.go - Data structure conversion between parser and database formats
// This follows the adapter pattern for clean separation of concerns

// SectorConverter handles conversion between parser and database sector formats
type SectorConverter struct{}

// NewSectorConverter creates a new sector converter
func NewSectorConverter() *SectorConverter {
	return &SectorConverter{}
}

// ToDatabase converts parser SectorData to database TSector
func (c *SectorConverter) ToDatabase(parserSector SectorData) database.TSector {
	dbSector := database.NULLSector()
	
	// Basic sector info
	dbSector.Warp = parserSector.Warps
	dbSector.Constellation = parserSector.Constellation
	dbSector.Beacon = parserSector.Beacon
	dbSector.Density = parserSector.Density
	dbSector.NavHaz = parserSector.NavHaz
	dbSector.Anomaly = parserSector.Anomaly
	
	// Convert boolean explored to enum type
	if parserSector.Explored {
		dbSector.Explored = database.EtDensity // Assume density scan if explored
	} else {
		dbSector.Explored = database.EtNo
	}
	
	// Port data conversion
	dbSector.SPort = c.convertPortData(parserSector.Port)
	
	// Convert related data
	dbSector.Ships = c.convertShips(parserSector.Ships)
	dbSector.Traders = c.convertTraders(parserSector.Traders)
	dbSector.Planets = c.convertPlanets(parserSector.Planets)
	
	// Convert mines - combine into space objects
	dbSector.MinesArmid, dbSector.MinesLimpet = c.convertMines(parserSector.Mines)
	
	dbSector.UpDate = time.Now()
	return dbSector
}

// FromDatabase converts database TSector to parser SectorData
func (c *SectorConverter) FromDatabase(dbSector database.TSector) SectorData {
	parserSector := SectorData{
		Warps:         dbSector.Warp,
		Constellation: dbSector.Constellation,
		Beacon:        dbSector.Beacon,
		Density:       dbSector.Density,
		NavHaz:        dbSector.NavHaz,
		Anomaly:       dbSector.Anomaly,
		Explored:      dbSector.Explored > database.EtNo,
		Port:          c.convertPortFromDB(dbSector.SPort),
		Ships:         c.convertShipsFromDB(dbSector.Ships),
		Traders:       c.convertTradersFromDB(dbSector.Traders),
		Planets:       c.convertPlanetsFromDB(dbSector.Planets),
		Mines:         c.convertMinesFromDB(dbSector.MinesArmid, dbSector.MinesLimpet),
	}
	
	return parserSector
}

// convertPortData converts parser port data to database format
func (c *SectorConverter) convertPortData(port PortData) database.TPort {
	dbPort := database.NULLPort()
	
	dbPort.Name = port.Name
	dbPort.Dead = port.Dead
	dbPort.ClassIndex = port.ClassIndex
	dbPort.BuildTime = port.BuildTime
	
	// Convert product arrays
	dbPort.BuyProduct[0] = port.BuyOre
	dbPort.BuyProduct[1] = port.BuyOrg
	dbPort.BuyProduct[2] = port.BuyEquip
	
	dbPort.ProductPercent[0] = port.OrePercent
	dbPort.ProductPercent[1] = port.OrgPercent
	dbPort.ProductPercent[2] = port.EquipPercent
	
	dbPort.ProductAmount[0] = port.OreAmount
	dbPort.ProductAmount[1] = port.OrgAmount
	dbPort.ProductAmount[2] = port.EquipAmount
	
	return dbPort
}

// convertPortFromDB converts database port data to parser format
func (c *SectorConverter) convertPortFromDB(dbPort database.TPort) PortData {
	return PortData{
		Name:         dbPort.Name,
		Dead:         dbPort.Dead,
		ClassIndex:   dbPort.ClassIndex,
		BuildTime:    dbPort.BuildTime,
		BuyOre:       dbPort.BuyProduct[0],
		BuyOrg:       dbPort.BuyProduct[1],
		BuyEquip:     dbPort.BuyProduct[2],
		OrePercent:   dbPort.ProductPercent[0],
		OrgPercent:   dbPort.ProductPercent[1],
		EquipPercent: dbPort.ProductPercent[2],
		OreAmount:    dbPort.ProductAmount[0],
		OrgAmount:    dbPort.ProductAmount[1],
		EquipAmount:  dbPort.ProductAmount[2],
	}
}

// convertShips converts parser ships to database format
func (c *SectorConverter) convertShips(ships []ShipInfo) []database.TShip {
	var dbShips []database.TShip
	for _, ship := range ships {
		dbShip := database.TShip{
			Name:     ship.Name,
			Owner:    ship.Owner,
			ShipType: ship.ShipType,
			Figs:     ship.Fighters,
		}
		dbShips = append(dbShips, dbShip)
	}
	return dbShips
}

// convertShipsFromDB converts database ships to parser format
func (c *SectorConverter) convertShipsFromDB(dbShips []database.TShip) []ShipInfo {
	var ships []ShipInfo
	for _, ship := range dbShips {
		parserShip := ShipInfo{
			Name:      ship.Name,
			Owner:     ship.Owner,
			ShipType:  ship.ShipType,
			Fighters:  ship.Figs,
			Alignment: "", // Not stored in db
		}
		ships = append(ships, parserShip)
	}
	return ships
}

// convertTraders converts parser traders to database format
func (c *SectorConverter) convertTraders(traders []TraderInfo) []database.TTrader {
	var dbTraders []database.TTrader
	for _, trader := range traders {
		dbTrader := database.TTrader{
			Name:     trader.Name,
			ShipType: trader.ShipType,
			ShipName: trader.ShipName,
			Figs:     trader.Fighters,
		}
		dbTraders = append(dbTraders, dbTrader)
	}
	return dbTraders
}

// convertTradersFromDB converts database traders to parser format
func (c *SectorConverter) convertTradersFromDB(dbTraders []database.TTrader) []TraderInfo {
	var traders []TraderInfo
	for _, trader := range dbTraders {
		parserTrader := TraderInfo{
			Name:      trader.Name,
			ShipType:  trader.ShipType,
			ShipName:  trader.ShipName,
			Fighters:  trader.Figs,
			Alignment: "", // Not stored in db
		}
		traders = append(traders, parserTrader)
	}
	return traders
}

// convertPlanets converts parser planets to database format
func (c *SectorConverter) convertPlanets(planets []PlanetInfo) []database.TPlanet {
	var dbPlanets []database.TPlanet
	for _, planet := range planets {
		dbPlanet := database.TPlanet{
			Name:     planet.Name,
			Owner:    planet.Owner,
			Fighters: planet.Fighters,
			Citadel:  planet.Citadel,
			Stardock: planet.Stardock,
		}
		dbPlanets = append(dbPlanets, dbPlanet)
	}
	return dbPlanets
}

// convertPlanetsFromDB converts database planets to parser format
func (c *SectorConverter) convertPlanetsFromDB(dbPlanets []database.TPlanet) []PlanetInfo {
	var planets []PlanetInfo
	for _, planet := range dbPlanets {
		parserPlanet := PlanetInfo{
			Name:     planet.Name,
			Owner:    planet.Owner,
			Fighters: planet.Fighters,
			Citadel:  planet.Citadel,
			Stardock: planet.Stardock,
		}
		planets = append(planets, parserPlanet)
	}
	return planets
}

// convertMines converts parser mines to database format
func (c *SectorConverter) convertMines(mines []MineInfo) (database.TSpaceObject, database.TSpaceObject) {
	var armidMines, limpetMines database.TSpaceObject
	
	for _, mine := range mines {
		if mine.Type == "Armid" {
			armidMines.Quantity += mine.Quantity
			if armidMines.Owner == "" {
				armidMines.Owner = mine.Owner
			}
		} else if mine.Type == "Limpet" {
			limpetMines.Quantity += mine.Quantity
			if limpetMines.Owner == "" {
				limpetMines.Owner = mine.Owner
			}
		}
	}
	
	return armidMines, limpetMines
}

// convertMinesFromDB converts database mines to parser format
func (c *SectorConverter) convertMinesFromDB(armidMines, limpetMines database.TSpaceObject) []MineInfo {
	var mines []MineInfo
	
	if armidMines.Quantity > 0 {
		mine := MineInfo{
			Type:     "Armid",
			Quantity: armidMines.Quantity,
			Owner:    armidMines.Owner,
		}
		mines = append(mines, mine)
	}
	
	if limpetMines.Quantity > 0 {
		mine := MineInfo{
			Type:     "Limpet",
			Quantity: limpetMines.Quantity,
			Owner:    limpetMines.Owner,
		}
		mines = append(mines, mine)
	}
	
	return mines
}

// PlayerStatsConverter handles conversion between parser and database player stats
type PlayerStatsConverter struct{}

// NewPlayerStatsConverter creates a new player stats converter
func NewPlayerStatsConverter() *PlayerStatsConverter {
	return &PlayerStatsConverter{}
}

// ToDatabase converts parser PlayerStats to database TPlayerStats
func (c *PlayerStatsConverter) ToDatabase(parserStats PlayerStats) database.TPlayerStats {
	return database.TPlayerStats{
		Turns:         parserStats.Turns,
		Credits:       parserStats.Credits,
		Fighters:      parserStats.Fighters,
		Shields:       parserStats.Shields,
		TotalHolds:    parserStats.TotalHolds,
		OreHolds:      parserStats.OreHolds,
		OrgHolds:      parserStats.OrgHolds,
		EquHolds:      parserStats.EquHolds,
		ColHolds:      parserStats.ColHolds,
		Photons:       parserStats.Photons,
		Armids:        parserStats.Armids,
		Limpets:       parserStats.Limpets,
		GenTorps:      parserStats.GenTorps,
		TwarpType:     parserStats.TwarpType,
		Cloaks:        parserStats.Cloaks,
		Beacons:       parserStats.Beacons,
		Atomics:       parserStats.Atomics,
		Corbomite:     parserStats.Corbomite,
		Eprobes:       parserStats.Eprobes,
		MineDisr:      parserStats.MineDisr,
		Alignment:     parserStats.Alignment,
		Experience:    parserStats.Experience,
		Corp:          parserStats.Corp,
		ShipNumber:    parserStats.ShipNumber,
		PsychicProbe:  parserStats.PsychicProbe,
		PlanetScanner: parserStats.PlanetScanner,
		ScanType:      parserStats.ScanType,
		ShipClass:     parserStats.ShipClass,
		CurrentSector: parserStats.CurrentSector,
		PlayerName:    parserStats.PlayerName,
	}
}

// MessageHistoryConverter handles conversion between parser and database message history
type MessageHistoryConverter struct{}

// NewMessageHistoryConverter creates a new message history converter
func NewMessageHistoryConverter() *MessageHistoryConverter {
	return &MessageHistoryConverter{}
}

// ToDatabase converts parser MessageHistory to database TMessageHistory
func (c *MessageHistoryConverter) ToDatabase(parserMessage MessageHistory) database.TMessageHistory {
	return database.TMessageHistory{
		Type:      database.TMessageType(parserMessage.Type),
		Timestamp: parserMessage.Timestamp,
		Content:   parserMessage.Content,
		Sender:    parserMessage.Sender,
		Channel:   parserMessage.Channel,
	}
}