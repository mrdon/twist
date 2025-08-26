package proxy

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	"twist/internal/api"
	"twist/internal/debug"
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

// proxyAdapter adapts the Proxy to work with menu.ProxyInterface
type proxyAdapter struct {
	proxy *Proxy
}

func (pa *proxyAdapter) GetScriptManager() menu.ScriptManagerInterface {
	return pa.proxy.scriptManager
}

func (pa *proxyAdapter) GetDatabase() interface{} {
	return pa.proxy.db
}

func (pa *proxyAdapter) SendInput(input string) {
	pa.proxy.SendInput(input)
}

func (pa *proxyAdapter) SendDirectToServer(input string) {
	pa.proxy.SendToServer(input)
}

func (pa *proxyAdapter) SendOutput(output string) {
	pa.proxy.SendToTUI(output)
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

	// Input buffering for script input collection
	scriptInputBuffer     strings.Builder
	scriptWaitingForInput bool
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

func New(tuiAPI api.TuiAPI) *Proxy {
	// Initialize with no database initially - will be loaded when game is detected
	var db database.Database = nil

	p := &Proxy{
		outputChan:   make(chan string, 100),
		inputChan:    make(chan string, 100),
		errorChan:    make(chan error, 100),
		db:           db,
		tuiAPI:       tuiAPI, // Store TuiAPI reference
		gameDetector: nil,    // Will be created when connection is established
	}

	// Initialize with disconnected state
	p.state = NewDisconnectedState()

	// Initialize terminal menu manager
	p.terminalMenuManager = menu.NewTerminalMenuManager()

	// Set up menu data injection function
	p.terminalMenuManager.SetInjectDataFunc(p.injectInboundData)

	// Set up proxy interface for menu system
	p.terminalMenuManager.SetProxyInterface(&proxyAdapter{p})

	// Initialize script input collector - reuses same logic as menu input
	p.scriptInputCollector = input.NewInputCollector(func(output string) {
		// Echo script input to screen via TuiAPI
		if p.tuiAPI != nil {
			p.tuiAPI.OnData([]byte(output))
		}
	})

	// Create script manager with direct database access
	p.scriptManager = scripting.NewScriptManager(p.db)

	// Setup script manager connections immediately so sendHandler is available
	adapter := &proxyAdapter{p}
	p.scriptManager.SetupConnections(adapter, nil)

	// Setup menu manager for script menu commands
	p.scriptManager.SetupMenuManager(p.terminalMenuManager)

	// Start input handler immediately so menu system works even when not connected
	p.inputHandlerStarted = true
	go p.handleInput()

	return p
}

// NewWithDatabase creates a new proxy with a pre-configured database (for testing)
func NewWithDatabase(tuiAPI api.TuiAPI, db database.Database) *Proxy {
	p := &Proxy{
		outputChan:   make(chan string, 100),
		inputChan:    make(chan string, 100),
		errorChan:    make(chan error, 100),
		db:           db, // Use the provided database
		tuiAPI:       tuiAPI,
		gameDetector: nil,
	}

	// Initialize with disconnected state
	p.state = NewDisconnectedState()

	// Initialize terminal menu manager
	p.terminalMenuManager = menu.NewTerminalMenuManager()

	// Set up menu data injection function
	p.terminalMenuManager.SetInjectDataFunc(p.injectInboundData)

	// Set up proxy interface for menu system
	p.terminalMenuManager.SetProxyInterface(&proxyAdapter{p})

	// Initialize script input collector
	p.scriptInputCollector = input.NewInputCollector(func(output string) {
		if p.tuiAPI != nil {
			p.tuiAPI.OnData([]byte(output))
		}
	})

	// Create script manager with direct database access
	p.scriptManager = scripting.NewScriptManager(p.db)

	// Setup script manager connections immediately
	adapter := &proxyAdapter{p}
	p.scriptManager.SetupConnections(adapter, nil)

	// Setup menu manager for script menu commands
	p.scriptManager.SetupMenuManager(p.terminalMenuManager)

	// Start input handler immediately
	p.inputHandlerStarted = true
	go p.handleInput()

	return p
}

// escapeANSI converts ANSI escape sequences to readable text
func escapeANSI(data []byte) string {
	str := string(data)
	// Replace escape character with \x1b for readability
	str = strings.ReplaceAll(str, "\x1b", "\\x1b")
	// Replace other common control characters
	str = strings.ReplaceAll(str, "\r", "\\r")
	str = strings.ReplaceAll(str, "\n", "\\n")
	str = strings.ReplaceAll(str, "\t", "\\t")
	return str
}

// extractContext returns 10 chars before and after the target string
func extractContext(data []byte, target string) string {
	str := string(data)
	index := strings.Index(str, target)
	if index == -1 {
		return ""
	}

	start := index - 10
	if start < 0 {
		start = 0
	}

	end := index + len(target) + 10
	if end > len(str) {
		end = len(str)
	}

	context := str[start:end]
	return escapeANSI([]byte(context))
}

func (p *Proxy) Connect(address string, options ...*api.ConnectOptions) error {

	// Parse options
	var opts *api.ConnectOptions
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	} else {
		opts = &api.ConnectOptions{}
	}
	if p.getState().IsConnected() {
		return fmt.Errorf("already connected")
	}

	// Parse address (default to telnet port if not specified)
	if !strings.Contains(address, ":") {
		address = address + ":23"
	}
	// Store connection details for database naming
	p.currentAddress = address
	parts := strings.Split(address, ":")
	if len(parts) >= 2 {
		p.currentHost = parts[0]
		p.currentPort = parts[1]
	} else {
		p.currentHost = address
		p.currentPort = "23"
	}
	// Create game detector with connection info
	connInfo := ConnectionInfo{Host: p.currentHost, Port: p.currentPort}
	p.gameDetector = NewGameDetector(connInfo)

	// If database path is provided, force load that database instead of auto-detection
	debug.Log("Initializing database: DatabasePath=%q", opts.DatabasePath)
	if opts.DatabasePath != "" {
		debug.Log("Using forced database path: %s", opts.DatabasePath)
		db := database.NewDatabase()
		if err := db.CreateDatabase(opts.DatabasePath); err != nil {
			if err := db.OpenDatabase(opts.DatabasePath); err != nil {
				return fmt.Errorf("failed to load forced database %s: %w", opts.DatabasePath, err)
			}
		}

		// Set up database and script manager directly
		p.db = db
		p.scriptManager = scripting.NewScriptManager(db)

		// Skip game detector's database loading by not setting callbacks
	} else {
		// Setup database loaded callback for normal auto-detection
		debug.Log("Setting up database loaded callback for auto-detection")
		p.gameDetector.SetDatabaseLoadedCallback(p.onDatabaseLoaded)
	}

	// Setup database state change callback
	p.gameDetector.SetDatabaseStateChangedCallback(p.onDatabaseStateChanged)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	writerFunc := func(data []byte) error {
		_, err := writer.Write(data)
		if err != nil {
			return err
		}
		return writer.Flush()
	}

	// Create connected state first, before starting pipeline
	connectedState := NewConnectedState(conn, reader, writer, nil, p.scriptManager, p.gameDetector)
	p.setState(connectedState)

	// Now create and start pipeline with state already set
	pipeline := streaming.NewPipelineWithWriter(p.tuiAPI, p.db, p.scriptManager, p, p.gameDetector, writerFunc)
	connectedState.pipeline = pipeline
	pipeline.Start()

	// Send initial telnet negotiation through pipeline
	err = pipeline.SendTelnetNegotiation()
	if err != nil {
		conn.Close()
		return fmt.Errorf("telnet negotiation failed: %w", err)
	}

	// Load and run initial script automatically on connection (if configured)
	if p.scriptManager != nil {
		if err := p.scriptManager.LoadInitialScript(); err != nil {
			// Script loading error - log but don't fail connection
		}

		// Load optional script if provided
		if opts.ScriptName != "" {
			if err := p.scriptManager.LoadAndRunScript(opts.ScriptName); err != nil {
				// Script loading error - log but don't fail connection
			}
		}
	}

	// Start goroutines for handling I/O
	// Since we already start handleInput in New(), it should already be running
	// No need for complex locking - just start handleOutput
	if !p.inputHandlerStarted {
		go p.handleInput()
		p.inputHandlerStarted = true
	}

	go p.handleOutput()
	return nil
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
		p.gameDetector = nil
	}

	// Close database to properly release resources
	if p.db != nil {
		if err := p.db.CloseDatabase(); err != nil {
			debug.Log("Error closing database during disconnect: %v", err)
		}
	}

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
		debug.Log("SendToTUI error: %v", err)
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
		if p.terminalMenuManager != nil {
			// Process menu key and suppress sending to server if consumed
			if p.terminalMenuManager.ProcessMenuKey(input) {
				// Menu key was processed - don't send to server
				continue
			}
		}

		// Check if terminal menu should handle this input - works even when disconnected
		if p.terminalMenuManager != nil && p.terminalMenuManager.IsActive() {
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
		if p.scriptManager != nil {
			if p.handleScriptInput(input) {
				// Input was consumed by script - don't send to server
				continue
			}
		}

		if !connected {
			continue
		}

		// OLD SCRIPT INPUT CODE MOVED TO handleScriptInput method
		// Check if any script is waiting for input - if so, buffer input until complete line
		if false && p.scriptManager != nil { // DISABLED - moved to handleScriptInput
			runningScripts := p.scriptManager.GetEngine().GetRunningScripts()
			debug.Log("PROXY INPUT: checking %d running scripts for input waiting", len(runningScripts))
			for _, script := range runningScripts {
				debug.Log("PROXY INPUT: checking script %s", script.GetID())
				// Check if this script is waiting for input (need to cast to get internal methods)
				if scriptEngine := p.scriptManager.GetEngine(); scriptEngine != nil {
					if internalEngine, ok := scriptEngine.(*scripting.Engine); ok {
						if internalScript, err := internalEngine.GetScript(script.GetID()); err == nil {
							isWaiting := internalScript.VM.IsWaitingForInput()
							debug.Log("PROXY INPUT: script %s IsWaitingForInput=%v", script.GetID(), isWaiting)
							if isWaiting {
								// Script is waiting for input - start buffering
								debug.Log("SCRIPT INPUT DEBUG: received input %q (len=%d), current buffer: %q", input, len(input), p.scriptInputBuffer.String())

								// Check if this is ENTER (carriage return, newline, or CRLF)
								isEnterPressed := (input == "\r" || input == "\n" || input == "\r\n")

								if isEnterPressed {
									// ENTER pressed - use any buffered input (the current input is just the ENTER key)
									finalInput := p.scriptInputBuffer.String()

									// Clear buffer and send input to script
									p.scriptInputBuffer.Reset()
									debug.Log("SCRIPT INPUT DEBUG: sending final input %q to script %s", finalInput, script.GetID())

									err := internalEngine.ResumeScriptWithInput(script.GetID(), finalInput)
									if err != nil {
										p.errorChan <- fmt.Errorf("failed to resume script with input: %w", err)
									}
								} else {
									// Regular character(s) - add to buffer
									p.scriptInputBuffer.WriteString(input)
								}

								// Input was consumed by script - don't send to server
								continue
							}
						}
					}
				}
			}

			// If no scripts are waiting for input, clear any stale buffer
			stillWaitingForInput := false
			for _, script := range runningScripts {
				if scriptEngine := p.scriptManager.GetEngine(); scriptEngine != nil {
					if internalEngine, ok := scriptEngine.(*scripting.Engine); ok {
						if internalScript, err := internalEngine.GetScript(script.GetID()); err == nil {
							if internalScript.VM.IsWaitingForInput() {
								stillWaitingForInput = true
								break
							}
						}
					}
				}
			}

			if !stillWaitingForInput {
				p.scriptInputBuffer.Reset()
			}
		}

		// Process outgoing text through script manager
		if p.scriptManager != nil {
			p.scriptManager.ProcessOutgoingText(input)
		}

		// Process user input through game detector
		if p.gameDetector != nil {
			p.gameDetector.ProcessUserInput(input)
		}

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

// GetScriptManager returns the script manager for external access
func (p *Proxy) GetScriptManager() *scripting.ScriptManager {
	return p.scriptManager
}

// SendBurstCommand sends a burst command using the existing terminal menu burst logic
func (p *Proxy) SendBurstCommand(burstText string) error {
	if burstText == "" {
		return errors.New("empty burst command")
	}

	if p.terminalMenuManager == nil {
		return errors.New("terminal menu manager not available")
	}

	// Use the existing burst processing logic from terminal menu manager
	// Replace * with newlines and send each command
	expandedText := strings.ReplaceAll(burstText, "*", "\r\n")

	// Split into individual commands and send each one using the proxy adapter pattern
	commands := strings.Split(expandedText, "\r\n")
	proxyAdapter := &proxyAdapter{p}
	for _, cmd := range commands {
		if strings.TrimSpace(cmd) != "" {
			// Use the same method that the terminal menu manager uses
			proxyAdapter.SendDirectToServer(strings.TrimSpace(cmd) + "\r\n")
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
func (p *Proxy) GetScriptStatus() map[string]interface{} {
	return p.scriptManager.GetStatus()
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
func (p *Proxy) GetCurrentSector() int {
	if p.db == nil {
		return 0
	}

	playerStats, err := p.db.LoadPlayerStats()
	if err != nil {
		return 0
	}

	return playerStats.CurrentSector
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
		debug.Log("PROXY: Firing OnCurrentSectorChanged for sector %d (oldSector=%d) [SOURCE: SetCurrentSector]", sectorNum, oldSector)
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
	debug.Log("onDatabaseLoaded: callback triggered with db=%v", db)
	// Update proxy state with new database
	p.db = db

	// Update existing script manager with new database instead of replacing it
	if p.scriptManager != nil {
		// Update the script manager's database reference directly
		p.scriptManager.SetDatabase(db)
		adapter := &proxyAdapter{p}
		p.scriptManager.SetupConnections(adapter, nil)
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

		newPipeline := streaming.NewPipelineWithWriter(p.tuiAPI, p.db, p.scriptManager, p, p.gameDetector, writerFunc)
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
	if p.gameDetector != nil {
		return p.gameDetector.GetCurrentGame()
	}
	return ""
}

// IsGameActive returns true if a game is currently active
func (p *Proxy) IsGameActive() bool {
	if p.gameDetector != nil {
		return p.gameDetector.IsGameActive()
	}
	return false
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
							debug.Log("Script input collection error: %v", err)
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
