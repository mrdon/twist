package scripting

import (
	"fmt"
	"twist/internal/debug"
	"twist/internal/proxy/database"
	"twist/internal/proxy/scripting/constants"
	"twist/internal/proxy/scripting/types"
	"twist/internal/proxy/scripting/vm"
)

// ProxyInterface defines methods for sending commands to the game server
type ProxyInterface interface {
	SendInput(input string)
}

// TerminalInterface defines methods for getting terminal output
type TerminalInterface interface {
	GetLines() []string
}

// GameAdapter adapts the game database to the scripting interface
type GameAdapter struct {
	db              database.Database
	systemConstants *constants.SystemConstants
	proxy           ProxyInterface
	terminal        TerminalInterface
}

// NewGameAdapter creates a new game adapter
func NewGameAdapter(db database.Database) *GameAdapter {
	adapter := &GameAdapter{db: db}
	// Initialize system constants with self-reference for game interface
	adapter.systemConstants = constants.NewSystemConstants(adapter)
	return adapter
}

// SetProxy sets the proxy interface for sending commands
func (g *GameAdapter) SetProxy(proxy ProxyInterface) {
	g.proxy = proxy
}

// SetTerminal sets the terminal interface for getting output
func (g *GameAdapter) SetTerminal(terminal TerminalInterface) {
	g.terminal = terminal
}

// SetDatabase updates the database reference
func (g *GameAdapter) SetDatabase(db database.Database) {
	g.db = db
}

// GetSector implements GameInterface
func (g *GameAdapter) GetSector(index int) (types.SectorData, error) {
	sector, err := g.db.LoadSector(index)
	if err != nil {
		return types.SectorData{}, err
	}
	
	// Convert database sector to script sector data
	scriptSector := types.SectorData{
		Number:        index, // Use the index parameter
		Warps:         make([]int, 0),
		NavHaz:        sector.NavHaz,
		Constellation: sector.Constellation,
		Beacon:        sector.Beacon,
		Density:       sector.Density,
		Anomaly:       sector.Anomaly,
		Explored:      int(sector.Explored),
		HasPort:       sector.SPort.ClassIndex > 0,
		PortName:      sector.SPort.Name,
		PortClass:     sector.SPort.ClassIndex,
		Ships:         make([]types.ShipData, 0),
		Traders:       make([]types.TraderData, 0),
		Planets:       make([]types.PlanetData, 0),
	}
	
	// Copy warps (TWX uses 1-6 indexing, we convert to 0-based slice)
	for i := 0; i < 6; i++ {
		if sector.Warp[i] > 0 {
			scriptSector.Warps = append(scriptSector.Warps, sector.Warp[i])
		}
	}
	
	// Convert ships
	for _, ship := range sector.Ships {
		scriptShip := types.ShipData{
			Name:     ship.Name,
			Owner:    ship.Owner,
			ShipType: ship.ShipType,
			Fighters: ship.Figs,
		}
		scriptSector.Ships = append(scriptSector.Ships, scriptShip)
	}
	
	// Convert traders
	for _, trader := range sector.Traders {
		scriptTrader := types.TraderData{
			Name:     trader.Name,
			ShipType: trader.ShipType,
			ShipName: trader.ShipName,
			Fighters: trader.Figs,
		}
		scriptSector.Traders = append(scriptSector.Traders, scriptTrader)
	}
	
	// Convert planets
	for _, planet := range sector.Planets {
		scriptPlanet := types.PlanetData{
			Name: planet.Name,
		}
		scriptSector.Planets = append(scriptSector.Planets, scriptPlanet)
	}
	
	return scriptSector, nil
}

// SetSectorParameter implements GameInterface
func (g *GameAdapter) SetSectorParameter(sector int, name, value string) error {
	// TODO: Implement parameter setting
	return fmt.Errorf("SetSectorParameter not implemented")
}

// GetSectorParameter implements GameInterface
func (g *GameAdapter) GetSectorParameter(sector int, name string) (string, error) {
	// TODO: Implement parameter getting
	return "", fmt.Errorf("GetSectorParameter not implemented")
}

// GetCourse implements GameInterface
func (g *GameAdapter) GetCourse(from, to int) ([]int, error) {
	// TODO: Implement course calculation
	return []int{from, to}, nil
}

// GetDistance implements GameInterface
func (g *GameAdapter) GetDistance(from, to int) (int, error) {
	// TODO: Implement distance calculation
	return 1, nil
}

// GetAllCourses implements GameInterface
func (g *GameAdapter) GetAllCourses(from int) (map[int][]int, error) {
	// TODO: Implement all courses calculation
	return make(map[int][]int), nil
}

// GetNearestWarps implements GameInterface
func (g *GameAdapter) GetNearestWarps(sector int, count int) ([]int, error) {
	// TODO: Implement nearest warps calculation
	return []int{}, nil
}

// GetCurrentSector implements GameInterface
func (g *GameAdapter) GetCurrentSector() int {
	// TODO: Get current sector from game state
	return 1
}

// GetCurrentPrompt implements GameInterface
func (g *GameAdapter) GetCurrentPrompt() string {
	// TODO: Get current prompt from game state
	return "Command [TL=00:00:00]:"
}

// SendCommand implements GameInterface
func (g *GameAdapter) SendCommand(cmd string) error {
	if g.proxy == nil {
		return fmt.Errorf("proxy not available")
	}
	g.proxy.SendInput(cmd)
	return nil
}

// GetLastOutput implements GameInterface
func (g *GameAdapter) GetLastOutput() string {
	if g.terminal == nil {
		return ""
	}
	lines := g.terminal.GetLines()
	if len(lines) == 0 {
		return ""
	}
	// Return the last line of output
	return lines[len(lines)-1]
}

// GetDatabase implements GameInterface
func (g *GameAdapter) GetDatabase() interface{} {
	return g.db
}

// SaveScriptVariable implements GameInterface
func (g *GameAdapter) SaveScriptVariable(name string, value *types.Value) error {
	// Like Pascal TWX, save individual variables with simple values
	// Arrays are handled by saving each element separately with its full path
	
	switch value.Type {
	case types.StringType:
		return g.db.SaveScriptVariable(name, value.String)
	case types.NumberType:
		return g.db.SaveScriptVariable(name, value.Number)
	case types.ArrayType:
		// For arrays, save each element individually with its full path
		// This matches Pascal TWX behavior where each TVarParam is stored separately
		for index, element := range value.Array {
			elementName := name + "[" + index + "]"
			if err := g.SaveScriptVariable(elementName, element); err != nil {
				return err
			}
		}
		// Save array metadata (size) separately if needed
		if value.ArraySize > 0 {
			return g.db.SaveScriptVariable(name+"[ARRAYSIZE]", value.ArraySize)
		}
		return nil
	default:
		return g.db.SaveScriptVariable(name, value.ToString())
	}
}

// LoadScriptVariable implements GameInterface
func (g *GameAdapter) LoadScriptVariable(name string) (*types.Value, error) {
	// Like Pascal TWX, load individual variables with simple values
	// Arrays are handled by loading individual elements by their full path
	
	dbValue, err := g.db.LoadScriptVariable(name)
	if err != nil {
		return nil, err
	}
	
	// Convert database value back to Value type (simple values only)
	switch v := dbValue.(type) {
	case string:
		// Check if this was stored as an array element - if so, just return the clean value
		// The key insight: array elements are stored individually, no special processing needed
		return &types.Value{
			Type:   types.StringType,
			String: v,
		}, nil
	case float64:
		return &types.Value{
			Type:   types.NumberType,
			Number: v,
		}, nil
	case int:
		return &types.Value{
			Type:   types.NumberType,
			Number: float64(v),
		}, nil
	default:
		// Default to string for unknown types
		return &types.Value{
			Type:   types.StringType,
			String: fmt.Sprintf("%v", v),
		}, nil
	}
}

// GetSystemConstants implements GameInterface
func (g *GameAdapter) GetSystemConstants() types.SystemConstantsInterface {
	return g.systemConstants
}

// ScriptManager provides high-level script management
// DatabaseProvider interface for getting the current database
type DatabaseProvider interface {
	GetDatabase() database.Database
}

type ScriptManager struct {
	engine         *Engine
	db             database.Database
	gameAdapter    *GameAdapter
	dbProvider     DatabaseProvider // For getting current database when needed
}

// NewScriptManager creates a new script manager
func NewScriptManager(db database.Database) *ScriptManager {
	gameAdapter := NewGameAdapter(db)
	engine := NewEngine(gameAdapter)
	
	return &ScriptManager{
		engine:      engine,
		db:          db,
		gameAdapter: gameAdapter,
	}
}

// NewScriptManagerWithProvider creates a new script manager that can request databases dynamically
func NewScriptManagerWithProvider(dbProvider DatabaseProvider) *ScriptManager {
	// Create a basic game adapter without a database initially
	// The adapter will request the database when needed
	gameAdapter := NewGameAdapter(nil)
	engine := NewEngine(gameAdapter)
	
	return &ScriptManager{
		engine:      engine,
		db:          nil, // Will be requested from provider when needed
		gameAdapter: gameAdapter,
		dbProvider:  dbProvider,
	}
}

// getCurrentDatabase returns the current database, either from direct reference or provider
func (sm *ScriptManager) getCurrentDatabase() database.Database {
	if sm.db != nil {
		return sm.db
	}
	if sm.dbProvider != nil {
		return sm.dbProvider.GetDatabase()
	}
	return nil
}

// UpdateDatabase updates the game adapter with the current database
func (sm *ScriptManager) UpdateDatabase() {
	if currentDB := sm.getCurrentDatabase(); currentDB != nil {
		sm.gameAdapter.SetDatabase(currentDB)
	}
}

// SetupConnections wires the proxy and terminal to the game adapter and sets up engine handlers
func (sm *ScriptManager) SetupConnections(proxy ProxyInterface, terminal TerminalInterface) {
	// Wire the proxy and terminal to the game adapter
	sm.gameAdapter.SetProxy(proxy)
	sm.gameAdapter.SetTerminal(terminal)
	
	// Update the game adapter with current database
	if currentDB := sm.getCurrentDatabase(); currentDB != nil {
		sm.gameAdapter.SetDatabase(currentDB)
	}
	
	// Set up engine handlers for script output
	sm.engine.SetSendHandler(func(text string) error {
		debug.Log("ScriptManager sendHandler called with: %q", text)
		return sm.gameAdapter.SendCommand(text)
	})
	
	// Set up output handler (scripts can output to terminal)
	sm.engine.SetOutputHandler(func(text string) error {
		// For now, scripts output via the same proxy input mechanism
		// This could be enhanced to have a separate output channel
		return sm.gameAdapter.SendCommand(text)
	})
	
	// Set up echo handler (local echo for script commands)
	sm.engine.SetEchoHandler(func(text string) error {
		// Echo is typically for local display, for now we'll send it as output
		return sm.gameAdapter.SendCommand(text)
	})
}

// GetEngine returns the scripting engine as interface{} to avoid import cycles
func (sm *ScriptManager) GetEngine() interface{} {
	return sm.engine
}

// LoadAndRunScript loads and runs a script file
func (sm *ScriptManager) LoadAndRunScript(filename string) error {
	debug.Log("ScriptManager.LoadAndRunScript called with filename: %s", filename)
	script, err := sm.engine.LoadScript(filename)
	if err != nil {
		debug.Log("Failed to load script %s: %v", filename, err)
		return err
	}
	
	debug.Log("Script %s loaded successfully with ID: %s", filename, script.ID)
	err = sm.engine.RunScript(script.ID)
	if err != nil {
		debug.Log("Failed to run script %s (ID: %s): %v", filename, script.ID, err)
		return err
	}
	
	debug.Log("Script %s (ID: %s) started successfully", filename, script.ID)
	return nil
}

// ExecuteCommand executes a single script command
func (sm *ScriptManager) ExecuteCommand(command string) error {
	return sm.engine.ExecuteScriptString(command, "command")
}

// ProcessGameText processes incoming game text through triggers
func (sm *ScriptManager) ProcessGameText(text string) error {
	return sm.engine.ProcessText(text)
}

// ProcessGameLine processes incoming game line through triggers
func (sm *ScriptManager) ProcessGameLine(line string) error {
	return sm.engine.ProcessTextLine(line)
}

// ProcessOutgoingText processes outgoing text through triggers
func (sm *ScriptManager) ProcessOutgoingText(text string) error {
	return sm.engine.ProcessTextOut(text)
}


// ActivateTriggers activates script triggers (mirrors Pascal TWXInterpreter.ActivateTriggers)
func (sm *ScriptManager) ActivateTriggers() error {
	// In Pascal TWX, ActivateTriggers processes delay triggers and reactivates disabled triggers
	// For now, we'll process delay triggers which is the main functionality
	return sm.engine.GetTriggerManager().ProcessDelayTriggers()
}

// ProcessAutoText processes auto text events (mirrors Pascal TWXInterpreter.AutoTextEvent)
func (sm *ScriptManager) ProcessAutoText(text string) error {
	// Process auto text triggers - these are triggers that automatically respond to text
	triggerManager := sm.engine.GetTriggerManager()
	
	// Get all auto text triggers and process them
	autoTextTriggers := triggerManager.GetTriggersByType(types.TriggerAutoText)
	
	for _, trigger := range autoTextTriggers {
		if trigger.Matches(text) {
			// Auto text triggers need a VM context to execute
			// We'll use the first running script's VM, or create a temporary one
			runningScripts := sm.engine.GetRunningScripts()
			var vmInterface types.VMInterface
			if len(runningScripts) > 0 {
				vmInterface = runningScripts[0].VM
			} else {
				// Create a temporary VM for executing the trigger
				vmInterface = vm.NewVirtualMachine(sm.gameAdapter)
			}
			
			if err := trigger.Execute(vmInterface); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// Stop stops all scripts and cleans up
func (sm *ScriptManager) Stop() error {
	return sm.engine.StopAllScripts()
}

// GetStatus returns script engine status
func (sm *ScriptManager) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"total_scripts":   sm.engine.GetScriptCount(),
		"running_scripts": sm.engine.GetRunningScriptCount(),
		"trigger_count":   sm.engine.GetTriggerManager().GetTriggerCount(),
	}
}