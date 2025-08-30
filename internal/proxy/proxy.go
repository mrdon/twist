package proxy

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	"twist/internal/api"
	"twist/internal/log"
	"twist/internal/proxy/database"
	"twist/internal/proxy/input"
	"twist/internal/proxy/menu"
	"twist/internal/proxy/scripting"
	"twist/internal/proxy/streaming"
)

// ProxyState interface for state pattern implementation
type ProxyState interface {
	// Core operations that vary by connection state
	SendToTUI(output string) error   // Script output → pipeline → TUI
	SendToServer(input string) error // User input → script processing → server
	IsConnected() bool
	GetParser() *streaming.TWXParser

	// Internal operations for I/O handlers
	writeServerData(data string) error         // Direct write to server connection
	readServerData(buffer []byte) (int, error) // Direct read from server connection
	processServerData(data []byte)             // Process server data → pipeline → TUI

	// Resource cleanup
	Close() error
}

// DisconnectedState represents the proxy when not connected to a server
type DisconnectedState struct{}

func NewDisconnectedState() *DisconnectedState {
	return &DisconnectedState{}
}

func (s *DisconnectedState) SendToTUI(output string) error {
	// Drop output when disconnected (current behavior)
	return nil
}

func (s *DisconnectedState) SendToServer(input string) error {
	// No-op when disconnected
	return nil
}

func (s *DisconnectedState) IsConnected() bool {
	return false
}

func (s *DisconnectedState) GetParser() *streaming.TWXParser {
	return nil
}

func (s *DisconnectedState) writeServerData(data string) error {
	return fmt.Errorf("not connected")
}

func (s *DisconnectedState) readServerData(buffer []byte) (int, error) {
	return 0, fmt.Errorf("not connected")
}

func (s *DisconnectedState) processServerData(data []byte) {
	// No-op when disconnected
}

func (s *DisconnectedState) Close() error {
	return nil // Nothing to close
}

// ConnectedState represents the proxy when connected to a server
type ConnectedState struct {
	// Network components
	conn     net.Conn
	reader   *bufio.Reader
	writer   *bufio.Writer
	pipeline *streaming.Pipeline

	// Processing components - always present when connected
	scriptManager *scripting.ScriptManager
	gameDetector  *GameDetector
}

func NewConnectedState(conn net.Conn, reader *bufio.Reader, writer *bufio.Writer, pipeline *streaming.Pipeline, scriptManager *scripting.ScriptManager, gameDetector *GameDetector) *ConnectedState {
	return &ConnectedState{
		conn:          conn,
		reader:        reader,
		writer:        writer,
		pipeline:      pipeline,
		scriptManager: scriptManager,
		gameDetector:  gameDetector,
	}
}

func (s *ConnectedState) SendToTUI(output string) error {
	data := []byte(output)
	if s.pipeline != nil {
		s.pipeline.InjectTUIData(data)
	}
	return nil
}

func (s *ConnectedState) SendToServer(input string) error {
	// Process through script manager - no nil check needed, always present
	s.scriptManager.ProcessOutgoingText(input)

	// Process through game detector - no nil check needed, always present
	s.gameDetector.ProcessUserInput(input)

	// Then write directly to server
	return s.writeServerData(input)
}

func (s *ConnectedState) IsConnected() bool {
	return true
}

func (s *ConnectedState) GetParser() *streaming.TWXParser {
	if s.pipeline == nil {
		return nil
	}
	return s.pipeline.GetParser()
}

func (s *ConnectedState) writeServerData(data string) error {
	_, err := s.writer.WriteString(data)
	if err != nil {
		return err
	}
	return s.writer.Flush()
}

func (s *ConnectedState) readServerData(buffer []byte) (int, error) {
	return s.reader.Read(buffer)
}

func (s *ConnectedState) processServerData(data []byte) {
	if s.pipeline != nil {
		s.pipeline.Write(data)
	}
}

func (s *ConnectedState) Close() error {
	// Stop pipeline first
	if s.pipeline != nil {
		s.pipeline.Stop()
	}

	// Close connection
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

type Proxy struct {
	// State pattern - holds ProxyState with mutex protection
	stateMu sync.RWMutex
	state   ProxyState

	// Channels for communication
	outputChan chan string
	inputChan  chan string
	errorChan  chan error

	// Core components
	scriptManager *scripting.ScriptManager
	db            database.Database

	// Terminal menu system
	terminalMenuManager *menu.TerminalMenuManager

	// Direct TuiAPI reference
	tuiAPI api.TuiAPI

	// Script input collector - reuses menu input collector for consistency
	scriptInputCollector *input.InputCollector

	// Game detection
	gameDetector *GameDetector

	// Connection tracking for callbacks
	currentAddress string // Track address for OnConnectionStatusChanged callbacks
	currentHost    string // Track hostname for database naming
	currentPort    string // Track port for database naming

	// Game state tracking (Phase 4.3) - based on parser CurrentSectorIndex
	currentSector int    // Track current sector number (from parser)
	playerName    string // Track current player name

	// Input handler state
	inputHandlerStarted bool
}

// State helper methods
func (p *Proxy) getState() ProxyState {
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()

	state := p.state
	if state == nil {
		return NewDisconnectedState()
	}
	return state
}

func (p *Proxy) setState(newState ProxyState) {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()

	oldState := p.state
	p.state = newState

	// Close old state's resources after storing new state
	if oldState != nil {
		oldState.Close()
	}
}

func New(conn net.Conn, address string, tuiAPI api.TuiAPI, options *api.ConnectOptions) *Proxy {
	// Parse address for database naming
	parts := strings.Split(address, ":")
	var currentHost, currentPort string
	if len(parts) >= 2 {
		currentHost = parts[0]
		currentPort = parts[1]
	} else {
		currentHost = address
		currentPort = "23"
	}

	// Create game detector with connection info
	connInfo := ConnectionInfo{Host: currentHost, Port: currentPort}
	gameDetector := NewGameDetector(connInfo)

	// Initialize database
	var db database.Database
	if options.DatabasePath != "" {
		// Use forced database path
		log.Info("Using forced database path", "path", options.DatabasePath)
		db = database.NewDatabase()
		if err := db.CreateDatabase(options.DatabasePath); err != nil {
			if err := db.OpenDatabase(options.DatabasePath); err != nil {
				panic(fmt.Errorf("failed to load forced database %s: %w", options.DatabasePath, err))
			}
		}
	}
	// Note: If no forced database, it will be set up via game detector callbacks

	// Create readers and writer for the connection
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	writerFunc := func(data []byte) error {
		_, err := writer.Write(data)
		if err != nil {
			return err
		}
		return writer.Flush()
	}

	p := &Proxy{
		outputChan:     make(chan string, 100),
		inputChan:      make(chan string, 100),
		errorChan:      make(chan error, 100),
		db:             db,
		tuiAPI:         tuiAPI,
		gameDetector:   gameDetector,
		currentAddress: address,
		currentHost:    currentHost,
		currentPort:    currentPort,
	}

	// Initialize terminal menu manager with function dependencies (no circular reference)
	p.terminalMenuManager = menu.NewTerminalMenuManager(
		p.injectTUIData,
		func() menu.ScriptManagerInterface { return p.scriptManager },
		func() interface{} { return p.db },
		p.SendInput,
		p.SendToServer,
	)

	// Initialize script input collector - reuses same logic as menu input
	p.scriptInputCollector = input.NewInputCollector(func(output string) {
		// Echo script input to screen via TuiAPI
		if p.tuiAPI != nil {
			p.tuiAPI.OnData([]byte(output))
		}
	})

	// Create script manager with direct database access
	p.scriptManager = scripting.NewScriptManager(p.db)

	// Setup script manager with function injection (no circular dependency)
	p.scriptManager.SetupConnections(p.SendInput, p.SendToTUI, nil)

	// Setup menu manager for script menu commands
	p.scriptManager.SetupMenuManager(p.terminalMenuManager)

	// Set up game detector callbacks to update database and notify TUI when loaded
	gameDetector.SetDatabaseLoadedCallback(p.onDatabaseLoaded)
	gameDetector.SetDatabaseStateChangedCallback(p.onDatabaseStateChanged)

	// Create pipeline with established connection (immutable)
	pipeline := streaming.NewPipeline(p.tuiAPI, func() database.Database { return p.db }, p.scriptManager, p, p.gameDetector, writerFunc)

	// Create connected state with pipeline
	connectedState := NewConnectedState(conn, reader, writer, pipeline, p.scriptManager, p.gameDetector)
	p.setState(connectedState)

	// Start the pipeline
	pipeline.Start()

	// Send initial telnet negotiation through pipeline
	err := pipeline.SendTelnetNegotiation()
	if err != nil {
		conn.Close()
		panic(fmt.Errorf("telnet negotiation failed: %w", err))
	}

	// Start I/O handlers
	p.inputHandlerStarted = true
	go p.handleInput()
	go p.handleOutput()

	// Load initial script if configured
	if err := p.scriptManager.LoadInitialScript(); err != nil {
		log.Error("Failed to load initial script", "error", err)
	}

	// Load optional script if provided
	if options.ScriptName != "" {
		if err := p.scriptManager.LoadAndRunScript(options.ScriptName); err != nil {
			log.Error("Failed to load optional script", "script", options.ScriptName, "error", err)
		}
	}

	return p
}

func (p *Proxy) Disconnect() error {
	if !p.getState().IsConnected() {
		return nil
	}

	// Transition to disconnected state first - this closes the connection
	// and causes handleOutput() to exit naturally
	p.setState(NewDisconnectedState())

	// Stop all scripts
	if p.scriptManager != nil {
		p.scriptManager.Stop()
	}

	// Close game detector
	if p.gameDetector != nil {
		p.gameDetector.Close()
	}

	// Close database to properly release resources
	if p.db != nil {
		if err := p.db.CloseDatabase(); err != nil {
			log.Info("Error closing database during disconnect", "error", err)
		}
	}

	// Notify TuiAPI about disconnection
	p.tuiAPI.OnConnectionStatusChanged(api.ConnectionStatusDisconnected, "")

	return nil
}

func (p *Proxy) IsConnected() bool {
	return p.getState().IsConnected()
}

func (p *Proxy) SendInput(input string) {
	select {
	case p.inputChan <- input:
	default:
		// Channel full, drop input
	}
}

func (p *Proxy) SendToTUI(output string) {
	err := p.getState().SendToTUI(output)
	if err != nil {
		// Log error but don't fail - maintains current behavior
		log.Info("SendToTUI error", "error", err)
	}
}

func (p *Proxy) SendToServer(input string) {
	state := p.getState()

	if !state.IsConnected() {
		return
	}

	// State handles all processing internally - no nil checks needed
	err := state.SendToServer(input)
	if err != nil {
		p.errorChan <- fmt.Errorf("write error: %w", err)
	}
}

func (p *Proxy) GetOutputChan() <-chan string {
	return p.outputChan
}

func (p *Proxy) GetErrorChan() <-chan error {
	return p.errorChan
}

// GetTerminal method removed - TUI now owns the terminal buffer

func (p *Proxy) handleInput() {
	for input := range p.inputChan {
		state := p.getState()
		connected := state.IsConnected()

		// Check for terminal menu activation - works even when disconnected
		// Process menu key and suppress sending to server if consumed
		if p.terminalMenuManager.ProcessMenuKey(input) {
			// Menu key was processed - don't send to server
			continue
		}

		// Check if terminal menu should handle this input - works even when disconnected
		if p.terminalMenuManager.IsActive() {
			// Menu is active - route input to menu system
			err := p.terminalMenuManager.MenuText(input)
			if err != nil {
				// Menu processing error - log but continue
				p.errorChan <- fmt.Errorf("menu processing error: %w", err)
			}
			// Don't send menu input to server
			continue
		}

		// SCRIPT INPUT SYSTEM: Check if any script is waiting for input - works even when disconnected
		if p.handleScriptInput(input) {
			// Input was consumed by script - don't send to server
			continue
		}

		if !connected {
			continue
		}

		// Process outgoing text through script manager
		p.scriptManager.ProcessOutgoingText(input)

		// Process user input through game detector
		p.gameDetector.ProcessUserInput(input)

		err := state.writeServerData(input)
		if err != nil {
			p.errorChan <- fmt.Errorf("write error: %w", err)
		}
	}
}

func (p *Proxy) handleOutput() {
	// Use a buffer for continuous reading
	buffer := make([]byte, 4096)

	for {
		state := p.getState()
		if !state.IsConnected() {
			break
		}

		// Cast to ConnectedState to access methods directly (avoid repeated state calls)
		connectedState, ok := state.(*ConnectedState)
		if !ok {
			break
		}

		// Read raw bytes from connection
		n, err := connectedState.readServerData(buffer)
		if err != nil {
			// Read error in handleOutput - connection likely closed
			if err.Error() != "EOF" {
				p.errorChan <- fmt.Errorf("read error: %w", err)
			} else {
				// Got EOF, sending to error channel
				p.errorChan <- fmt.Errorf("connection closed: %w", err)
			}
			break
		}

		if n > 0 {
			rawData := buffer[:n]
			// Send raw data directly to the streaming pipeline
			connectedState.processServerData(rawData)
		}
	}

	// If we exit the loop, it means connection was lost
	// handleOutput exiting, setting disconnected state
	p.setState(NewDisconnectedState())
}

// injectInboundData injects data into the inbound stream as if it came from the server
// This is used by the terminal menu system to display menu output
func (p *Proxy) injectInboundData(data []byte) {
	p.getState().processServerData(data)
}

// injectTUIData sends client-side data directly to TUI without server processing
func (p *Proxy) injectTUIData(data []byte) {
	log.Info("injectTUIData called with data", "data", string(data))
	currentState := p.getState()
	if connectedState, ok := currentState.(*ConnectedState); ok {
		if connectedState.pipeline != nil {
			log.Info("injectTUIData: calling pipeline.InjectTUIData")
			connectedState.pipeline.InjectTUIData(data)
		} else {
			log.Info("injectTUIData: no pipeline available")
		}
	} else {
		log.Info("injectTUIData: not in connected state")
	}
}

// GetScriptManager returns the script manager for external access
func (p *Proxy) GetScriptManager() *scripting.ScriptManager {
	return p.scriptManager
}

// SendBurstCommand sends a burst command using the existing terminal menu burst logic
func (p *Proxy) SendBurstCommand(burstText string) error {
	if burstText == "" {
		return errors.New("empty burst command")
	}

	// Use the existing burst processing logic from terminal menu manager
	// Replace * with newlines and send each command
	expandedText := strings.ReplaceAll(burstText, "*", "\r\n")

	// Split into individual commands and send each one directly
	commands := strings.Split(expandedText, "\r\n")
	for _, cmd := range commands {
		if strings.TrimSpace(cmd) != "" {
			p.SendToServer(strings.TrimSpace(cmd) + "\r\n")
		}
	}

	return nil
}

// LoadScript loads a script from file
func (p *Proxy) LoadScript(filename string) error {
	return p.scriptManager.LoadAndRunScript(filename)
}

// ExecuteScriptCommand executes a single script command
func (p *Proxy) ExecuteScriptCommand(command string) error {
	return p.scriptManager.ExecuteCommand(command)
}

// GetScriptStatus returns script engine status
func (p *Proxy) GetScriptStatus() api.ScriptStatusInfo {
	if p.scriptManager == nil {
		return api.ScriptStatusInfo{
			ActiveCount: 0,
			TotalCount:  0,
			ScriptNames: []string{},
		}
	}

	statusMap := p.scriptManager.GetStatus()

	activeCount := 0
	totalCount := 0

	if total, ok := statusMap["total_scripts"].(int); ok {
		totalCount = total
	}
	if running, ok := statusMap["running_scripts"].(int); ok {
		activeCount = running
	}

	scriptNames := []string{}
	if names, ok := statusMap["script_names"].([]string); ok {
		scriptNames = names
	}

	return api.ScriptStatusInfo{
		ActiveCount: activeCount,
		TotalCount:  totalCount,
		ScriptNames: scriptNames,
	}
}

// StopAllScripts stops all running scripts
func (p *Proxy) StopAllScripts() error {
	return p.scriptManager.Stop()
}

// GetDatabase returns the database for API access
func (p *Proxy) GetDatabase() database.Database {
	return p.db
}

// GetParser returns the TWX parser for accessing live game state
func (p *Proxy) GetParser() *streaming.TWXParser {
	return p.getState().GetParser()
}

// GetSector returns sector data using database LoadSector method
func (p *Proxy) GetSector(sectorNum int) (database.TSector, error) {
	return p.db.LoadSector(sectorNum)
}

// GetCurrentSector returns the current sector number from database (like TWX Database.pas)
func (p *Proxy) GetCurrentSector() (int, error) {
	if p.db == nil {
		return 0, fmt.Errorf("database not available")
	}

	playerStats, err := p.db.LoadPlayerStats()
	if err != nil {
		return 0, err
	}

	return playerStats.CurrentSector, nil
}

// SetCurrentSector sets the current sector number and triggers callbacks
func (p *Proxy) SetCurrentSector(sectorNum int) {
	oldSector := p.currentSector
	p.currentSector = sectorNum
	// Check if callback needed
	shouldCallback := oldSector != sectorNum && p.tuiAPI != nil
	currentTuiAPI := p.tuiAPI

	// Trigger callback if sector changed and TuiAPI is available
	if shouldCallback {
		sectorInfo := api.SectorInfo{Number: sectorNum}
		log.Info("PROXY: Firing OnCurrentSectorChanged [SOURCE: SetCurrentSector]", "sector", sectorNum, "old_sector", oldSector)
		go currentTuiAPI.OnCurrentSectorChanged(sectorInfo)
	}
}

// GetPlayerName returns the current player name from database (like TWX Database.pas)
func (p *Proxy) GetPlayerName() string {
	if p.db == nil {
		return ""
	}

	playerStats, err := p.db.LoadPlayerStats()
	if err != nil {
		return ""
	}

	return playerStats.PlayerName
}

// SetPlayerName sets the current player name
func (p *Proxy) SetPlayerName(name string) {
	p.playerName = name
}

// onDatabaseLoaded is called when the game detector loads a database
func (p *Proxy) onDatabaseLoaded(db database.Database, scriptManager *scripting.ScriptManager) error {
	log.Info("onDatabaseLoaded: callback triggered", "db", db)
	// Update proxy state with new database
	p.db = db

	// Update existing script manager with new database instead of replacing it
	if p.scriptManager != nil {
		// Update the script manager's database reference directly
		p.scriptManager.SetDatabase(db)
		p.scriptManager.SetupConnections(p.SendInput, p.SendToTUI, nil)
	}

	// If we have a connected state, update its pipeline without recreating the state
	currentState := p.getState()
	if connectedState, ok := currentState.(*ConnectedState); ok {
		// Stop the old pipeline first
		if connectedState.pipeline != nil {
			connectedState.pipeline.Stop()
		}

		// Create new pipeline with same writer function
		writerFunc := func(data []byte) error {
			_, err := connectedState.writer.Write(data)
			if err != nil {
				return err
			}
			return connectedState.writer.Flush()
		}

		newPipeline := streaming.NewPipeline(p.tuiAPI, func() database.Database { return p.db }, p.scriptManager, p, p.gameDetector, writerFunc)
		connectedState.pipeline = newPipeline
		newPipeline.Start()
	}

	return nil
}

// onDatabaseStateChanged is called when the game detector loads/unloads a database
func (p *Proxy) onDatabaseStateChanged(gameName, serverHost, serverPort, dbName string, isLoaded bool) {

	// Create database state info for TUI notification
	info := api.DatabaseStateInfo{
		GameName:     gameName,
		ServerHost:   serverHost,
		ServerPort:   serverPort,
		DatabaseName: dbName,
		IsLoaded:     isLoaded,
	}

	// Notify TUI about database state change
	if p.tuiAPI != nil {
		p.tuiAPI.OnDatabaseStateChanged(info)
	}
}

// GetCurrentGame returns the currently detected game name
func (p *Proxy) GetCurrentGame() string {
	return p.gameDetector.GetCurrentGame()
}

// IsGameActive returns true if a game is currently active
func (p *Proxy) IsGameActive() bool {
	return p.gameDetector.IsGameActive()
}

// LoadGameDatabase loads a specific game database (legacy method for backward compatibility)
func (p *Proxy) LoadGameDatabase(gameName string) error {
	// This method is now handled by the game detector
	// Keep for backward compatibility but functionality moved to game detector
	return fmt.Errorf("database loading is now handled automatically by game detection")
}

// handleScriptInput handles input for scripts when menu system is not active
// Returns true if input was consumed by a script, false otherwise
func (p *Proxy) handleScriptInput(input string) bool {
	runningScripts := p.scriptManager.GetEngine().GetRunningScripts()

	for _, script := range runningScripts {
		// Check if this script is waiting for input (need to cast to get internal methods)
		if scriptEngine := p.scriptManager.GetEngine(); scriptEngine != nil {
			if internalEngine, ok := scriptEngine.(*scripting.Engine); ok {
				if internalScript, err := internalEngine.GetScript(script.GetID()); err == nil {
					isWaiting := internalScript.VM.IsWaitingForInput()
					if isWaiting {
						// Script is waiting - start input collection if not already collecting
						if !p.scriptInputCollector.IsCollecting() {
							// Start collecting input for this script
							prompt := "SCRIPT_INPUT_" + script.GetID()
							p.scriptInputCollector.RegisterCompletionHandler(prompt, func(menuName, value string) error {
								// Send completed input to script
								err := internalEngine.ResumeScriptWithInput(script.GetID(), value)
								if err != nil {
									p.errorChan <- fmt.Errorf("failed to resume script with input: %w", err)
								}
								return nil
							})
							p.scriptInputCollector.StartCollection(prompt, "")
						}

						// Use shared input collector - handles buffering, echoing, backspace, etc.
						err := p.scriptInputCollector.HandleInput(input)
						if err != nil {
							log.Info("Script input collection error", "error", err)
						}

						// Input was consumed by script
						return true
					}
				}
			}
		}
	}

	// Input was not consumed by any script
	return false
}

// SendData converts byte data to string and sends via SendInput
func (p *Proxy) SendData(data []byte) error {
	p.SendInput(string(data))
	return nil
}

// GetSectorInfo returns information about a specific sector
func (p *Proxy) GetSectorInfo(sectorNum int) (api.SectorInfo, error) {
	if p.db == nil {
		return api.SectorInfo{Number: sectorNum}, errors.New("database not available")
	}

	// Validate sector number range
	if sectorNum < 1 || sectorNum > 99999 {
		return api.SectorInfo{Number: sectorNum}, errors.New("invalid sector number")
	}

	sectorInfo, err := p.db.GetSectorInfo(sectorNum)
	if err != nil {
		return api.SectorInfo{}, err
	}

	return sectorInfo, nil
}

// GetPortInfo returns port information for a specific sector
func (p *Proxy) GetPortInfo(sectorNum int) (*api.PortInfo, error) {
	if p.db == nil {
		return nil, errors.New("database not available")
	}

	// Validate sector number range
	if sectorNum < 1 || sectorNum > 99999 {
		return nil, errors.New("invalid sector number")
	}

	portInfo, err := p.db.GetPortInfo(sectorNum)
	if err != nil {
		return nil, err
	}

	return portInfo, nil
}

// GetPlayerInfo returns the current player information
func (p *Proxy) GetPlayerInfo() (api.PlayerInfo, error) {
	currentSector, err := p.GetCurrentSector()
	if err != nil {
		return api.PlayerInfo{}, err
	}
	playerName := p.GetPlayerName()

	return api.PlayerInfo{
		Name:          playerName,
		CurrentSector: currentSector,
	}, nil
}

// GetPlayerStats returns the current player statistics
func (p *Proxy) GetPlayerStats() (*api.PlayerStatsInfo, error) {
	if p.db == nil {
		return nil, errors.New("database not available")
	}

	apiStats, err := p.db.GetPlayerStatsInfo()
	if err != nil {
		return nil, err
	}

	return &apiStats, nil
}

// GetScriptList returns a list of all scripts with their status
func (p *Proxy) GetScriptList() ([]api.ScriptInfo, error) {
	scriptManager := p.GetScriptManager()
	if scriptManager == nil {
		return []api.ScriptInfo{}, nil
	}

	engine := scriptManager.GetEngine()
	if engine == nil {
		return []api.ScriptInfo{}, nil
	}

	allScripts := engine.GetAllScripts()
	runningScripts := engine.GetRunningScripts()

	runningMap := make(map[string]bool)
	for _, runningScript := range runningScripts {
		runningMap[runningScript.GetID()] = true
	}

	apiScripts := make([]api.ScriptInfo, len(allScripts))
	for i, script := range allScripts {
		apiScripts[i] = api.ScriptInfo{
			ID:       script.GetID(),
			Name:     script.GetName(),
			Filename: script.GetFilename(),
			IsActive: runningMap[script.GetID()],
		}
	}

	return apiScripts, nil
}
