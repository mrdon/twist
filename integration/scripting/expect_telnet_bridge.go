package scripting

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
	"twist/internal/api"
	"twist/internal/proxy"
	"twist/internal/proxy/database"
)

// ExpectTelnetBridge connects our expect engine to a real proxy via telnet server
type ExpectTelnetBridge struct {
	t            *testing.T
	proxy        *proxy.Proxy
	telnetServer *ExpectTelnetServer  // Use expect-enabled server
	expectEngine *SimpleExpectEngine
	database     database.Database
	databasePath string  // Store the database file path for tests
	
	// Channels for communication
	dataToServer   chan string    // Data expect engine wants to send to proxy (as if from game server)
	dataFromClient chan string    // Data received from proxy (user input)
	serverOutput   chan string    // Data sent to proxy (server responses)
	
	// State
	connected      bool
	cleanupFuncs   []func()
	mu             sync.Mutex
}

// NewExpectTelnetBridge creates a new bridge between expect engine and real proxy
func NewExpectTelnetBridge(t *testing.T) *ExpectTelnetBridge {
	bridge := &ExpectTelnetBridge{
		t:              t,
		dataToServer:   make(chan string, 100),
		dataFromClient: make(chan string, 100),
		serverOutput:   make(chan string, 100),
		cleanupFuncs:   make([]func(), 0),
	}
	
	// Register cleanup
	t.Cleanup(bridge.Cleanup)
	
	return bridge
}

// SetupDatabase creates a real database for testing
func (b *ExpectTelnetBridge) SetupDatabase() *ExpectTelnetBridge {
	b.databasePath = b.t.TempDir() + "/test.db"
	
	// Create real database
	b.database = database.NewDatabase()
	err := b.database.CreateDatabase(b.databasePath)
	if err != nil {
		b.t.Fatalf("Failed to create test database: %v", err)
	}
	
	// Close and reopen for clean state
	b.database.CloseDatabase()
	b.database = database.NewDatabase()
	err = b.database.OpenDatabase(b.databasePath)
	if err != nil {
		b.t.Fatalf("Failed to open test database: %v", err)
	}
	
	b.cleanupFuncs = append(b.cleanupFuncs, func() {
		if b.database != nil {
			b.database.CloseDatabase()
		}
	})
	
	return b
}

// SetupTelnetServer creates and starts the telnet server
func (b *ExpectTelnetBridge) SetupTelnetServer() *ExpectTelnetBridge {
	b.telnetServer = NewExpectTelnetServer(b.t)
	
	port, err := b.telnetServer.Start()
	if err != nil {
		b.t.Fatalf("Failed to start telnet server: %v", err)
	}
	
	b.t.Logf("Test telnet server started on port %d", port)
	
	// Set up server to handle expect engine commands
	go b.handleServerCommands()
	
	b.cleanupFuncs = append(b.cleanupFuncs, func() {
		if b.telnetServer != nil {
			b.telnetServer.Stop()
		}
	})
	
	return b
}

// SetupProxy creates a real proxy and connects it to our telnet server
func (b *ExpectTelnetBridge) SetupProxy() *ExpectTelnetBridge {
	if b.telnetServer == nil {
		b.t.Fatal("Must call SetupTelnetServer() before SetupProxy()")
	}
	
	// Create a TuiAPI that captures output for expect engine
	tuiAPI := &ExpectTuiAPI{bridge: b}
	
	// Create real proxy with real TuiAPI and our test database
	b.proxy = proxy.NewWithDatabase(tuiAPI, b.database)
	
	// Connect to our test telnet server
	address := fmt.Sprintf("localhost:%d", b.telnetServer.port)
	err := b.proxy.Connect(address)
	if err != nil {
		b.t.Fatalf("Failed to connect proxy to telnet server: %v", err)
	}
	
	b.connected = true
	b.t.Logf("Proxy connected to test server at %s", address)
	
	b.cleanupFuncs = append(b.cleanupFuncs, func() {
		if b.proxy != nil {
			b.proxy.Disconnect()
		}
	})
	
	return b
}

// SetupExpectEngine creates the expect engine
func (b *ExpectTelnetBridge) SetupExpectEngine() *ExpectTelnetBridge {
	if b.proxy == nil {
		b.t.Fatal("Must call SetupProxy() before SetupExpectEngine()")
	}
	
	// Create expect engine that can send data through proxy
	// Client sends "\r" for "*" since it's simulating user input
	b.expectEngine = NewExpectEngine(b.t, func(input string) {
		b.t.Logf("EXPECT ENGINE SENDING INPUT: %q", input)
		b.proxy.SendInput(input)
	}, "\r")
	
	return b
}

// handleServerCommands processes commands from expect engine to control server responses
func (b *ExpectTelnetBridge) handleServerCommands() {
	// For now, we'll set initial responses that the telnet server will send
	// This is simpler than trying to dynamically send data
	initialResponses := []string{
		"Trade Wars 2002 - The Game\r\n",
		"Enter your login name: ",
	}
	
	if b.telnetServer != nil {
		b.telnetServer.SetResponses(initialResponses)
	}
}

// SetServerScript sets the server-side expect script with automatic sync token
func (b *ExpectTelnetBridge) SetServerScript(script string) *ExpectTelnetBridge {
	if b.telnetServer != nil {
		// Automatically append sync token to server script
		enhancedScript := script + `
# Automatic sync token
send "___SYNC_END___\r\n"
log "Server: Sync token sent, data processing complete"
`
		b.telnetServer.SetServerScript(enhancedScript)
	}
	return b
}

// WaitForServerScript waits for server script to complete
func (b *ExpectTelnetBridge) WaitForServerScript(timeout time.Duration) error {
	if b.telnetServer != nil {
		return b.telnetServer.WaitForServerScript(timeout)
	}
	return nil
}

// SendServerData sends data to proxy as if it came from game server
func (b *ExpectTelnetBridge) SendServerData(data string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.t.Logf("SENDING SERVER DATA: %q", data)
	
	// Use dynamic data sending to send data immediately
	if b.telnetServer != nil {
		b.telnetServer.SendDynamicData(data)
	}
}

// RunExpectScript executes an expect script against the real proxy system
func (b *ExpectTelnetBridge) RunExpectScript(script string) error {
	if b.expectEngine == nil {
		return fmt.Errorf("expect engine not set up - call SetupExpectEngine()")
	}
	
	b.t.Logf("RUNNING EXPECT SCRIPT:\n%s", script)
	
	// Execute the expect script
	return b.expectEngine.Run(script)
}

// RunSyncedScripts runs server and client scripts with automatic synchronization and returns opened database
func (b *ExpectTelnetBridge) RunSyncedScripts(serverScript, clientScript string) (*sql.DB, error) {
	if b.expectEngine == nil {
		return nil, fmt.Errorf("expect engine not set up - call SetupExpectEngine()")
	}
	
	// Set server script (will automatically add sync token)
	b.SetServerScript(serverScript)
	
	// Enhance client script with automatic sync token wait
	enhancedClientScript := clientScript + `
# Automatic sync wait
expect "___SYNC_END___"
log "Client: Received sync token, data processing complete"
`
	
	// Run client script
	err := b.RunExpectScript(enhancedClientScript)
	if err != nil {
		return nil, fmt.Errorf("client expect script failed: %w", err)
	}
	
	// Wait for server script to complete
	err = b.WaitForServerScript(5 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("server script failed: %w", err)
	}
	
	// Open and return database for test verification
	db, err := sql.Open("sqlite3", b.databasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database for verification: %w", err)
	}
	
	return db, nil
}

// LoadScript loads and runs a TWX script through the proxy
func (b *ExpectTelnetBridge) LoadScript(scriptContent string) *ExpectTelnetBridge {
	if b.proxy == nil {
		b.t.Fatal("Must call SetupProxy() before LoadScript()")
	}
	
	// Create temporary script file
	scriptPath := b.t.TempDir() + "/test_script.ts"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0644)
	if err != nil {
		b.t.Fatalf("Failed to create script file: %v", err)
	}
	
	// Load script through proxy
	err = b.proxy.LoadScript(scriptPath)
	if err != nil {
		b.t.Fatalf("Failed to load script: %v", err)
	}
	
	b.t.Logf("Script loaded: %s", scriptPath)
	return b
}

// GetDatabase returns the database instance for testing
func (b *ExpectTelnetBridge) GetDatabase() database.Database {
	return b.database
}

// GetDatabasePath returns the database file path for direct SQL access
func (b *ExpectTelnetBridge) GetDatabasePath() string {
	return b.databasePath
}

// Cleanup cleans up all resources
func (b *ExpectTelnetBridge) Cleanup() {
	for i := len(b.cleanupFuncs) - 1; i >= 0; i-- {
		b.cleanupFuncs[i]()
	}
}

// ExpectTuiAPI captures proxy output and feeds it to expect engine
type ExpectTuiAPI struct {
	bridge *ExpectTelnetBridge
}

func (e *ExpectTuiAPI) OnConnectionStatusChanged(status api.ConnectionStatus, address string) {
	e.bridge.t.Logf("Connection status: %s (%s)", status, address)
}

func (e *ExpectTuiAPI) OnConnectionError(err error) {
	e.bridge.t.Logf("Connection error: %v", err)
}

func (e *ExpectTuiAPI) OnData(data []byte) {
	// This is the key connection - proxy output goes to expect engine
	output := string(data)
	e.bridge.t.Logf("PROXY OUTPUT -> EXPECT ENGINE: %q", output)
	
	if e.bridge.expectEngine != nil {
		e.bridge.expectEngine.AddOutput(output)
	}
}

func (e *ExpectTuiAPI) OnScriptStatusChanged(status api.ScriptStatusInfo) {
	e.bridge.t.Logf("Script status: %+v", status)
}

func (e *ExpectTuiAPI) OnScriptError(scriptName string, err error) {
	e.bridge.t.Logf("Script error in %s: %v", scriptName, err)
}

func (e *ExpectTuiAPI) OnDatabaseStateChanged(info api.DatabaseStateInfo) {
	e.bridge.t.Logf("Database state: %+v", info)
}

func (e *ExpectTuiAPI) OnCurrentSectorChanged(sectorInfo api.SectorInfo) {
	e.bridge.t.Logf("Sector changed: %+v", sectorInfo)
}

func (e *ExpectTuiAPI) OnTraderDataUpdated(sectorNumber int, traders []api.TraderInfo) {
	e.bridge.t.Logf("Traders updated in sector %d", sectorNumber)
}

func (e *ExpectTuiAPI) OnPlayerStatsUpdated(stats api.PlayerStatsInfo) {
	e.bridge.t.Logf("Player stats updated: %+v", stats)
}