package streaming

import (
	"database/sql"
	"twist/internal/debug"
)

// SectorCollections manages all collection trackers for a sector
// Collections require full replacement since we can't do incremental updates
type SectorCollections struct {
	sectorIndex    int
	shipsTracker   *ShipsCollectionTracker
	tradersTracker *TradersCollectionTracker
	planetsTracker *PlanetsCollectionTracker
}

// NewSectorCollections creates a new sector collections manager
func NewSectorCollections(sectorIndex int) *SectorCollections {
	return &SectorCollections{
		sectorIndex:    sectorIndex,
		shipsTracker:   NewShipsCollectionTracker(sectorIndex),
		tradersTracker: NewTradersCollectionTracker(sectorIndex),
		planetsTracker: NewPlanetsCollectionTracker(sectorIndex),
	}
}

// AddShip adds a ship to the discovered ships collection
func (sc *SectorCollections) AddShip(name, owner, shipType string, fighters int) {
	sc.shipsTracker.AddShip(name, owner, shipType, fighters)
}

// AddTrader adds a trader to the discovered traders collection
func (sc *SectorCollections) AddTrader(name, shipName, shipType string, fighters int) {
	sc.tradersTracker.AddTrader(name, shipName, shipType, fighters)
}

// AddPlanet adds a planet to the discovered planets collection
func (sc *SectorCollections) AddPlanet(name, owner string, fighters int, citadel, stardock bool) {
	sc.planetsTracker.AddPlanet(name, owner, fighters, citadel, stardock)
}

// HasData returns true if any collections have data
func (sc *SectorCollections) HasData() bool {
	return sc.shipsTracker.HasShips() ||
		sc.tradersTracker.HasTraders() ||
		sc.planetsTracker.HasPlanets()
}

// Execute performs atomic replacement of all collections in the sector
func (sc *SectorCollections) Execute(db *sql.DB) error {
	// Execute all collection updates in sequence
	if sc.shipsTracker.HasShips() {
		if err := sc.shipsTracker.Execute(db); err != nil {
			return err
		}
	}

	if sc.tradersTracker.HasTraders() {
		if err := sc.tradersTracker.Execute(db); err != nil {
			return err
		}
	}

	if sc.planetsTracker.HasPlanets() {
		if err := sc.planetsTracker.Execute(db); err != nil {
			return err
		}
	}

	return nil
}

// ShipData represents a discovered ship
type ShipData struct {
	Name     string
	Owner    string
	ShipType string
	Fighters int
}

// ShipsCollectionTracker manages atomic replacement of ships for a sector
type ShipsCollectionTracker struct {
	sectorIndex int
	ships       []ShipData
}

// NewShipsCollectionTracker creates a new ships collection tracker
func NewShipsCollectionTracker(sectorIndex int) *ShipsCollectionTracker {
	return &ShipsCollectionTracker{
		sectorIndex: sectorIndex,
		ships:       make([]ShipData, 0),
	}
}

// AddShip adds a ship to the collection
func (s *ShipsCollectionTracker) AddShip(name, owner, shipType string, fighters int) {
	s.ships = append(s.ships, ShipData{
		Name:     name,
		Owner:    owner,
		ShipType: shipType,
		Fighters: fighters,
	})
}

// HasShips returns true if ships were discovered
func (s *ShipsCollectionTracker) HasShips() bool {
	return len(s.ships) > 0
}

// Execute performs atomic replace: DELETE + INSERT in transaction
func (s *ShipsCollectionTracker) Execute(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// Clear existing ships for this sector
	_, err = tx.Exec("DELETE FROM ships WHERE sector_index = ?", s.sectorIndex)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Insert discovered ships
	for _, ship := range s.ships {
		_, err = tx.Exec(`
			INSERT INTO ships (sector_index, name, owner, ship_type, fighters) 
			VALUES (?, ?, ?, ?, ?)`,
			s.sectorIndex, ship.Name, ship.Owner, ship.ShipType, ship.Fighters)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	debug.Info("COLLECTIONS: Updated ships for sector", "count", len(s.ships), "sector", s.sectorIndex)
	return nil
}

// TraderData represents a discovered trader
type TraderData struct {
	Name     string
	ShipName string
	ShipType string
	Fighters int
}

// TradersCollectionTracker manages atomic replacement of traders for a sector
type TradersCollectionTracker struct {
	sectorIndex int
	traders     []TraderData
}

// NewTradersCollectionTracker creates a new traders collection tracker
func NewTradersCollectionTracker(sectorIndex int) *TradersCollectionTracker {
	return &TradersCollectionTracker{
		sectorIndex: sectorIndex,
		traders:     make([]TraderData, 0),
	}
}

// AddTrader adds a trader to the collection
func (t *TradersCollectionTracker) AddTrader(name, shipName, shipType string, fighters int) {
	t.traders = append(t.traders, TraderData{
		Name:     name,
		ShipName: shipName,
		ShipType: shipType,
		Fighters: fighters,
	})
}

// HasTraders returns true if traders were discovered
func (t *TradersCollectionTracker) HasTraders() bool {
	return len(t.traders) > 0
}

// Execute performs atomic replace: DELETE + INSERT in transaction
func (t *TradersCollectionTracker) Execute(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// Clear existing traders for this sector
	_, err = tx.Exec("DELETE FROM traders WHERE sector_index = ?", t.sectorIndex)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Insert discovered traders
	for _, trader := range t.traders {
		_, err = tx.Exec(`
			INSERT INTO traders (sector_index, name, ship_type, ship_name, fighters) 
			VALUES (?, ?, ?, ?, ?)`,
			t.sectorIndex, trader.Name, trader.ShipType, trader.ShipName, trader.Fighters)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	debug.Info("COLLECTIONS: Updated traders for sector", "count", len(t.traders), "sector", t.sectorIndex)
	return nil
}

// PlanetData represents a discovered planet
type PlanetData struct {
	Name     string
	Owner    string
	Fighters int
	Citadel  bool
	Stardock bool
}

// PlanetsCollectionTracker manages atomic replacement of planets for a sector
type PlanetsCollectionTracker struct {
	sectorIndex int
	planets     []PlanetData
}

// NewPlanetsCollectionTracker creates a new planets collection tracker
func NewPlanetsCollectionTracker(sectorIndex int) *PlanetsCollectionTracker {
	return &PlanetsCollectionTracker{
		sectorIndex: sectorIndex,
		planets:     make([]PlanetData, 0),
	}
}

// AddPlanet adds a planet to the collection
func (p *PlanetsCollectionTracker) AddPlanet(name, owner string, fighters int, citadel, stardock bool) {
	p.planets = append(p.planets, PlanetData{
		Name:     name,
		Owner:    owner,
		Fighters: fighters,
		Citadel:  citadel,
		Stardock: stardock,
	})
}

// HasPlanets returns true if planets were discovered
func (p *PlanetsCollectionTracker) HasPlanets() bool {
	return len(p.planets) > 0
}

// Execute performs atomic replace: DELETE + INSERT in transaction
func (p *PlanetsCollectionTracker) Execute(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// Clear existing planets for this sector
	_, err = tx.Exec("DELETE FROM planets WHERE sector_index = ?", p.sectorIndex)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Insert discovered planets
	for _, planet := range p.planets {
		_, err = tx.Exec(`
			INSERT INTO planets (sector_index, name, owner, fighters, citadel, stardock) 
			VALUES (?, ?, ?, ?, ?, ?)`,
			p.sectorIndex, planet.Name, planet.Owner, planet.Fighters, planet.Citadel, planet.Stardock)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	debug.Info("COLLECTIONS: Updated planets for sector", "count", len(p.planets), "sector", p.sectorIndex)
	return nil
}
