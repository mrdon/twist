package manager

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"twist/internal/database"
)

// ScriptInfo represents metadata about a loaded script
type ScriptInfo struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Filename       string     `json:"filename"`
	Version        int        `json:"version"`
	Running        bool       `json:"is_running"`
	System         bool       `json:"is_system"`
	LoadedAt       time.Time  `json:"loaded_at"`
	StoppedAt      *time.Time `json:"stopped_at,omitempty"`
	IncludeScripts []string   `json:"include_scripts"`
	Description    string     `json:"description"`
}

// ScriptManager manages script lifecycle, loading, and metadata
type ScriptManager struct {
	db              database.Database
	scripts         map[string]*ScriptInfo // scriptID -> ScriptInfo
	systemScriptDir string
}

// NewScriptManager creates a new script manager
func NewScriptManager(db database.Database) *ScriptManager {
	return &ScriptManager{
		db:              db,
		scripts:         make(map[string]*ScriptInfo),
		systemScriptDir: "system", // Default system script directory
	}
}

// SetSystemScriptDirectory sets the directory for system scripts
func (sm *ScriptManager) SetSystemScriptDirectory(dir string) {
	sm.systemScriptDir = dir
}

// LoadScript loads and registers a script
func (sm *ScriptManager) LoadScript(filename string, isSystem bool) (*ScriptInfo, error) {
	// Generate unique script ID based on filename and timestamp
	scriptID := sm.generateScriptID(filename)
	
	// Extract script name from filename
	name := filepath.Base(filename)
	if ext := filepath.Ext(name); ext != "" {
		name = strings.TrimSuffix(name, ext)
	}
	
	// TODO: Read actual script file and parse header for version and description
	// For now, use defaults based on Pascal ScriptCmp.pas
	version := 6 // COMPILED_SCRIPT_VERSION from Pascal
	description := ""
	includeScripts := []string{}
	
	scriptInfo := &ScriptInfo{
		ID:             scriptID,
		Name:           name,
		Filename:       filename,
		Version:        version,
		Running:        true,
		System:         isSystem,
		LoadedAt:       time.Now(),
		IncludeScripts: includeScripts,
		Description:    description,
	}
	
	// Store in memory
	sm.scripts[scriptID] = scriptInfo
	
	// Persist to database
	if err := sm.saveScriptToDB(scriptInfo); err != nil {
		// Remove from memory if database save fails
		delete(sm.scripts, scriptID)
		return nil, fmt.Errorf("failed to save script to database: %w", err)
	}
	
	return scriptInfo, nil
}

// StopScript stops a running script
func (sm *ScriptManager) StopScript(scriptID string) error {
	scriptInfo, exists := sm.scripts[scriptID]
	if !exists {
		return fmt.Errorf("script %s not found", scriptID)
	}
	
	if !scriptInfo.Running {
		return fmt.Errorf("script %s is already stopped", scriptID)
	}
	
	// Mark as stopped
	now := time.Now()
	scriptInfo.Running = false
	scriptInfo.StoppedAt = &now
	
	// Update in database
	if err := sm.updateScriptInDB(scriptInfo); err != nil {
		return fmt.Errorf("failed to update script in database: %w", err)
	}
	
	return nil
}

// StopScriptByName stops a script by name (supports partial matching)
func (sm *ScriptManager) StopScriptByName(name string) error {
	var found *ScriptInfo
	
	// First try exact name match
	for _, script := range sm.scripts {
		if script.Name == name && script.Running {
			found = script
			break
		}
	}
	
	// If not found, try partial match
	if found == nil {
		for _, script := range sm.scripts {
			if strings.Contains(strings.ToLower(script.Name), strings.ToLower(name)) && script.Running {
				found = script
				break
			}
		}
	}
	
	if found == nil {
		return fmt.Errorf("no running script found with name matching %s", name)
	}
	
	return sm.StopScript(found.ID)
}

// GetActiveScripts returns all currently running scripts
func (sm *ScriptManager) GetActiveScripts() []*ScriptInfo {
	var active []*ScriptInfo
	for _, script := range sm.scripts {
		if script.Running {
			active = append(active, script)
		}
	}
	return active
}

// GetScriptByID returns script info by ID
func (sm *ScriptManager) GetScriptByID(scriptID string) (*ScriptInfo, bool) {
	script, exists := sm.scripts[scriptID]
	return script, exists
}

// GetScriptByName returns script info by name (exact match)
func (sm *ScriptManager) GetScriptByName(name string) (*ScriptInfo, bool) {
	for _, script := range sm.scripts {
		if script.Name == name {
			return script, true
		}
	}
	return nil, false
}

// GetScriptVersion returns the version of a script
func (sm *ScriptManager) GetScriptVersion(name string) (string, error) {
	script, exists := sm.GetScriptByName(name)
	if !exists {
		// Try to read from file if not loaded
		// TODO: Implement file reading for unloaded scripts
		return "6", nil // Default version
	}
	
	return fmt.Sprintf("%d", script.Version), nil
}

// LoadSystemScript loads and executes a system script
func (sm *ScriptManager) LoadSystemScript(name string) error {
	// Construct full path to system script
	systemPath := filepath.Join(sm.systemScriptDir, name)
	if !strings.HasSuffix(systemPath, ".twx") {
		systemPath += ".twx"
	}
	
	// TODO: Check if file exists and is readable
	// TODO: Load and execute the system script
	
	// For now, just register it as a system script
	_, err := sm.LoadScript(systemPath, true)
	return err
}

// RestoreFromDatabase loads script info from database on startup
func (sm *ScriptManager) RestoreFromDatabase() error {
	query := `
	SELECT script_id, name, filename, version, is_running, is_system, 
	       loaded_at, stopped_at, include_scripts, description
	FROM scripts
	ORDER BY loaded_at;`
	
	rows, err := sm.db.GetDB().Query(query)
	if err != nil {
		return fmt.Errorf("failed to query scripts: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var script ScriptInfo
		var loadedAtStr string
		var stoppedAtStr *string
		var includeScriptsJSON string
		
		err := rows.Scan(
			&script.ID, &script.Name, &script.Filename, &script.Version,
			&script.Running, &script.System, &loadedAtStr, &stoppedAtStr,
			&includeScriptsJSON, &script.Description,
		)
		if err != nil {
			return fmt.Errorf("failed to scan script row: %w", err)
		}
		
		// Parse timestamps - try multiple formats
		if script.LoadedAt, err = time.Parse("2006-01-02 15:04:05", loadedAtStr); err != nil {
			// Try RFC3339 format
			if script.LoadedAt, err = time.Parse(time.RFC3339, loadedAtStr); err != nil {
				return fmt.Errorf("failed to parse loaded_at: %w", err)
			}
		}
		
		if stoppedAtStr != nil {
			if stoppedAt, err := time.Parse("2006-01-02 15:04:05", *stoppedAtStr); err != nil {
				// Try RFC3339 format
				if stoppedAt, err := time.Parse(time.RFC3339, *stoppedAtStr); err == nil {
					script.StoppedAt = &stoppedAt
				}
			} else {
				script.StoppedAt = &stoppedAt
			}
		}
		
		// Parse include scripts JSON
		if includeScriptsJSON != "" {
			if err := json.Unmarshal([]byte(includeScriptsJSON), &script.IncludeScripts); err != nil {
				// If JSON parsing fails, initialize empty array
				script.IncludeScripts = []string{}
			}
		}
		
		// Store in memory
		sm.scripts[script.ID] = &script
	}
	
	return rows.Err()
}

// generateScriptID creates a unique ID for a script
func (sm *ScriptManager) generateScriptID(filename string) string {
	base := filepath.Base(filename)
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s_%d", base, timestamp)
}

// saveScriptToDB saves script info to database
func (sm *ScriptManager) saveScriptToDB(script *ScriptInfo) error {
	includeScriptsJSON, _ := json.Marshal(script.IncludeScripts)
	
	query := `
	INSERT INTO scripts (script_id, name, filename, version, is_running, is_system, 
	                    loaded_at, include_scripts, description)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`
	
	_, err := sm.db.GetDB().Exec(query,
		script.ID, script.Name, script.Filename, script.Version,
		script.Running, script.System, script.LoadedAt.Format("2006-01-02 15:04:05"),
		string(includeScriptsJSON), script.Description,
	)
	
	return err
}

// updateScriptInDB updates script info in database
func (sm *ScriptManager) updateScriptInDB(script *ScriptInfo) error {
	includeScriptsJSON, _ := json.Marshal(script.IncludeScripts)
	
	var stoppedAtStr *string
	if script.StoppedAt != nil {
		str := script.StoppedAt.Format("2006-01-02 15:04:05")
		stoppedAtStr = &str
	}
	
	query := `
	UPDATE scripts 
	SET name = ?, filename = ?, version = ?, is_running = ?, is_system = ?,
	    stopped_at = ?, include_scripts = ?, description = ?
	WHERE script_id = ?;`
	
	_, err := sm.db.GetDB().Exec(query,
		script.Name, script.Filename, script.Version, script.Running, script.System,
		stoppedAtStr, string(includeScriptsJSON), script.Description, script.ID,
	)
	
	return err
}

// Implementation of types.ScriptInterface for ScriptInfo
func (si *ScriptInfo) GetID() string {
	return si.ID
}

func (si *ScriptInfo) GetFilename() string {
	return si.Filename
}

func (si *ScriptInfo) GetName() string {
	return si.Name
}

func (si *ScriptInfo) IsRunning() bool {
	return si.Running
}

func (si *ScriptInfo) IsSystem() bool {
	return si.System
}

func (si *ScriptInfo) Stop() error {
	// This would be called by the script manager
	// For now, just mark as stopped locally
	si.Running = false
	now := time.Now()
	si.StoppedAt = &now
	return nil
}