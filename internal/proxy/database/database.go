package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Database interface matching TWX IModDatabase
type Database interface {
	// Core database operations
	OpenDatabase(filename string) error
	CloseDatabase() error
	CreateDatabase(filename string) error
	
	// Sector operations (matching TWX methods)
	SaveSector(sector TSector, index int) error
	LoadSector(index int) (TSector, error)
	
	// Enhanced SaveSector with collections (Pascal-compliant signature)
	SaveSectorWithCollections(sector TSector, index int, ships []TShip, traders []TTrader, planets []TPlanet) error
	
	// Port operations (Phase 2: Database Schema Optimization)
	SavePort(port TPort, sectorIndex int) error
	LoadPort(sectorIndex int) (TPort, error)
	DeletePort(sectorIndex int) error
	FindPortsByClass(classIndex int) ([]TPort, error)
	FindPortsBuying(product TProductType) ([]TPort, error)
	
	// TWX compatibility methods
	GetDatabaseOpen() bool
	GetSectors() int
	
	// Script variable operations
	SaveScriptVariable(name string, value interface{}) error
	LoadScriptVariable(name string) (interface{}, error)
	
	// Parser integration methods
	SavePlayerStats(stats TPlayerStats) error
	LoadPlayerStats() (TPlayerStats, error)
	AddMessageToHistory(message TMessageHistory) error
	GetMessageHistory(limit int) ([]TMessageHistory, error)
	
	// Fighter management
	ResetPersonalCorpFighters() error
	
	// Modern additions
	BeginTransaction() error
	CommitTransaction() error
	RollbackTransaction() error
	
	// Internal access for advanced operations
	GetDB() *sql.DB
}

// SQLiteDatabase implements Database interface using SQLite
type SQLiteDatabase struct {
	db           *sql.DB
	dbOpen       bool
	filename     string
	sectors      int
	tx           *sql.Tx  // Current transaction
	
	// Prepared statements for performance
	loadSectorStmt *sql.Stmt
	saveSectorStmt *sql.Stmt
}

// NewDatabase creates a new SQLite database instance
func NewDatabase() *SQLiteDatabase {
	return &SQLiteDatabase{}
}

// OpenDatabase opens an existing SQLite database (matching TWX method)
func (d *SQLiteDatabase) OpenDatabase(filename string) error {
	if d.dbOpen {
		return fmt.Errorf("database already open")
	}
	
	var err error
	d.db, err = sql.Open("sqlite3", filename+"?_foreign_keys=on")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	
	// Test the connection
	if err = d.db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	
	// Create schema (handles IF NOT EXISTS)
	if err = d.createSchema(); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	
	// Check if database has proper schema
	if err = d.validateSchema(); err != nil {
		return fmt.Errorf("invalid database schema: %w", err)  
	}
	
	// Get sector count
	d.sectors, err = d.getSectorCount()
	if err != nil {
		return fmt.Errorf("failed to get sector count: %w", err)
	}
	
	// Prepare statements
	if err = d.prepareStatements(); err != nil {
		return fmt.Errorf("failed to prepare statements: %w", err)
	}
	
	d.filename = filename
	d.dbOpen = true
	
	return nil
}

// CreateDatabase creates a new SQLite database with TWX-compatible schema
func (d *SQLiteDatabase) CreateDatabase(filename string) error {
	
	var err error
	d.db, err = sql.Open("sqlite3", filename+"?_foreign_keys=on")
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	
	// Test the connection
	if err = d.db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	
	// Create complete schema (no migrations for new app)
	if err = d.createSchema(); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	
	// Validate schema was created correctly
	if err = d.validateSchema(); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}
	
	// Get sector count
	d.sectors, err = d.getSectorCount()
	if err != nil {
		return fmt.Errorf("failed to get sector count: %w", err)
	}
	
	// Prepare statements for performance
	if err = d.prepareStatements(); err != nil {
		return fmt.Errorf("failed to prepare statements: %w", err)
	}
	
	d.filename = filename
	d.dbOpen = true
	
	return nil
}

// CloseDatabase closes the database connection (matching TWX method)
func (d *SQLiteDatabase) CloseDatabase() error {
	if !d.dbOpen {
		return nil
	}
	
	// Close prepared statements
	if d.loadSectorStmt != nil {
		d.loadSectorStmt.Close()
	}
	if d.saveSectorStmt != nil {
		d.saveSectorStmt.Close()
	}
	
	// Close database
	if d.db != nil {
		if err := d.db.Close(); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}
	
	d.dbOpen = false
	d.filename = ""
	return nil
}

// LoadSector retrieves a sector by index (matching TWX method)
func (d *SQLiteDatabase) LoadSector(index int) (TSector, error) {
	if !d.dbOpen {
		return NULLSector(), fmt.Errorf("database not open")
	}
	
	if index <= 0 {
		return NULLSector(), fmt.Errorf("invalid sector index: %d", index)
	}
	
	sector := NULLSector()
	
	// Load main sector data (Phase 2: port data removed from sectors table)
	row := d.loadSectorStmt.QueryRow(index)
	
	var upDate sql.NullTime
	
	err := row.Scan(
		&sector.Warp[0], &sector.Warp[1], &sector.Warp[2], 
		&sector.Warp[3], &sector.Warp[4], &sector.Warp[5],
		&sector.Constellation, &sector.Beacon, &sector.NavHaz,
		&sector.Density, &sector.Anomaly, &sector.Warps, &sector.Explored,
		&upDate, 
		&sector.Figs.Quantity, &sector.Figs.Owner, &sector.Figs.FigType,
		&sector.MinesArmid.Quantity, &sector.MinesArmid.Owner,
		&sector.MinesLimpet.Quantity, &sector.MinesLimpet.Owner,
	)
	
	if err == sql.ErrNoRows {
		// Sector doesn't exist, return blank sector (like TWX)
		return NULLSector(), nil
	} else if err != nil {
		return NULLSector(), fmt.Errorf("failed to load sector %d: %w", index, err)
	}
	
	// Handle nullable timestamps
	if upDate.Valid {
		sector.UpDate = upDate.Time
	}
	
	// Load related data (ships, traders, planets)
	if err = d.loadSectorRelatedData(index, &sector); err != nil {
		return sector, fmt.Errorf("failed to load related data for sector %d: %w", index, err)
	}
	
	return sector, nil
}

// SaveSector stores a sector (matching TWX method signature)
func (d *SQLiteDatabase) SaveSector(sector TSector, index int) error {
	if !d.dbOpen {
		return fmt.Errorf("database not open")
	}
	
	if index <= 0 {
		return fmt.Errorf("invalid sector index: %d", index)
	}
	
	// Debug: Verify database connection and table existence
	if d.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	
	// Test a simple query to ensure the connection works
	var tableCount int
	if err := d.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sectors'").Scan(&tableCount); err != nil {
		return fmt.Errorf("failed to query sqlite_master: %w", err)  
	}
	
	if tableCount == 0 {
		return fmt.Errorf("sectors table does not exist (found %d tables named 'sectors')", tableCount)
	}
	
	
	// Start transaction if not already in one
	shouldCommit := false
	if d.tx == nil {
		if err := d.BeginTransaction(); err != nil {
			return err
		}
		shouldCommit = true
	}
	
	// Update timestamp
	sector.UpDate = time.Now()
	
	// Save main sector data (Phase 2: port data removed from sectors table)
	saveQuery := `
	INSERT OR REPLACE INTO sectors (
		sector_index,
		warp1, warp2, warp3, warp4, warp5, warp6,
		constellation, beacon, nav_haz, density, anomaly, warps, explored, update_time,
		figs_quantity, figs_owner, figs_type,
		mines_armid_quantity, mines_armid_owner,
		mines_limpet_quantity, mines_limpet_owner
	) VALUES (
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
	);`
	
	_, err := d.tx.Exec(saveQuery,
		index,
		sector.Warp[0], sector.Warp[1], sector.Warp[2],
		sector.Warp[3], sector.Warp[4], sector.Warp[5],
		sector.Constellation, sector.Beacon, sector.NavHaz,
		sector.Density, sector.Anomaly, sector.Warps, int(sector.Explored),
		sector.UpDate,
		sector.Figs.Quantity, sector.Figs.Owner, int(sector.Figs.FigType),
		sector.MinesArmid.Quantity, sector.MinesArmid.Owner,
		sector.MinesLimpet.Quantity, sector.MinesLimpet.Owner,
	)
	
	if err != nil {
		if shouldCommit {
			d.RollbackTransaction()
		}
		return fmt.Errorf("failed to save sector %d: %w", index, err)
	}
	
	// Save related data
	if err = d.saveSectorRelatedData(index, sector); err != nil {
		if shouldCommit {
			d.RollbackTransaction()
		}
		return fmt.Errorf("failed to save related data for sector %d: %w", index, err)
	}
	
	
	if shouldCommit {
		return d.CommitTransaction()
	}
	
	return nil
}

// SaveSectorWithCollections stores a sector with explicit collections (Pascal-compliant signature)
// This mirrors Pascal TWX: SaveSector(FCurrentSector, FCurrentSectorIndex, FShipList, FTraderList, FPlanetList)
func (d *SQLiteDatabase) SaveSectorWithCollections(sector TSector, index int, ships []TShip, traders []TTrader, planets []TPlanet) error {
	if !d.dbOpen {
		return fmt.Errorf("database not open")
	}
	
	if index <= 0 {
		return fmt.Errorf("invalid sector index: %d", index)
	}
	
	// Start transaction for atomic operation
	shouldCommit := false
	if d.tx == nil {
		if err := d.BeginTransaction(); err != nil {
			return err
		}
		shouldCommit = true
	}
	
	// Update timestamp
	sector.UpDate = time.Now()
	
	// Save main sector data (Phase 2: port data removed from sectors table)
	saveQuery := `
	INSERT OR REPLACE INTO sectors (
		sector_index,
		warp1, warp2, warp3, warp4, warp5, warp6,
		constellation, beacon, nav_haz, density, anomaly, warps, explored, update_time,
		figs_quantity, figs_owner, figs_type,
		mines_armid_quantity, mines_armid_owner,
		mines_limpet_quantity, mines_limpet_owner
	) VALUES (
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
	);`
	
	_, err := d.tx.Exec(saveQuery,
		index,
		sector.Warp[0], sector.Warp[1], sector.Warp[2],
		sector.Warp[3], sector.Warp[4], sector.Warp[5],
		sector.Constellation, sector.Beacon, sector.NavHaz,
		sector.Density, sector.Anomaly, sector.Warps, int(sector.Explored),
		sector.UpDate,
		sector.Figs.Quantity, sector.Figs.Owner, int(sector.Figs.FigType),
		sector.MinesArmid.Quantity, sector.MinesArmid.Owner,
		sector.MinesLimpet.Quantity, sector.MinesLimpet.Owner,
	)
	
	if err != nil {
		if shouldCommit {
			d.RollbackTransaction()
		}
		return fmt.Errorf("failed to save sector %d: %w", index, err)
	}
	
	// Save collections with explicit parameters (Pascal-compliant approach)
	if err = d.saveSectorCollectionsWithParams(index, ships, traders, planets); err != nil {
		if shouldCommit {
			d.RollbackTransaction()
		}
		return fmt.Errorf("failed to save collections for sector %d: %w", index, err)
	}
	
	if shouldCommit {
		return d.CommitTransaction()
	}
	
	return nil
}

// GetDatabaseOpen returns whether database is open (TWX compatibility)
func (d *SQLiteDatabase) GetDatabaseOpen() bool {
	return d.dbOpen
}

// GetSectors returns the number of sectors (TWX compatibility)  
func (d *SQLiteDatabase) GetSectors() int {
	return d.sectors
}

func (d *SQLiteDatabase) GetDB() *sql.DB {
	return d.db
}

// Transaction methods
func (d *SQLiteDatabase) BeginTransaction() error {
	if d.tx != nil {
		return fmt.Errorf("transaction already active")
	}
	
	var err error
	d.tx, err = d.db.Begin()
	return err
}

func (d *SQLiteDatabase) CommitTransaction() error {
	if d.tx == nil {
		return fmt.Errorf("no active transaction")
	}
	
	err := d.tx.Commit()
	d.tx = nil
	return err
}

func (d *SQLiteDatabase) RollbackTransaction() error {
	if d.tx == nil {
		return fmt.Errorf("no active transaction")  
	}
	
	err := d.tx.Rollback()
	d.tx = nil
	return err
}


// SaveScriptVariable saves a script variable to persistent storage
func (d *SQLiteDatabase) SaveScriptVariable(name string, value interface{}) error {
	if !d.dbOpen {
		return fmt.Errorf("database not open")
	}

	// Determine value type and storage format
	var varType int
	var stringValue string
	var numberValue float64

	switch v := value.(type) {
	case string:
		// Check if this is a serialized array (TWX_ARRAY: prefix)
		if strings.HasPrefix(v, "TWX_ARRAY:") {
			varType = 2 // ArrayType
			stringValue = v
			numberValue = 0
		} else {
			varType = 0 // StringType
			stringValue = v
			numberValue = 0
		}
	case float64:
		varType = 1 // NumberType  
		stringValue = ""
		numberValue = v
	case int:
		varType = 1 // NumberType
		stringValue = ""
		numberValue = float64(v)
	default:
		// For complex types, store as string representation
		varType = 0
		stringValue = fmt.Sprintf("%v", v)
		numberValue = 0
	}

	query := `
	INSERT OR REPLACE INTO script_vars (var_name, var_type, string_value, number_value, updated_at)
	VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP);`

	_, err := d.db.Exec(query, name, varType, stringValue, numberValue)
	if err != nil {
		return fmt.Errorf("failed to save script variable %s: %w", name, err)
	}


	return nil
}

// LoadScriptVariable loads a script variable from persistent storage
func (d *SQLiteDatabase) LoadScriptVariable(name string) (interface{}, error) {
	if !d.dbOpen {
		return nil, fmt.Errorf("database not open")
	}

	query := `
	SELECT var_type, string_value, number_value
	FROM script_vars 
	WHERE var_name = ?;`

	var varType int
	var stringValue string
	var numberValue float64

	err := d.db.QueryRow(query, name).Scan(&varType, &stringValue, &numberValue)
	if err != nil {
		if err == sql.ErrNoRows {
			// Variable doesn't exist, return nil/empty value
			return "", nil
		}
		return nil, fmt.Errorf("failed to load script variable %s: %w", name, err)
	}

	// Return appropriate type based on stored type
	switch varType {
	case 0: // StringType
		return stringValue, nil
	case 1: // NumberType
		return numberValue, nil
	case 2: // ArrayType
		// Arrays are stored as strings with TWX_ARRAY: prefix
		return stringValue, nil
	default:
		// Default to string for unknown types
		return stringValue, nil
	}
}

// SavePlayerStats saves current player statistics to database
func (d *SQLiteDatabase) SavePlayerStats(stats TPlayerStats) error {
	if !d.dbOpen {
		return fmt.Errorf("database not open")
	}

	query := `
	INSERT OR REPLACE INTO player_stats (
		id, turns, credits, fighters, shields, total_holds, ore_holds, org_holds, equ_holds, col_holds,
		photons, armids, limpets, gen_torps, twarp_type, cloaks, beacons, atomics, corbomite, eprobes,
		mine_disr, alignment, experience, corp, ship_number, psychic_probe, planet_scanner, scan_type,
		ship_class, current_sector, player_name, updated_at
	) VALUES (
		1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP
	);`

	_, err := d.db.Exec(query,
		stats.Turns, stats.Credits, stats.Fighters, stats.Shields, stats.TotalHolds,
		stats.OreHolds, stats.OrgHolds, stats.EquHolds, stats.ColHolds, stats.Photons,
		stats.Armids, stats.Limpets, stats.GenTorps, stats.TwarpType, stats.Cloaks,
		stats.Beacons, stats.Atomics, stats.Corbomite, stats.Eprobes, stats.MineDisr,
		stats.Alignment, stats.Experience, stats.Corp, stats.ShipNumber, stats.PsychicProbe,
		stats.PlanetScanner, stats.ScanType, stats.ShipClass, stats.CurrentSector, stats.PlayerName,
	)

	if err != nil {
		return fmt.Errorf("failed to save player stats: %w", err)
	}

	return nil
}

// LoadPlayerStats loads current player statistics from database
func (d *SQLiteDatabase) LoadPlayerStats() (TPlayerStats, error) {
	if !d.dbOpen {
		return TPlayerStats{}, fmt.Errorf("database not open")
	}

	query := `
	SELECT turns, credits, fighters, shields, total_holds, ore_holds, org_holds, equ_holds, col_holds,
		   photons, armids, limpets, gen_torps, twarp_type, cloaks, beacons, atomics, corbomite, eprobes,
		   mine_disr, alignment, experience, corp, ship_number, psychic_probe, planet_scanner, scan_type,
		   ship_class, COALESCE(current_sector, 0), COALESCE(player_name, '')
	FROM player_stats WHERE id = 1;`

	var stats TPlayerStats
	err := d.db.QueryRow(query).Scan(
		&stats.Turns, &stats.Credits, &stats.Fighters, &stats.Shields, &stats.TotalHolds,
		&stats.OreHolds, &stats.OrgHolds, &stats.EquHolds, &stats.ColHolds, &stats.Photons,
		&stats.Armids, &stats.Limpets, &stats.GenTorps, &stats.TwarpType, &stats.Cloaks,
		&stats.Beacons, &stats.Atomics, &stats.Corbomite, &stats.Eprobes, &stats.MineDisr,
		&stats.Alignment, &stats.Experience, &stats.Corp, &stats.ShipNumber, &stats.PsychicProbe,
		&stats.PlanetScanner, &stats.ScanType, &stats.ShipClass, &stats.CurrentSector, &stats.PlayerName,
	)

	if err == sql.ErrNoRows {
		// Return empty stats if none exist
		return TPlayerStats{}, nil
	} else if err != nil {
		return TPlayerStats{}, fmt.Errorf("failed to load player stats: %w", err)
	}

	return stats, nil
}

// AddMessageToHistory adds a message to the message history
func (d *SQLiteDatabase) AddMessageToHistory(message TMessageHistory) error {
	if !d.dbOpen {
		return fmt.Errorf("database not open")
	}

	query := `
	INSERT INTO message_history (message_type, timestamp, content, sender, channel)
	VALUES (?, ?, ?, ?, ?);`

	_, err := d.db.Exec(query, int(message.Type), message.Timestamp, message.Content, message.Sender, message.Channel)
	if err != nil {
		return fmt.Errorf("failed to add message to history: %w", err)
	}

	return nil
}

// GetMessageHistory retrieves recent messages from history
func (d *SQLiteDatabase) GetMessageHistory(limit int) ([]TMessageHistory, error) {
	if !d.dbOpen {
		return nil, fmt.Errorf("database not open")
	}

	if limit <= 0 {
		limit = 100 // Default limit
	}

	query := `
	SELECT message_type, timestamp, content, sender, channel
	FROM message_history
	ORDER BY timestamp DESC
	LIMIT ?;`

	rows, err := d.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get message history: %w", err)
	}
	defer rows.Close()

	var messages []TMessageHistory
	for rows.Next() {
		var message TMessageHistory
		var messageType int
		
		if err := rows.Scan(&messageType, &message.Timestamp, &message.Content, &message.Sender, &message.Channel); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		
		message.Type = TMessageType(messageType)
		messages = append(messages, message)
	}

	return messages, nil
}

// ResetPersonalCorpFighters clears all personal and corp fighter deployments (mirrors TWX Pascal ResetFigDatabase)
func (d *SQLiteDatabase) ResetPersonalCorpFighters() error {
	if !d.dbOpen {
		return fmt.Errorf("database not open")
	}

	// Update all sectors to clear fighters where owner is personal or corp
	query := `
	UPDATE sectors 
	SET figs_quantity = 0, figs_owner = '', figs_type = 3
	WHERE figs_owner IN ('yours', 'belong to your Corp');`

	_, err := d.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to reset personal/corp fighters: %w", err)
	}

	return nil
}

// Port operations (Phase 2: Database Schema Optimization)

// SavePort saves port information to the dedicated ports table
func (d *SQLiteDatabase) SavePort(port TPort, sectorIndex int) error {
	if !d.dbOpen {
		return fmt.Errorf("database not open")
	}
	
	if sectorIndex <= 0 {
		return fmt.Errorf("invalid sector index")
	}

	query := `
	INSERT OR REPLACE INTO ports (
		sector_index, name, class_index, dead, build_time,
		buy_fuel_ore, buy_organics, buy_equipment,
		percent_fuel_ore, percent_organics, percent_equipment,
		amount_fuel_ore, amount_organics, amount_equipment,
		updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP);`

	// Use transaction if active, otherwise use direct connection
	var err error
	if d.tx != nil {
		_, err = d.tx.Exec(query,
			sectorIndex, port.Name, port.ClassIndex, port.Dead, port.BuildTime,
			port.BuyProduct[PtFuelOre], port.BuyProduct[PtOrganics], port.BuyProduct[PtEquipment],
			port.ProductPercent[PtFuelOre], port.ProductPercent[PtOrganics], port.ProductPercent[PtEquipment],
			port.ProductAmount[PtFuelOre], port.ProductAmount[PtOrganics], port.ProductAmount[PtEquipment])
	} else {
		_, err = d.db.Exec(query,
			sectorIndex, port.Name, port.ClassIndex, port.Dead, port.BuildTime,
			port.BuyProduct[PtFuelOre], port.BuyProduct[PtOrganics], port.BuyProduct[PtEquipment],
			port.ProductPercent[PtFuelOre], port.ProductPercent[PtOrganics], port.ProductPercent[PtEquipment],
			port.ProductAmount[PtFuelOre], port.ProductAmount[PtOrganics], port.ProductAmount[PtEquipment])
	}
	
	if err != nil {
		return fmt.Errorf("failed to save port for sector %d: %w", sectorIndex, err)
	}

	return nil
}

// LoadPort loads port information from the dedicated ports table
func (d *SQLiteDatabase) LoadPort(sectorIndex int) (TPort, error) {
	var port TPort
	
	if !d.dbOpen {
		return port, fmt.Errorf("database not open")
	}
	
	if sectorIndex <= 0 {
		return port, fmt.Errorf("invalid sector index")
	}

	query := `
	SELECT name, class_index, dead, build_time,
		   buy_fuel_ore, buy_organics, buy_equipment,
		   percent_fuel_ore, percent_organics, percent_equipment,
		   amount_fuel_ore, amount_organics, amount_equipment,
		   updated_at
	FROM ports WHERE sector_index = ?;`

	var updateTime time.Time
	var err error
	
	// Use transaction if active, otherwise use direct connection
	if d.tx != nil {
		err = d.tx.QueryRow(query, sectorIndex).Scan(
			&port.Name, &port.ClassIndex, &port.Dead, &port.BuildTime,
			&port.BuyProduct[PtFuelOre], &port.BuyProduct[PtOrganics], &port.BuyProduct[PtEquipment],
			&port.ProductPercent[PtFuelOre], &port.ProductPercent[PtOrganics], &port.ProductPercent[PtEquipment],
			&port.ProductAmount[PtFuelOre], &port.ProductAmount[PtOrganics], &port.ProductAmount[PtEquipment],
			&updateTime)
	} else {
		err = d.db.QueryRow(query, sectorIndex).Scan(
			&port.Name, &port.ClassIndex, &port.Dead, &port.BuildTime,
			&port.BuyProduct[PtFuelOre], &port.BuyProduct[PtOrganics], &port.BuyProduct[PtEquipment],
			&port.ProductPercent[PtFuelOre], &port.ProductPercent[PtOrganics], &port.ProductPercent[PtEquipment],
			&port.ProductAmount[PtFuelOre], &port.ProductAmount[PtOrganics], &port.ProductAmount[PtEquipment],
			&updateTime)
	}
	
	if err != nil {
		if err == sql.ErrNoRows {
			// No port in this sector - return empty port struct
			return TPort{}, nil
		}
		return port, fmt.Errorf("failed to load port for sector %d: %w", sectorIndex, err)
	}
	
	port.UpDate = updateTime
	return port, nil
}

// DeletePort removes port information from the dedicated ports table
func (d *SQLiteDatabase) DeletePort(sectorIndex int) error {
	if !d.dbOpen {
		return fmt.Errorf("database not open")
	}
	
	if sectorIndex <= 0 {
		return fmt.Errorf("invalid sector index")
	}

	query := `DELETE FROM ports WHERE sector_index = ?;`
	_, err := d.db.Exec(query, sectorIndex)
	if err != nil {
		return fmt.Errorf("failed to delete port for sector %d: %w", sectorIndex, err)
	}

	return nil
}

// FindPortsByClass finds all ports with a specific class
func (d *SQLiteDatabase) FindPortsByClass(classIndex int) ([]TPort, error) {
	if !d.dbOpen {
		return nil, fmt.Errorf("database not open")
	}

	query := `
	SELECT sector_index, name, class_index, dead, build_time,
		   buy_fuel_ore, buy_organics, buy_equipment,
		   percent_fuel_ore, percent_organics, percent_equipment,
		   amount_fuel_ore, amount_organics, amount_equipment,
		   updated_at
	FROM ports WHERE class_index = ? ORDER BY name;`

	rows, err := d.db.Query(query, classIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to find ports by class %d: %w", classIndex, err)
	}
	defer rows.Close()

	var ports []TPort
	for rows.Next() {
		var port TPort
		var sectorIndex int
		var updateTime time.Time
		
		if err := rows.Scan(
			&sectorIndex, &port.Name, &port.ClassIndex, &port.Dead, &port.BuildTime,
			&port.BuyProduct[PtFuelOre], &port.BuyProduct[PtOrganics], &port.BuyProduct[PtEquipment],
			&port.ProductPercent[PtFuelOre], &port.ProductPercent[PtOrganics], &port.ProductPercent[PtEquipment],
			&port.ProductAmount[PtFuelOre], &port.ProductAmount[PtOrganics], &port.ProductAmount[PtEquipment],
			&updateTime); err != nil {
			return nil, fmt.Errorf("failed to scan port: %w", err)
		}
		
		port.UpDate = updateTime
		ports = append(ports, port)
	}

	return ports, nil
}

// FindPortsBuying finds all ports buying a specific product
func (d *SQLiteDatabase) FindPortsBuying(product TProductType) ([]TPort, error) {
	if !d.dbOpen {
		return nil, fmt.Errorf("database not open")
	}

	var column string
	switch product {
	case PtFuelOre:
		column = "buy_fuel_ore"
	case PtOrganics:
		column = "buy_organics"
	case PtEquipment:
		column = "buy_equipment"
	default:
		return nil, fmt.Errorf("invalid product type: %d", product)
	}

	query := fmt.Sprintf(`
	SELECT sector_index, name, class_index, dead, build_time,
		   buy_fuel_ore, buy_organics, buy_equipment,
		   percent_fuel_ore, percent_organics, percent_equipment,
		   amount_fuel_ore, amount_organics, amount_equipment,
		   updated_at
	FROM ports WHERE %s = TRUE ORDER BY name;`, column)

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to find ports buying %v: %w", product, err)
	}
	defer rows.Close()

	var ports []TPort
	for rows.Next() {
		var port TPort
		var sectorIndex int
		var updateTime time.Time
		
		if err := rows.Scan(
			&sectorIndex, &port.Name, &port.ClassIndex, &port.Dead, &port.BuildTime,
			&port.BuyProduct[PtFuelOre], &port.BuyProduct[PtOrganics], &port.BuyProduct[PtEquipment],
			&port.ProductPercent[PtFuelOre], &port.ProductPercent[PtOrganics], &port.ProductPercent[PtEquipment],
			&port.ProductAmount[PtFuelOre], &port.ProductAmount[PtOrganics], &port.ProductAmount[PtEquipment],
			&updateTime); err != nil {
			return nil, fmt.Errorf("failed to scan port: %w", err)
		}
		
		port.UpDate = updateTime
		ports = append(ports, port)
	}

	return ports, nil
}
