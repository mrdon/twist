package database

import (
	"fmt"
	"strings"
)

// Migration represents a database migration
type Migration struct {
	ID          int
	Description string
	SQL         string
}

// migrations contains all database migrations in order
var migrations = []Migration{
	{
		ID:          1,
		Description: "Initial schema creation",
		SQL: `
CREATE TABLE IF NOT EXISTS schema_version (
	version INTEGER PRIMARY KEY,
	applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`,
	},
	{
		ID:          2,
		Description: "Add fighter type support to sectors table (if not exists)",
		SQL: `
-- Check if column exists and add it if it doesn't
-- SQLite doesn't support IF NOT EXISTS for ALTER TABLE, so we'll handle this gracefully`,
	},
	// Future migrations can be added here
}

// runMigrations executes all pending migrations
func (d *SQLiteDatabase) runMigrations() error {
	d.logger.Printf("Checking for database migrations")
	
	// Ensure schema_version table exists
	if err := d.ensureSchemaVersionTable(); err != nil {
		return fmt.Errorf("failed to create schema_version table: %w", err)
	}
	
	// Get current schema version
	currentVersion, err := d.getCurrentSchemaVersion()
	if err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}
	
	d.logger.Printf("Current schema version: %d", currentVersion)
	
	// Apply pending migrations
	for _, migration := range migrations {
		if migration.ID > currentVersion {
			d.logger.Printf("Applying migration %d: %s", migration.ID, migration.Description)
			
			if err := d.applyMigration(migration); err != nil {
				return fmt.Errorf("failed to apply migration %d: %w", migration.ID, err)
			}
			
			d.logger.Printf("Successfully applied migration %d", migration.ID)
		}
	}
	
	d.logger.Printf("All migrations completed")
	return nil
}

// ensureSchemaVersionTable creates the schema_version table if it doesn't exist
func (d *SQLiteDatabase) ensureSchemaVersionTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	
	_, err := d.db.Exec(query)
	return err
}

// getCurrentSchemaVersion returns the current schema version
func (d *SQLiteDatabase) getCurrentSchemaVersion() (int, error) {
	query := `SELECT COALESCE(MAX(version), 0) FROM schema_version;`
	
	var version int
	err := d.db.QueryRow(query).Scan(&version)
	if err != nil {
		return 0, err
	}
	
	return version, nil
}

// applyMigration applies a single migration
func (d *SQLiteDatabase) applyMigration(migration Migration) error {
	// Handle special migrations that need column existence checks
	if migration.ID == 2 {
		return d.applyFighterTypeMigration(migration)
	}
	
	// Start transaction
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Execute migration SQL (handle multiple statements)
	statements := strings.Split(migration.SQL, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute migration statement: %w", err)
		}
	}
	
	// Record migration as applied
	recordQuery := `INSERT INTO schema_version (version) VALUES (?);`
	if _, err := tx.Exec(recordQuery, migration.ID); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}
	
	return nil
}

// applyFighterTypeMigration handles the fighter type column addition safely
func (d *SQLiteDatabase) applyFighterTypeMigration(migration Migration) error {
	// Check if figs_type column already exists
	query := `SELECT COUNT(*) FROM pragma_table_info('sectors') WHERE name = 'figs_type';`
	var count int
	err := d.db.QueryRow(query).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for figs_type column: %w", err)
	}
	
	// Start transaction
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Add column only if it doesn't exist
	if count == 0 {
		alterQuery := `ALTER TABLE sectors ADD COLUMN figs_type INTEGER DEFAULT 0;`
		if _, err := tx.Exec(alterQuery); err != nil {
			return fmt.Errorf("failed to add figs_type column: %w", err)
		}
		d.logger.Printf("Added figs_type column to sectors table")
	} else {
		d.logger.Printf("figs_type column already exists, skipping")
	}
	
	// Record migration as applied
	recordQuery := `INSERT INTO schema_version (version) VALUES (?);`
	if _, err := tx.Exec(recordQuery, migration.ID); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}
	
	return nil
}

// getMigrationStatus returns the status of all migrations
func (d *SQLiteDatabase) getMigrationStatus() ([]MigrationStatus, error) {
	// Get applied migrations
	appliedQuery := `SELECT version FROM schema_version ORDER BY version;`
	rows, err := d.db.Query(appliedQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	appliedVersions := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		appliedVersions[version] = true
	}
	
	// Build status list
	var status []MigrationStatus
	for _, migration := range migrations {
		applied := appliedVersions[migration.ID]
		status = append(status, MigrationStatus{
			ID:          migration.ID,
			Description: migration.Description,
			Applied:     applied,
		})
	}
	
	return status, nil
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	ID          int
	Description string
	Applied     bool
}

// AddMigration adds a new migration (for development/testing)
func AddMigration(id int, description, sql string) {
	migrations = append(migrations, Migration{
		ID:          id,
		Description: description,
		SQL:         sql,
	})
}

// validateSchema performs enhanced schema validation
func (d *SQLiteDatabase) validateSchemaEnhanced() error {
	// Check if main sectors table exists with expected columns
	query := `
	SELECT name FROM pragma_table_info('sectors') 
	WHERE name IN ('sector_index', 'warp1', 'constellation', 'beacon', 'figs_type');`
	
	rows, err := d.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to validate schema: %w", err)
	}
	defer rows.Close()
	
	columnCount := 0
	var columnNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("failed to scan column name: %w", err)
		}
		columnNames = append(columnNames, name)
		columnCount++
	}
	
	if columnCount < 4 {
		return fmt.Errorf("database schema is invalid or incomplete (found columns: %v)", columnNames)
	}
	
	d.logger.Printf("Schema validation passed with %d expected columns", columnCount)
	return nil
}