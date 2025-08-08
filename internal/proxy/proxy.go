package proxy

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"

	"twist/internal/proxy/streaming"
	"twist/internal/proxy/database"
	"twist/internal/proxy/scripting"
	"twist/internal/api"
)

type Proxy struct {
	conn     net.Conn
	reader   *bufio.Reader
	writer   *bufio.Writer
	mu       sync.RWMutex
	connected bool
	
	// Channels for communication
	outputChan chan string
	inputChan  chan string
	errorChan  chan error
	
	// Core components
	pipeline      *streaming.Pipeline
	scriptManager *scripting.ScriptManager
	db            database.Database
	
	// Direct TuiAPI reference
	tuiAPI api.TuiAPI
	
	// Game detection
	gameDetector  *GameDetector
	
	// Connection tracking for callbacks
	currentAddress string  // Track address for OnConnectionStatusChanged callbacks
	currentHost    string  // Track hostname for database naming
	currentPort    string  // Track port for database naming
	
	// Game state tracking (Phase 4.3) - based on parser CurrentSectorIndex
	currentSector int    // Track current sector number (from parser)
	playerName    string // Track current player name
}

func New(tuiAPI api.TuiAPI) *Proxy {
	// Initialize with no database initially - will be loaded when game is detected
	var db database.Database = nil
	
	p := &Proxy{
		outputChan:    make(chan string, 100),
		inputChan:     make(chan string, 100),
		errorChan:     make(chan error, 100),
		connected:     false,
		db:            db,
		tuiAPI:        tuiAPI,  // Store TuiAPI reference
		pipeline:      nil,     // Pipeline created only after connection
		gameDetector:  nil,     // Will be created when connection is established
	}
	
	// Create script manager that can request database dynamically
	p.scriptManager = scripting.NewScriptManagerWithProvider(p)
	
	// Setup script manager connections immediately so sendHandler is available
	p.scriptManager.SetupConnections(p, nil)
	
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

func (p *Proxy) Connect(address string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.connected {
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
	
	// Setup database loaded callback
	p.gameDetector.SetDatabaseLoadedCallback(p.onDatabaseLoaded)
	
	// Setup database state change callback
	p.gameDetector.SetDatabaseStateChangedCallback(p.onDatabaseStateChanged)
	

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	p.conn = conn
	p.reader = bufio.NewReader(conn)
	p.writer = bufio.NewWriter(conn)
	p.connected = true

	// NOW create and start the streaming pipeline with a proper writer
	writerFunc := func(data []byte) error {
		_, err := p.writer.Write(data)
		if err != nil {
			return err
		}
		return p.writer.Flush()
	}
	
	// Create pipeline with game detector - database may be nil initially
	p.pipeline = streaming.NewPipelineWithWriter(p.tuiAPI, p.db, p.scriptManager, p, p.gameDetector, writerFunc)
	
	p.pipeline.Start()

	// Send initial telnet negotiation through pipeline
	err = p.pipeline.SendTelnetNegotiation()
	if err != nil {
		conn.Close()
		return fmt.Errorf("telnet negotiation failed: %w", err)
	}

	// Load and run login script automatically on connection
	if p.scriptManager != nil {
		loginScriptPath := "login.ts"
		if err := p.scriptManager.LoadAndRunScript(loginScriptPath); err != nil {
		} else {
		}
	} else {
	}

	// Start goroutines for handling I/O
	go p.handleInput()
	go p.handleOutput()

	return nil
}

func (p *Proxy) Disconnect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.connected {
		return nil
	}

	p.connected = false
	
	// Stop all scripts
	if p.scriptManager != nil {
		p.scriptManager.Stop()
	}
	
	// Stop the streaming pipeline
	if p.pipeline != nil {
		p.pipeline.Stop()
	}
	
	// Close game detector
	if p.gameDetector != nil {
		p.gameDetector.Close()
		p.gameDetector = nil
	}
	
	if p.conn != nil {
		p.conn.Close()
		p.conn = nil
	}

	return nil
}

func (p *Proxy) IsConnected() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.connected
}

func (p *Proxy) SendInput(input string) {
	select {
	case p.inputChan <- input:
	default:
		// Channel full, drop input
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
		p.mu.RLock()
		connected := p.connected && p.writer != nil
		p.mu.RUnlock()

		if !connected {
			continue
		}

		// Process outgoing text through script manager
		if p.scriptManager != nil {
			p.scriptManager.ProcessOutgoingText(input)
		}
		
		// Process user input through game detector
		if p.gameDetector != nil {
			p.gameDetector.ProcessUserInput(input)
		}

		_, err := p.writer.WriteString(input)
		if err != nil {
			p.errorChan <- fmt.Errorf("write error: %w", err)
			continue
		}

		err = p.writer.Flush()
		if err != nil {
			p.errorChan <- fmt.Errorf("flush error: %w", err)
		}
	}
}

func (p *Proxy) handleOutput() {
	// Use a buffer for continuous reading
	buffer := make([]byte, 4096)
	
	for {
		p.mu.RLock()
		connected := p.connected
		p.mu.RUnlock()

		if !connected {
			break
		}

		// Read raw bytes from connection
		n, err := p.reader.Read(buffer)
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
			p.pipeline.Write(rawData)
		}
	}
	
	// If we exit the loop, it means connection was lost
	// handleOutput exiting, setting connected=false
	p.mu.Lock()
	p.connected = false
	p.mu.Unlock()
}

// GetScriptManager returns the script manager for external access
func (p *Proxy) GetScriptManager() *scripting.ScriptManager {
	return p.scriptManager
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
	if p.pipeline == nil {
		return nil
	}
	return p.pipeline.GetParser()
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
	p.mu.Lock()
	oldSector := p.currentSector
	p.currentSector = sectorNum
	// Keep lock during callback check to prevent race conditions
	shouldCallback := oldSector != sectorNum && p.tuiAPI != nil
	currentTuiAPI := p.tuiAPI // Capture reference while locked
	p.mu.Unlock()
	
	// Trigger callback if sector changed and TuiAPI is available
	if shouldCallback {
		sectorInfo := api.SectorInfo{Number: sectorNum}
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
	p.mu.Lock()
	defer p.mu.Unlock()
	p.playerName = name
}

// onDatabaseLoaded is called when the game detector loads a database
func (p *Proxy) onDatabaseLoaded(db database.Database, scriptManager *scripting.ScriptManager) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Update proxy state with new database
	p.db = db
	
	// Update existing script manager with new database instead of replacing it
	if p.scriptManager != nil {
		p.scriptManager.UpdateDatabase()
		p.scriptManager.SetupConnections(p, nil)
	}
	
	// If pipeline exists, update it with the new database
	if p.pipeline != nil {
		// Recreate pipeline with new database
		writerFunc := func(data []byte) error {
			_, err := p.writer.Write(data)
			if err != nil {
				return err
			}
			return p.writer.Flush()
		}
		
		p.pipeline.Stop()
		p.pipeline = streaming.NewPipelineWithWriter(p.tuiAPI, p.db, p.scriptManager, p, p.gameDetector, writerFunc)
		p.pipeline.Start()
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
	} else {
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

