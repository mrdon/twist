package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
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
	
	// TWX compatibility methods
	GetDatabaseOpen() bool
	GetSectors() int
	
	// Script variable operations
	SaveScriptVariable(name string, value interface{}) error
	LoadScriptVariable(name string) (interface{}, error)
	
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
	logger       *log.Logger
	dataLogger   *log.Logger // Data logging for parsing tracking
	tx           *sql.Tx  // Current transaction
	
	// Prepared statements for performance
	loadSectorStmt *sql.Stmt
	saveSectorStmt *sql.Stmt
}

// NewDatabase creates a new SQLite database instance
func NewDatabase() *SQLiteDatabase {
	// Set up debug logging
	logFile, err := os.OpenFile("twist_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	
	// Set up data logging for parsing tracking
	dataLogFile, err := os.OpenFile("data.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open data log file: %v", err)
	}
	
	logger := log.New(logFile, "[DATABASE] ", log.LstdFlags|log.Lshortfile)
	dataLogger := log.New(dataLogFile, "", log.LstdFlags)
	
	return &SQLiteDatabase{
		logger:     logger,
		dataLogger: dataLogger,
	}
}

// OpenDatabase opens an existing SQLite database (matching TWX method)
func (d *SQLiteDatabase) OpenDatabase(filename string) error {
	if d.dbOpen {
		return fmt.Errorf("database already open")
	}
	
	d.logger.Printf("Opening database: %s", filename)
	
	var err error
	d.db, err = sql.Open("sqlite3", filename+"?_foreign_keys=on")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	
	// Test the connection
	if err = d.db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	
	// Run migrations (this will also validate schema)
	if err = d.runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	
	// Check if database has proper schema
	if err = d.validateSchemaEnhanced(); err != nil {
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
	
	d.logger.Printf("Database opened successfully - %d sectors", d.sectors)
	return nil
}

// CreateDatabase creates a new SQLite database with TWX-compatible schema
func (d *SQLiteDatabase) CreateDatabase(filename string) error {
	d.logger.Printf("Creating database: %s", filename)
	
	var err error
	d.db, err = sql.Open("sqlite3", filename+"?_foreign_keys=on")
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	
	// Create schema
	if err = d.createSchema(); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	
	// Run migrations
	if err = d.runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	
	// Prepare statements for performance
	if err = d.prepareStatements(); err != nil {
		return fmt.Errorf("failed to prepare statements: %w", err)
	}
	
	d.filename = filename
	d.dbOpen = true
	d.sectors = 0
	
	d.logger.Printf("Database created successfully")
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
	d.logger.Printf("Database closed successfully")
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
	
	d.logger.Printf("Loading sector %d", index)
	
	sector := NULLSector()
	
	// Load main sector data
	row := d.loadSectorStmt.QueryRow(index)
	
	var upDate, sportUpDate sql.NullTime
	
	err := row.Scan(
		&sector.Warp[0], &sector.Warp[1], &sector.Warp[2], 
		&sector.Warp[3], &sector.Warp[4], &sector.Warp[5],
		&sector.Constellation, &sector.Beacon, &sector.NavHaz,
		&sector.Density, &sector.Anomaly, &sector.Warps, &sector.Explored,
		&upDate, 
		&sector.SPort.Name, &sector.SPort.Dead, &sector.SPort.ClassIndex,
		&sector.SPort.BuildTime, &sportUpDate,
		&sector.SPort.BuyProduct[0], &sector.SPort.BuyProduct[1], &sector.SPort.BuyProduct[2],
		&sector.SPort.ProductPercent[0], &sector.SPort.ProductPercent[1], &sector.SPort.ProductPercent[2],
		&sector.SPort.ProductAmount[0], &sector.SPort.ProductAmount[1], &sector.SPort.ProductAmount[2],
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
	if sportUpDate.Valid {
		sector.SPort.UpDate = sportUpDate.Time
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
	
	d.logger.Printf("Saving sector %d", index)
	
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
	
	// Save main sector data
	_, err := d.saveSectorStmt.Exec(
		index,
		sector.Warp[0], sector.Warp[1], sector.Warp[2],
		sector.Warp[3], sector.Warp[4], sector.Warp[5],
		sector.Constellation, sector.Beacon, sector.NavHaz,
		sector.Density, sector.Anomaly, sector.Warps, int(sector.Explored),
		sector.UpDate,
		sector.SPort.Name, sector.SPort.Dead, sector.SPort.ClassIndex,
		sector.SPort.BuildTime, sector.SPort.UpDate,
		sector.SPort.BuyProduct[0], sector.SPort.BuyProduct[1], sector.SPort.BuyProduct[2],
		sector.SPort.ProductPercent[0], sector.SPort.ProductPercent[1], sector.SPort.ProductPercent[2],
		sector.SPort.ProductAmount[0], sector.SPort.ProductAmount[1], sector.SPort.ProductAmount[2],
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
	
	// Log all data being saved to data.log for parsing tracking
	d.logSectorData(index, sector)
	
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

// logSectorData logs comprehensive sector data to data.log for parsing tracking
func (d *SQLiteDatabase) logSectorData(index int, sector TSector) {
	// Convert warp array to slice for logging
	var warps []int
	for i := 0; i < sector.Warps; i++ {
		warps = append(warps, sector.Warp[i])
	}

	// Log main sector data
	d.dataLogger.Printf("SECTOR_SAVED: Number=%d, Warps=%v, NavHaz=%d%%, Constellation='%s', Beacon='%s', Density=%d, Anomaly=%t, Explored=%d, UpdateTime='%s'",
		index,
		warps,
		sector.NavHaz,
		sector.Constellation,
		sector.Beacon,
		sector.Density,
		sector.Anomaly,
		int(sector.Explored),
		sector.UpDate.Format("2006-01-02 15:04:05"),
	)

	// Log port data if present
	if sector.SPort.Name != "" || sector.SPort.ClassIndex >= 0 {
		d.dataLogger.Printf("PORT_SAVED: Sector=%d, Name='%s', Dead=%t, Class=%d, BuildTime=%d, BuyProducts=[%t,%t,%t], Percentages=[%d%%,%d%%,%d%%], Amounts=[%d,%d,%d], UpdateTime='%s'",
			index,
			sector.SPort.Name,
			sector.SPort.Dead,
			sector.SPort.ClassIndex,
			sector.SPort.BuildTime,
			sector.SPort.BuyProduct[0], sector.SPort.BuyProduct[1], sector.SPort.BuyProduct[2],
			sector.SPort.ProductPercent[0], sector.SPort.ProductPercent[1], sector.SPort.ProductPercent[2],
			sector.SPort.ProductAmount[0], sector.SPort.ProductAmount[1], sector.SPort.ProductAmount[2],
			sector.SPort.UpDate.Format("2006-01-02 15:04:05"),
		)
	}

	// Log fighters if present
	if sector.Figs.Quantity > 0 {
		figTypeStr := "None"
		switch sector.Figs.FigType {
		case FtToll:
			figTypeStr = "Toll"
		case FtDefensive:
			figTypeStr = "Defensive"
		case FtOffensive:
			figTypeStr = "Offensive"
		}
		d.dataLogger.Printf("FIGHTERS_SAVED: Sector=%d, Quantity=%d, Owner='%s', Type=%s",
			index, sector.Figs.Quantity, sector.Figs.Owner, figTypeStr)
	}

	// Log mines if present
	if sector.MinesArmid.Quantity > 0 {
		d.dataLogger.Printf("MINES_SAVED: Sector=%d, Type=Armid, Quantity=%d, Owner='%s'",
			index, sector.MinesArmid.Quantity, sector.MinesArmid.Owner)
	}
	if sector.MinesLimpet.Quantity > 0 {
		d.dataLogger.Printf("MINES_SAVED: Sector=%d, Type=Limpet, Quantity=%d, Owner='%s'",
			index, sector.MinesLimpet.Quantity, sector.MinesLimpet.Owner)
	}

	// Log ships if present
	for i, ship := range sector.Ships {
		d.dataLogger.Printf("SHIP_SAVED: Sector=%d, Index=%d, Name='%s', Owner='%s', Type='%s', Fighters=%d",
			index, i, ship.Name, ship.Owner, ship.ShipType, ship.Figs)
	}

	// Log traders if present
	for i, trader := range sector.Traders {
		d.dataLogger.Printf("TRADER_SAVED: Sector=%d, Index=%d, Name='%s', ShipType='%s', ShipName='%s', Fighters=%d",
			index, i, trader.Name, trader.ShipType, trader.ShipName, trader.Figs)
	}

	// Log planets if present
	for i, planet := range sector.Planets {
		d.dataLogger.Printf("PLANET_SAVED: Sector=%d, Index=%d, Name='%s'",
			index, i, planet.Name)
	}

	// Log sector variables if present
	for i, sectorVar := range sector.Vars {
		d.dataLogger.Printf("SECTOR_VAR_SAVED: Sector=%d, Index=%d, VarName='%s', Value='%s'",
			index, i, sectorVar.VarName, sectorVar.Value)
	}
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
		varType = 0 // StringType
		stringValue = v
		numberValue = 0
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

	d.logger.Printf("SCRIPT_VAR_SAVED: Name='%s', Type=%d, StringValue='%s', NumberValue=%f",
		name, varType, stringValue, numberValue)

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
	default:
		// Default to string for unknown types
		return stringValue, nil
	}
}