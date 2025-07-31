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
	
	return nil
}

// CreateDatabase creates a new SQLite database with TWX-compatible schema
func (d *SQLiteDatabase) CreateDatabase(filename string) error {
	
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