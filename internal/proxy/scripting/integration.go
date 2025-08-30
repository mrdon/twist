package scripting

import (
	"fmt"
	"strings"
	"twist/internal/log"
	"twist/internal/proxy/database"
	"twist/internal/proxy/interfaces"
	"twist/internal/proxy/scripting/constants"
	"twist/internal/proxy/scripting/types"
	"twist/internal/proxy/scripting/vm"
)

// ProxyInterface defines methods for sending commands to the game server
type ProxyInterface interface {
	SendInput(input string)
	SendOutput(output string) // Send output directly to terminal without routing through input
}

// TerminalInterface defines methods for getting terminal output
type TerminalInterface interface {
	GetLines() []string
}

// GameAdapter adapts the game database to the scripting interface
type GameAdapter struct {
	db              database.Database
	systemConstants *constants.SystemConstants
	sendInput       func(string) // Function to send input (no circular dependency)
	sendOutput      func(string) // Function to send output (no circular dependency)
	terminal        TerminalInterface
	menuManager     interface{} // Terminal menu manager
}

// NewGameAdapter creates a new game adapter
func NewGameAdapter(db database.Database) *GameAdapter {
	log.Info("NewGameAdapter: creating adapter", "db", db)
	adapter := &GameAdapter{db: db}
	// Initialize system constants with self-reference for game interface
	adapter.systemConstants = constants.NewSystemConstants(adapter)
	log.Info("NewGameAdapter: created adapter", "db", adapter.db)
	return adapter
}

// SetProxyFunctions sets the proxy functions for sending commands (no circular dependency)
func (g *GameAdapter) SetProxyFunctions(sendInput func(string), sendOutput func(string)) {
	g.sendInput = sendInput
	g.sendOutput = sendOutput
}

// SetTerminal sets the terminal interface for getting output
func (g *GameAdapter) SetTerminal(terminal TerminalInterface) {
	g.terminal = terminal
}

// SetMenuManager sets the menu manager for script input collection
func (g *GameAdapter) SetMenuManager(menuManager interface{}) {
	g.menuManager = menuManager
}

// GetMenuManager returns the terminal menu manager
func (g *GameAdapter) GetMenuManager() interface{} {
	return g.menuManager
}

// SetDatabase updates the database reference
func (g *GameAdapter) SetDatabase(db database.Database) {
	log.Info("GameAdapter.SetDatabase: changing database", "from", g.db, "to", db)
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
		HasPort:       false,
		PortName:      "",
		PortClass:     0,
		Ships:         make([]types.ShipData, 0),
		Traders:       make([]types.TraderData, 0),
		Planets:       make([]types.PlanetData, 0),
	}

	// Load port data from separate ports table
	port, err := g.db.LoadPort(index)
	if err == nil {
		scriptSector.HasPort = true
		scriptSector.PortName = port.Name
		scriptSector.PortClass = port.ClassIndex
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
	if g.sendInput == nil {
		return fmt.Errorf("sendInput function not available")
	}
	g.sendInput(cmd)
	return nil
}

// SendDirectOutput sends output directly to terminal without routing through input system
func (g *GameAdapter) SendDirectOutput(text string) error {
	if g.sendOutput == nil {
		return fmt.Errorf("sendOutput function not available")
	}
	g.sendOutput(text)
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
	engine        *Engine
	db            database.Database
	gameAdapter   *GameAdapter
	dbProvider    DatabaseProvider // For getting current database when needed
	initialScript string           // Script to load automatically on connection
}

// NewScriptManager creates a new script manager
func NewScriptManager(db database.Database) *ScriptManager {
	log.Info("NewScriptManager: creating", "db", db)
	gameAdapter := NewGameAdapter(db)
	engine := NewEngine(gameAdapter)

	sm := &ScriptManager{
		engine:      engine,
		db:          db,
		gameAdapter: gameAdapter,
	}
	log.Info("NewScriptManager: created", "sm.db", sm.db, "gameAdapter.db", gameAdapter.db)
	return sm
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
	log.Info("getCurrentDatabase", "sm.db", sm.db, "sm.dbProvider", sm.dbProvider)
	if sm.db != nil {
		log.Info("getCurrentDatabase: returning sm.db", "db", sm.db)
		return sm.db
	}
	if sm.dbProvider != nil {
		providerDB := sm.dbProvider.GetDatabase()
		log.Info("getCurrentDatabase: provider returned db", "db", providerDB)
		return providerDB
	}
	log.Info("getCurrentDatabase: returning nil")
	return nil
}

// UpdateDatabase updates the game adapter with the current database
func (sm *ScriptManager) UpdateDatabase() {
	if currentDB := sm.getCurrentDatabase(); currentDB != nil {
		sm.gameAdapter.SetDatabase(currentDB)
	}
}

// SetDatabase updates the script manager's database reference directly
func (sm *ScriptManager) SetDatabase(db database.Database) {
	log.Info("ScriptManager.SetDatabase: updating database", "from", sm.db, "to", db)
	sm.db = db
	// Also update the game adapter immediately
	sm.gameAdapter.SetDatabase(db)
}

// SetupConnections wires the proxy and terminal to the game adapter and sets up engine handlers
func (sm *ScriptManager) SetupConnections(sendInput func(string), sendOutput func(string), terminal TerminalInterface) {
	// Wire the functions to the game adapter (no circular dependency)
	sm.gameAdapter.SetProxyFunctions(sendInput, sendOutput)
	sm.gameAdapter.SetTerminal(terminal)

	// Update the game adapter with current database
	if currentDB := sm.getCurrentDatabase(); currentDB != nil {
		sm.gameAdapter.SetDatabase(currentDB)
	}

	// Set up engine handlers for script output
	sm.engine.SetSendHandler(func(text string) error {
		return sm.gameAdapter.SendCommand(text)
	})

	// Set up output handler (scripts can output to terminal)
	sm.engine.SetOutputHandler(func(text string) error {
		// Script error messages should not be routed through the input system
		// as they would be interpreted as user input by the menu system
		if strings.HasPrefix(text, "Script error") {
			// Send script errors directly to terminal output
			return sm.gameAdapter.SendDirectOutput(text)
		}
		// Regular script output (like ECHO commands) can still go through SendCommand
		return sm.gameAdapter.SendCommand(text)
	})

	// Set up echo handler (local echo for script commands)
	sm.engine.SetEchoHandler(func(text string) error {
		// Echo should display locally only, not send to server
		return sm.gameAdapter.SendDirectOutput(text)
	})
}

// SetupMenuManager sets up the menu manager for script menu commands
func (sm *ScriptManager) SetupMenuManager(menuManager interface{}) {
	sm.gameAdapter.SetMenuManager(menuManager)
}

// GetEngine returns the scripting engine with proper typing
func (sm *ScriptManager) GetEngine() interfaces.ScriptEngine {
	return sm.engine
}

// HasScriptWaitingForInput checks if any script is currently waiting for input
// Returns the script ID and name if found, empty strings if none
func (sm *ScriptManager) HasScriptWaitingForInput() (string, string) {
	runningScripts := sm.engine.GetRunningScriptsInternal()
	for _, script := range runningScripts {
		if script.VM.IsWaitingForInput() {
			return script.ID, script.Name
		}
	}
	return "", ""
}

// ResumeScriptWithInput resumes a script with user input
func (sm *ScriptManager) ResumeScriptWithInput(scriptID, input string) error {
	return sm.engine.ResumeScriptWithInput(scriptID, input)
}

// LoadAndRunScript loads and runs a script file
func (sm *ScriptManager) LoadAndRunScript(filename string) error {
	script, err := sm.engine.LoadScript(filename)
	if err != nil {
		return err
	}

	err = sm.engine.RunScript(script.ID)
	if err != nil {
		return err
	}

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
// Returns (matched, error) - matched=true if any TextLineTrigger fired
func (sm *ScriptManager) ProcessGameLine(line string) (bool, error) {
	return sm.engine.ProcessTextLine(line)
}

// ProcessOutgoingText processes outgoing text through triggers
func (sm *ScriptManager) ProcessOutgoingText(text string) error {
	// Check for ESC key (ASCII 27) to stop running scripts - TWX compatibility
	if len(text) > 0 && text[0] == 27 { // ESC key
		// Stop all running scripts like TWX does
		err := sm.engine.StopAllScripts()
		if err != nil {
			return err
		}
		// Don't process the ESC key further - it was consumed for script termination
		return nil
	}

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
			runningScripts := sm.engine.GetRunningScriptsInternal()
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

// CollectScriptInput handles script input collection via the menu system
// This matches TWX's BeginScriptInput() functionality
func (sm *ScriptManager) CollectScriptInput(prompt string) (string, error) {
	// Get the menu manager from the game adapter
	menuManager := sm.gameAdapter.GetMenuManager()
	if menuManager == nil {
		return "", fmt.Errorf("menu manager not available for script input collection")
	}

	// Try to use the menu manager's input collector
	if inputCollector, ok := menuManager.(interface {
		StartScriptInputCollection(prompt string, callback func(string)) error
	}); ok {
		// Find the script that's waiting for input
		runningScripts := sm.engine.GetRunningScriptsInternal()
		var waitingScript *Script
		for _, script := range runningScripts {
			if script.VM.IsWaitingForInput() {
				waitingScript = script
				break
			}
		}

		if waitingScript == nil {
			return "", fmt.Errorf("no script found waiting for input")
		}

		// Start input collection with callback that resumes the script
		err := inputCollector.StartScriptInputCollection(prompt, func(input string) {
			// Resume the script with the collected input
			sm.engine.ResumeScriptWithInput(waitingScript.ID, input)
		})
		if err != nil {
			return "", err
		}

		// Return immediately - the script will be resumed asynchronously
		// This matches TWX's behavior where getinput pauses execution
		return "", nil
	}

	return "", fmt.Errorf("menu manager does not support script input collection")
}

// SetInitialScript sets the script to load automatically on connection
func (sm *ScriptManager) SetInitialScript(scriptName string) {
	sm.initialScript = scriptName
}

// GetInitialScript returns the initial script name, if any
func (sm *ScriptManager) GetInitialScript() string {
	return sm.initialScript
}

// LoadInitialScript loads the initial script if one is configured
func (sm *ScriptManager) LoadInitialScript() error {
	if sm.initialScript == "" {
		return nil // No initial script configured
	}
	return sm.LoadAndRunScript(sm.initialScript)
}
