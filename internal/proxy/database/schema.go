package database

import (
	"fmt"
)

// createSchema creates the TWX-compatible SQLite schema
func (d *SQLiteDatabase) createSchema() error {

	// Main sectors table matching TWX TSector exactly
	sectorsTable := `
	CREATE TABLE IF NOT EXISTS sectors (
		sector_index INTEGER PRIMARY KEY,
		
		-- Warp array[1..6] (0-indexed in Go)
		warp1 INTEGER DEFAULT 0,
		warp2 INTEGER DEFAULT 0, 
		warp3 INTEGER DEFAULT 0,
		warp4 INTEGER DEFAULT 0,
		warp5 INTEGER DEFAULT 0,
		warp6 INTEGER DEFAULT 0,
		
		-- Basic sector info
		constellation TEXT DEFAULT '',
		beacon TEXT DEFAULT '',
		nav_haz INTEGER DEFAULT 0,
		density INTEGER DEFAULT -1,
		anomaly BOOLEAN DEFAULT FALSE,
		warps INTEGER DEFAULT 0,
		explored INTEGER DEFAULT 0,
		update_time DATETIME,
		
		-- Embedded SPort data (TPort)
		sport_name TEXT DEFAULT '',
		sport_dead BOOLEAN DEFAULT FALSE,
		sport_class_index INTEGER DEFAULT -1,
		sport_build_time INTEGER DEFAULT 0,
		sport_update DATETIME,
		
		-- Port products array[TProductType] 
		sport_buy_fuel_ore BOOLEAN DEFAULT FALSE,
		sport_buy_organics BOOLEAN DEFAULT FALSE,
		sport_buy_equipment BOOLEAN DEFAULT FALSE,
		sport_percent_fuel_ore INTEGER DEFAULT 0,
		sport_percent_organics INTEGER DEFAULT 0,
		sport_percent_equipment INTEGER DEFAULT 0,
		sport_amount_fuel_ore INTEGER DEFAULT 0,
		sport_amount_organics INTEGER DEFAULT 0,
		sport_amount_equipment INTEGER DEFAULT 0,
		
		-- Space objects (TSpaceObject)
		figs_quantity INTEGER DEFAULT 0,
		figs_owner TEXT DEFAULT '',
		figs_type INTEGER DEFAULT 0,
		
		mines_armid_quantity INTEGER DEFAULT 0,
		mines_armid_owner TEXT DEFAULT '',
		
		mines_limpet_quantity INTEGER DEFAULT 0,
		mines_limpet_owner TEXT DEFAULT ''
	);`

	// Ships table (dynamic list)
	shipsTable := `
	CREATE TABLE IF NOT EXISTS ships (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sector_index INTEGER NOT NULL,
		name TEXT DEFAULT '',
		owner TEXT DEFAULT '',
		ship_type TEXT DEFAULT '',
		fighters INTEGER DEFAULT 0,
		FOREIGN KEY (sector_index) REFERENCES sectors(sector_index) ON DELETE CASCADE
	);`

	// Traders table (dynamic list)
	tradersTable := `
	CREATE TABLE IF NOT EXISTS traders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sector_index INTEGER NOT NULL,
		name TEXT DEFAULT '',
		ship_type TEXT DEFAULT '',
		ship_name TEXT DEFAULT '',
		fighters INTEGER DEFAULT 0,
		FOREIGN KEY (sector_index) REFERENCES sectors(sector_index) ON DELETE CASCADE
	);`

	// Planets table (dynamic list) - enhanced to match parser data
	planetsTable := `
	CREATE TABLE IF NOT EXISTS planets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sector_index INTEGER NOT NULL,
		name TEXT DEFAULT '',
		owner TEXT DEFAULT '',
		fighters INTEGER DEFAULT 0,
		citadel BOOLEAN DEFAULT FALSE,
		stardock BOOLEAN DEFAULT FALSE,
		FOREIGN KEY (sector_index) REFERENCES sectors(sector_index) ON DELETE CASCADE
	);`

	// Sector variables table (dynamic list)
	sectorVarsTable := `
	CREATE TABLE IF NOT EXISTS sector_vars (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sector_index INTEGER NOT NULL,
		var_name TEXT NOT NULL,
		value TEXT DEFAULT '',
		UNIQUE(sector_index, var_name),
		FOREIGN KEY (sector_index) REFERENCES sectors(sector_index) ON DELETE CASCADE
	);`

	// Script variables table (global persistent variables)
	scriptVarsTable := `
	CREATE TABLE IF NOT EXISTS script_vars (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		var_name TEXT NOT NULL UNIQUE,
		var_type INTEGER DEFAULT 0,
		string_value TEXT DEFAULT '',
		number_value REAL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Script variables table with array support (per-script variables)
	scriptVariablesTable := `
	CREATE TABLE IF NOT EXISTS script_variables (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		script_id TEXT NOT NULL,
		var_name TEXT NOT NULL,
		var_type INTEGER NOT NULL,
		var_value TEXT,
		array_size INTEGER DEFAULT 0,
		parent_var_id INTEGER,
		index_path TEXT,  -- JSON array of index values
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (script_id) REFERENCES scripts(script_id),
		FOREIGN KEY (parent_var_id) REFERENCES script_variables(id)
	);`

	// Scripts table (active script management)
	scriptsTable := `
	CREATE TABLE IF NOT EXISTS scripts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		script_id TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		filename TEXT NOT NULL,
		version INTEGER DEFAULT 6,
		is_running BOOLEAN DEFAULT TRUE,
		is_system BOOLEAN DEFAULT FALSE,
		loaded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		stopped_at DATETIME,
		include_scripts TEXT DEFAULT '', -- JSON array of included script names
		description TEXT DEFAULT ''
	);`

	// Script triggers table (Pascal TWX compatible trigger persistence)
	scriptTriggersTable := `
	CREATE TABLE IF NOT EXISTS script_triggers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		script_id TEXT NOT NULL,
		trigger_id TEXT NOT NULL,
		trigger_type INTEGER NOT NULL, -- 1=TextLine, 2=Text, 3=TextOut, 4=Delay, 5=Event, etc.
		pattern TEXT NOT NULL,
		label_name TEXT NOT NULL,
		response TEXT DEFAULT '',
		lifecycle INTEGER DEFAULT -1, -- -1=permanent, >0=limited uses
		is_active BOOLEAN DEFAULT TRUE,
		script_param TEXT DEFAULT '', -- Additional trigger parameters
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (script_id) REFERENCES scripts(script_id),
		UNIQUE(script_id, trigger_id)
	);`

	// Script call stack table (for GOSUB/RETURN persistence across VM instances)
	scriptCallStackTable := `
	CREATE TABLE IF NOT EXISTS script_call_stack (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		script_id TEXT NOT NULL,
		frame_index INTEGER NOT NULL, -- 0-based stack position (0 = bottom, higher = top)
		label TEXT NOT NULL,
		position INTEGER NOT NULL,
		return_addr INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (script_id) REFERENCES scripts(script_id),
		UNIQUE(script_id, frame_index)
	);`

	// Message history table (TWX parser message tracking)
	messageHistoryTable := `
	CREATE TABLE IF NOT EXISTS message_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		message_type INTEGER NOT NULL, -- 0=General, 1=Fighter, 2=Computer, 3=Radio, 4=Fedlink, 5=Planet
		timestamp DATETIME NOT NULL,
		content TEXT NOT NULL,
		sender TEXT DEFAULT '',
		channel INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Player stats table (current player statistics from TWX parser)
	playerStatsTable := `
	CREATE TABLE IF NOT EXISTS player_stats (
		id INTEGER PRIMARY KEY DEFAULT 1, -- Single row table
		turns INTEGER DEFAULT 0,
		credits INTEGER DEFAULT 0,
		fighters INTEGER DEFAULT 0,
		shields INTEGER DEFAULT 0,
		total_holds INTEGER DEFAULT 0,
		ore_holds INTEGER DEFAULT 0,
		org_holds INTEGER DEFAULT 0,
		equ_holds INTEGER DEFAULT 0,
		col_holds INTEGER DEFAULT 0,
		photons INTEGER DEFAULT 0,
		armids INTEGER DEFAULT 0,
		limpets INTEGER DEFAULT 0,
		gen_torps INTEGER DEFAULT 0,
		twarp_type INTEGER DEFAULT 0,
		cloaks INTEGER DEFAULT 0,
		beacons INTEGER DEFAULT 0,
		atomics INTEGER DEFAULT 0,
		corbomite INTEGER DEFAULT 0,
		eprobes INTEGER DEFAULT 0,
		mine_disr INTEGER DEFAULT 0,
		alignment INTEGER DEFAULT 0,
		experience INTEGER DEFAULT 0,
		corp INTEGER DEFAULT 0,
		ship_number INTEGER DEFAULT 0,
		psychic_probe BOOLEAN DEFAULT FALSE,
		planet_scanner BOOLEAN DEFAULT FALSE,
		scan_type INTEGER DEFAULT 0,
		ship_class TEXT DEFAULT '',
		current_sector INTEGER DEFAULT 0,
		player_name TEXT DEFAULT '',
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT single_row CHECK (id = 1)
	);`

	// New dedicated ports table (Phase 2: Database Schema Optimization)
	portsTable := `
	CREATE TABLE IF NOT EXISTS ports (
		sector_index INTEGER PRIMARY KEY,  -- 1:1 relationship with sectors
		name TEXT NOT NULL DEFAULT '',
		class_index INTEGER NOT NULL DEFAULT 0,
		dead BOOLEAN DEFAULT FALSE,
		build_time INTEGER DEFAULT 0,
		
		-- Product information (normalized)
		buy_fuel_ore BOOLEAN DEFAULT FALSE,
		buy_organics BOOLEAN DEFAULT FALSE, 
		buy_equipment BOOLEAN DEFAULT FALSE,
		percent_fuel_ore INTEGER DEFAULT 0,
		percent_organics INTEGER DEFAULT 0,
		percent_equipment INTEGER DEFAULT 0,
		amount_fuel_ore INTEGER DEFAULT 0,
		amount_organics INTEGER DEFAULT 0,
		amount_equipment INTEGER DEFAULT 0,
		
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		
		FOREIGN KEY (sector_index) REFERENCES sectors(sector_index) ON DELETE CASCADE
	);`

	// Create indexes for performance
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_sectors_constellation ON sectors(constellation);`,
		`CREATE INDEX IF NOT EXISTS idx_sectors_beacon ON sectors(beacon);`,
		`CREATE INDEX IF NOT EXISTS idx_sectors_port ON sectors(sport_name) WHERE sport_name != '';`,
		`CREATE INDEX IF NOT EXISTS idx_ships_sector ON ships(sector_index);`,
		`CREATE INDEX IF NOT EXISTS idx_traders_sector ON traders(sector_index);`,
		`CREATE INDEX IF NOT EXISTS idx_planets_sector ON planets(sector_index);`,
		`CREATE INDEX IF NOT EXISTS idx_planets_owner ON planets(owner) WHERE owner != '';`,
		`CREATE INDEX IF NOT EXISTS idx_sector_vars_sector ON sector_vars(sector_index);`,
		`CREATE INDEX IF NOT EXISTS idx_sector_vars_name ON sector_vars(var_name);`,
		`CREATE INDEX IF NOT EXISTS idx_script_vars_name ON script_vars(var_name);`,
		`CREATE INDEX IF NOT EXISTS idx_script_variables_script ON script_variables(script_id);`,
		`CREATE INDEX IF NOT EXISTS idx_script_variables_name ON script_variables(var_name);`,
		`CREATE INDEX IF NOT EXISTS idx_script_variables_parent ON script_variables(parent_var_id);`,
		`CREATE INDEX IF NOT EXISTS idx_scripts_id ON scripts(script_id);`,
		`CREATE INDEX IF NOT EXISTS idx_scripts_running ON scripts(is_running);`,
		`CREATE INDEX IF NOT EXISTS idx_scripts_name ON scripts(name);`,
		`CREATE INDEX IF NOT EXISTS idx_script_triggers_script ON script_triggers(script_id);`,
		`CREATE INDEX IF NOT EXISTS idx_script_triggers_id ON script_triggers(trigger_id);`,
		`CREATE INDEX IF NOT EXISTS idx_script_triggers_active ON script_triggers(is_active);`,
		`CREATE INDEX IF NOT EXISTS idx_script_call_stack_script ON script_call_stack(script_id);`,
		`CREATE INDEX IF NOT EXISTS idx_script_call_stack_frame ON script_call_stack(script_id, frame_index);`,
		`CREATE INDEX IF NOT EXISTS idx_message_history_timestamp ON message_history(timestamp);`,
		`CREATE INDEX IF NOT EXISTS idx_message_history_type ON message_history(message_type);`,
		`CREATE INDEX IF NOT EXISTS idx_message_history_sender ON message_history(sender) WHERE sender != '';`,

		// Optimized ports table indexes
		`CREATE INDEX IF NOT EXISTS idx_ports_name ON ports(name) WHERE name != '';`,
		`CREATE INDEX IF NOT EXISTS idx_ports_class ON ports(class_index);`,
		`CREATE INDEX IF NOT EXISTS idx_ports_building ON ports(build_time) WHERE build_time > 0;`,
		`CREATE INDEX IF NOT EXISTS idx_ports_buying_ore ON ports(buy_fuel_ore) WHERE buy_fuel_ore = TRUE;`,
		`CREATE INDEX IF NOT EXISTS idx_ports_buying_org ON ports(buy_organics) WHERE buy_organics = TRUE;`,
		`CREATE INDEX IF NOT EXISTS idx_ports_buying_equ ON ports(buy_equipment) WHERE buy_equipment = TRUE;`,
		`CREATE INDEX IF NOT EXISTS idx_ports_updated ON ports(updated_at);`,
	}

	// Execute all DDL statements
	statements := []string{sectorsTable, shipsTable, tradersTable, planetsTable, sectorVarsTable, scriptVarsTable, scriptVariablesTable, scriptsTable, scriptTriggersTable, scriptCallStackTable, messageHistoryTable, playerStatsTable, portsTable}
	statements = append(statements, indexes...)

	for _, stmt := range statements {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute schema statement: %w", err)
		}
	}

	return nil
}

// validateSchema checks if the database has the expected schema
func (d *SQLiteDatabase) validateSchema() error {
	// Check if main sectors table exists with expected columns
	query := `
	SELECT COUNT(*) FROM pragma_table_info('sectors') 
	WHERE name IN ('sector_index', 'warp1', 'constellation', 'beacon');`

	var count int
	err := d.db.QueryRow(query).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to validate schema: %w", err)
	}

	if count < 4 {
		return fmt.Errorf("database schema is invalid or incomplete")
	}

	return nil
}

// getSectorCount returns the total number of sectors in the database
func (d *SQLiteDatabase) getSectorCount() (int, error) {
	query := `SELECT COALESCE(MAX(sector_index), 0) FROM sectors;`

	var count int
	err := d.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get sector count: %w", err)
	}

	return count, nil
}

// prepareStatements creates prepared statements for performance
func (d *SQLiteDatabase) prepareStatements() error {
	// Load sector statement (Phase 2: port data removed from sectors)
	loadQuery := `
	SELECT 
		warp1, warp2, warp3, warp4, warp5, warp6,
		constellation, beacon, nav_haz, density, anomaly, warps, explored, update_time,
		figs_quantity, figs_owner, figs_type,
		mines_armid_quantity, mines_armid_owner,
		mines_limpet_quantity, mines_limpet_owner
	FROM sectors WHERE sector_index = ?;`

	var err error
	d.loadSectorStmt, err = d.db.Prepare(loadQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare load sector statement: %w", err)
	}

	// Save sector statement (UPSERT) - Phase 2: port data removed from sectors
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

	d.saveSectorStmt, err = d.db.Prepare(saveQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare save sector statement: %w", err)
	}

	return nil
}

// loadSectorRelatedData loads ships, traders, planets for a sector
func (d *SQLiteDatabase) loadSectorRelatedData(sectorIndex int, sector *TSector) error {
	// Load ships
	shipsQuery := `SELECT name, owner, ship_type, fighters FROM ships WHERE sector_index = ?;`
	rows, err := d.db.Query(shipsQuery, sectorIndex)
	if err != nil {
		return fmt.Errorf("failed to load ships: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ship TShip
		if err := rows.Scan(&ship.Name, &ship.Owner, &ship.ShipType, &ship.Figs); err != nil {
			return fmt.Errorf("failed to scan ship: %w", err)
		}
		sector.Ships = append(sector.Ships, ship)
	}

	// Load traders
	tradersQuery := `SELECT name, ship_type, ship_name, fighters FROM traders WHERE sector_index = ?;`
	rows, err = d.db.Query(tradersQuery, sectorIndex)
	if err != nil {
		return fmt.Errorf("failed to load traders: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var trader TTrader
		if err := rows.Scan(&trader.Name, &trader.ShipType, &trader.ShipName, &trader.Figs); err != nil {
			return fmt.Errorf("failed to scan trader: %w", err)
		}
		sector.Traders = append(sector.Traders, trader)
	}

	// Load planets
	planetsQuery := `SELECT name, owner, fighters, citadel, stardock FROM planets WHERE sector_index = ?;`
	rows, err = d.db.Query(planetsQuery, sectorIndex)
	if err != nil {
		return fmt.Errorf("failed to load planets: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var planet TPlanet
		if err := rows.Scan(&planet.Name, &planet.Owner, &planet.Fighters, &planet.Citadel, &planet.Stardock); err != nil {
			return fmt.Errorf("failed to scan planet: %w", err)
		}
		sector.Planets = append(sector.Planets, planet)
	}

	// Load sector variables
	varsQuery := `SELECT var_name, value FROM sector_vars WHERE sector_index = ?;`
	rows, err = d.db.Query(varsQuery, sectorIndex)
	if err != nil {
		return fmt.Errorf("failed to load sector vars: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var sectorVar TSectorVar
		if err := rows.Scan(&sectorVar.VarName, &sectorVar.Value); err != nil {
			return fmt.Errorf("failed to scan sector var: %w", err)
		}
		sector.Vars = append(sector.Vars, sectorVar)
	}

	return nil
}

// saveSectorRelatedData saves ships, traders, planets for a sector
func (d *SQLiteDatabase) saveSectorRelatedData(sectorIndex int, sector TSector) error {
	// Clear existing related data
	tables := []string{"ships", "traders", "planets", "sector_vars"}
	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s WHERE sector_index = ?;", table)
		if _, err := d.tx.Exec(query, sectorIndex); err != nil {
			return fmt.Errorf("failed to clear %s: %w", table, err)
		}
	}

	// Save ships
	if len(sector.Ships) > 0 {
		shipQuery := `INSERT INTO ships (sector_index, name, owner, ship_type, fighters) VALUES (?, ?, ?, ?, ?);`
		for _, ship := range sector.Ships {
			if _, err := d.tx.Exec(shipQuery, sectorIndex, ship.Name, ship.Owner, ship.ShipType, ship.Figs); err != nil {
				return fmt.Errorf("failed to save ship: %w", err)
			}
		}
	}

	// Save traders
	if len(sector.Traders) > 0 {
		traderQuery := `INSERT INTO traders (sector_index, name, ship_type, ship_name, fighters) VALUES (?, ?, ?, ?, ?);`
		for _, trader := range sector.Traders {
			if _, err := d.tx.Exec(traderQuery, sectorIndex, trader.Name, trader.ShipType, trader.ShipName, trader.Figs); err != nil {
				return fmt.Errorf("failed to save trader: %w", err)
			}
		}
	}

	// Save planets
	if len(sector.Planets) > 0 {
		planetQuery := `INSERT INTO planets (sector_index, name, owner, fighters, citadel, stardock) VALUES (?, ?, ?, ?, ?, ?);`
		for _, planet := range sector.Planets {
			if _, err := d.tx.Exec(planetQuery, sectorIndex, planet.Name, planet.Owner, planet.Fighters, planet.Citadel, planet.Stardock); err != nil {
				return fmt.Errorf("failed to save planet: %w", err)
			}
		}
	}

	// Save sector variables
	if len(sector.Vars) > 0 {
		varQuery := `INSERT INTO sector_vars (sector_index, var_name, value) VALUES (?, ?, ?);`
		for _, sectorVar := range sector.Vars {
			if _, err := d.tx.Exec(varQuery, sectorIndex, sectorVar.VarName, sectorVar.Value); err != nil {
				return fmt.Errorf("failed to save sector var: %w", err)
			}
		}
	}

	return nil
}

// saveSectorCollectionsWithParams saves collections passed as separate parameters (Pascal-compliant approach)
func (d *SQLiteDatabase) saveSectorCollectionsWithParams(sectorIndex int, ships []TShip, traders []TTrader, planets []TPlanet) error {
	// Clear existing related data
	tables := []string{"ships", "traders", "planets", "sector_vars"}
	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s WHERE sector_index = ?;", table)
		if _, err := d.tx.Exec(query, sectorIndex); err != nil {
			return fmt.Errorf("failed to clear %s: %w", table, err)
		}
	}

	// Save ships from parameter
	if len(ships) > 0 {
		shipQuery := `INSERT INTO ships (sector_index, name, owner, ship_type, fighters) VALUES (?, ?, ?, ?, ?);`
		for _, ship := range ships {
			if _, err := d.tx.Exec(shipQuery, sectorIndex, ship.Name, ship.Owner, ship.ShipType, ship.Figs); err != nil {
				return fmt.Errorf("failed to save ship: %w", err)
			}
		}
	}

	// Save traders from parameter
	if len(traders) > 0 {
		traderQuery := `INSERT INTO traders (sector_index, name, ship_type, ship_name, fighters) VALUES (?, ?, ?, ?, ?);`
		for _, trader := range traders {
			if _, err := d.tx.Exec(traderQuery, sectorIndex, trader.Name, trader.ShipType, trader.ShipName, trader.Figs); err != nil {
				return fmt.Errorf("failed to save trader: %w", err)
			}
		}
	}

	// Save planets from parameter
	if len(planets) > 0 {
		planetQuery := `INSERT INTO planets (sector_index, name, owner, fighters, citadel, stardock) VALUES (?, ?, ?, ?, ?, ?);`
		for _, planet := range planets {
			if _, err := d.tx.Exec(planetQuery, sectorIndex, planet.Name, planet.Owner, planet.Fighters, planet.Citadel, planet.Stardock); err != nil {
				return fmt.Errorf("failed to save planet: %w", err)
			}
		}
	}

	//	sectorIndex, len(ships), len(traders), len(planets))

	return nil
}
