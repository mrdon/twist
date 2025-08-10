package scripting

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"twist/internal/api"
	"twist/internal/proxy"
	"twist/internal/proxy/database"
)

// ScriptTestFramework provides a reusable testing infrastructure for TWX scripts
// Similar to pytest fixtures, but for TWX script integration testing
type ScriptTestFramework struct {
	t                *testing.T
	telnetServer     *TestTelnetServer
	proxy            *proxy.Proxy
	mockTuiAPI       *MockTuiAPI
	database         database.Database
	tempDir          string
	scriptFiles      []string
	cleanupFunctions []func()
}

// NewScriptTestFramework creates a new script testing framework
func NewScriptTestFramework(t *testing.T) *ScriptTestFramework {
	framework := &ScriptTestFramework{
		t:                t,
		scriptFiles:      make([]string, 0),
		cleanupFunctions: make([]func(), 0),
		tempDir:          t.TempDir(),
	}

	// Register cleanup
	t.Cleanup(framework.Cleanup)

	return framework
}

// SetupDatabase creates and initializes a fresh database for testing
func (f *ScriptTestFramework) SetupDatabase() *ScriptTestFramework {
	dbPath := filepath.Join(f.tempDir, "test.db")
	
	// Clean up any existing database file
	if _, err := os.Stat(dbPath); err == nil {
		os.Remove(dbPath)
	}
	
	// Create a completely new database instance
	f.database = database.NewDatabase()
	
	// Create database file
	err := f.database.CreateDatabase(dbPath)
	if err != nil {
		f.t.Fatalf("Failed to create test database: %v", err)
	}
	
	// Close after creation, then reopen - this ensures clean state
	f.database.CloseDatabase()
	
	// Create fresh instance and open
	f.database = database.NewDatabase()
	err = f.database.OpenDatabase(dbPath)
	if err != nil {
		f.t.Fatalf("Failed to open test database: %v", err)
	}
	
	f.cleanupFunctions = append(f.cleanupFunctions, func() {
		if f.database != nil {
			f.database.CloseDatabase()
		}
	})
	
	return f
}

// SetupTelnetServer creates and starts a telnet server for testing
func (f *ScriptTestFramework) SetupTelnetServer() *ScriptTestFramework {
	f.telnetServer = NewTestTelnetServer(f.t)
	port, err := f.telnetServer.Start()
	if err != nil {
		f.t.Fatalf("Failed to start telnet server: %v", err)
	}
	
	f.t.Logf("Telnet server started on port %d", port)
	
	f.cleanupFunctions = append(f.cleanupFunctions, func() {
		if f.telnetServer != nil {
			f.telnetServer.Stop()
		}
	})
	
	return f
}

// SetupProxy creates and configures the proxy with mock TUI
func (f *ScriptTestFramework) SetupProxy() *ScriptTestFramework {
	f.mockTuiAPI = NewMockTuiAPI(f.t)
	f.proxy = proxy.New(f.mockTuiAPI)
	
	f.cleanupFunctions = append(f.cleanupFunctions, func() {
		if f.proxy != nil {
			if err := f.proxy.Disconnect(); err != nil {
				f.t.Logf("Disconnect error: %v", err)
			}
		}
	})
	
	return f
}

// ConnectToTelnetServer connects the proxy to the telnet server
func (f *ScriptTestFramework) ConnectToTelnetServer() *ScriptTestFramework {
	if f.telnetServer == nil {
		f.t.Fatal("Telnet server not set up - call SetupTelnetServer() first")
	}
	if f.proxy == nil {
		f.t.Fatal("Proxy not set up - call SetupProxy() first")
	}
	
	address := f.telnetServer.GetAddress()
	err := f.proxy.Connect(address)
	if err != nil {
		f.t.Fatalf("Failed to connect proxy to telnet server: %v", err)
	}
	
	// Wait for connection to stabilize
	time.Sleep(50 * time.Millisecond)
	
	return f
}

// CreateScript creates a temporary script file with the given content
func (f *ScriptTestFramework) CreateScript(name, content string) string {
	scriptPath := filepath.Join(f.tempDir, name)
	err := os.WriteFile(scriptPath, []byte(content), 0644)
	if err != nil {
		f.t.Fatalf("Failed to create script file %s: %v", scriptPath, err)
	}
	
	f.scriptFiles = append(f.scriptFiles, scriptPath)
	return scriptPath
}

// LoadAndRunScript loads and executes a script
func (f *ScriptTestFramework) LoadAndRunScript(scriptPath string) *ScriptTestFramework {
	if f.proxy == nil {
		f.t.Fatal("Proxy not set up - call SetupProxy() first")
	}
	
	err := f.proxy.LoadScript(scriptPath)
	if err != nil {
		f.t.Fatalf("Failed to load script %s: %v", scriptPath, err)
	}
	
	// Wait for script to start
	time.Sleep(100 * time.Millisecond)
	
	return f
}

// ConfigureTelnetResponses sets up the telnet server responses
func (f *ScriptTestFramework) ConfigureTelnetResponses(responses []string) *ScriptTestFramework {
	if f.telnetServer == nil {
		f.t.Fatal("Telnet server not set up - call SetupTelnetServer() first")
	}
	
	f.telnetServer.SetResponses(responses)
	return f
}

// SendUserInput sends input to the proxy (simulating user typing)
func (f *ScriptTestFramework) SendUserInput(input string) *ScriptTestFramework {
	if f.proxy == nil {
		f.t.Fatal("Proxy not set up - call SetupProxy() first")
	}
	
	f.proxy.SendInput(input + "\r\n")
	time.Sleep(50 * time.Millisecond) // Wait for processing
	
	return f
}

// SendMultipleInputs sends multiple user inputs in sequence
func (f *ScriptTestFramework) SendMultipleInputs(inputs []string) *ScriptTestFramework {
	for _, input := range inputs {
		f.SendUserInput(input)
	}
	return f
}

// WaitForScriptCompletion waits for the script to finish processing
func (f *ScriptTestFramework) WaitForScriptCompletion() *ScriptTestFramework {
	time.Sleep(200 * time.Millisecond)
	return f
}

// GetTelnetInputs returns all inputs received by the telnet server
func (f *ScriptTestFramework) GetTelnetInputs() []string {
	if f.telnetServer == nil {
		return []string{}
	}
	return f.telnetServer.GetInputs()
}

// GetReceivedData returns all data received by the mock TUI
func (f *ScriptTestFramework) GetReceivedData() [][]byte {
	if f.mockTuiAPI == nil {
		return [][]byte{}
	}
	return f.mockTuiAPI.GetReceivedData()
}

// AssertInputCount verifies the expected number of inputs were sent to telnet server
func (f *ScriptTestFramework) AssertInputCount(expectedCount int) *ScriptTestFramework {
	inputs := f.GetTelnetInputs()
	actualCount := len(inputs)
	
	if actualCount != expectedCount {
		f.t.Errorf("Expected %d inputs sent to telnet server, got %d", expectedCount, actualCount)
		f.t.Errorf("Inputs received: %v", inputs)
	}
	
	return f
}

// AssertInputsContain verifies that the telnet server received inputs containing expected values
func (f *ScriptTestFramework) AssertInputsContain(expectedValues []string) *ScriptTestFramework {
	inputs := f.GetTelnetInputs()
	
	for i, expected := range expectedValues {
		if i >= len(inputs) {
			f.t.Errorf("Missing input %d (expected to contain %q)", i+1, expected)
			continue
		}
		
		if !strings.Contains(inputs[i], expected) {
			f.t.Errorf("Input %d should contain %q, got %q", i+1, expected, inputs[i])
		}
	}
	
	return f
}

// AssertMinimumInputCount ensures at least the minimum number of inputs were received
func (f *ScriptTestFramework) AssertMinimumInputCount(minCount int) *ScriptTestFramework {
	inputs := f.GetTelnetInputs()
	actualCount := len(inputs)
	
	if actualCount < minCount {
		f.t.Errorf("Expected at least %d inputs sent to telnet server, got %d", minCount, actualCount)
		f.t.Errorf("This indicates script may have stopped early. Inputs received: %v", inputs)
	} else {
		f.t.Logf("SUCCESS: Script processed %d inputs (expected minimum %d)", actualCount, minCount)
	}
	
	return f
}

// Cleanup cleans up all resources
func (f *ScriptTestFramework) Cleanup() {
	for i := len(f.cleanupFunctions) - 1; i >= 0; i-- {
		f.cleanupFunctions[i]()
	}
	
	// Clean up script files
	for _, scriptPath := range f.scriptFiles {
		os.Remove(scriptPath)
	}
}

// TestTelnetServer is a telnet server specifically designed for testing
type TestTelnetServer struct {
	listener     net.Listener
	connections  []net.Conn
	responses    []string
	inputs       []string
	currentStep  int
	mutex        sync.Mutex
	t            *testing.T
	port         int
	
	// Dynamic data sending
	dynamicData  chan string
}

// NewTestTelnetServer creates a new telnet server for testing
func NewTestTelnetServer(t *testing.T) *TestTelnetServer {
	return &TestTelnetServer{
		connections: make([]net.Conn, 0),
		responses:   make([]string, 0),
		inputs:      make([]string, 0),
		currentStep: 0,
		t:           t,
		dynamicData: make(chan string, 100),
	}
}

// Start starts the telnet server on a random port
func (ts *TestTelnetServer) Start() (int, error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	
	ts.listener = listener
	ts.port = listener.Addr().(*net.TCPAddr).Port
	
	go ts.handleConnections()
	
	ts.t.Logf("Test telnet server started on port %d", ts.port)
	return ts.port, nil
}

// GetAddress returns the server address
func (ts *TestTelnetServer) GetAddress() string {
	return fmt.Sprintf("localhost:%d", ts.port)
}

// SetResponses sets the pre-configured responses the server will send
func (ts *TestTelnetServer) SetResponses(responses []string) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	ts.responses = responses
	ts.currentStep = 0
}

// GetInputs returns all inputs received from the client
func (ts *TestTelnetServer) GetInputs() []string {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	return append([]string(nil), ts.inputs...)
}

// SendDynamicData sends data to all connected clients immediately
func (ts *TestTelnetServer) SendDynamicData(data string) {
	select {
	case ts.dynamicData <- data:
		ts.t.Logf("Queued dynamic data: %q", data)
	default:
		ts.t.Errorf("Dynamic data channel full, dropping: %q", data)
	}
}

// Stop stops the telnet server
func (ts *TestTelnetServer) Stop() {
	if ts.listener != nil {
		ts.listener.Close()
	}
	
	// Close dynamic data channel
	close(ts.dynamicData)
	
	ts.mutex.Lock()
	for _, conn := range ts.connections {
		conn.Close()
	}
	ts.connections = ts.connections[:0]
	ts.mutex.Unlock()
}

// handleConnections handles incoming telnet connections
func (ts *TestTelnetServer) handleConnections() {
	for {
		conn, err := ts.listener.Accept()
		if err != nil {
			return // Server closed
		}
		
		ts.mutex.Lock()
		ts.connections = append(ts.connections, conn)
		ts.mutex.Unlock()
		
		go ts.handleConnection(conn)
	}
}

// handleConnection handles a single telnet connection
func (ts *TestTelnetServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	
	ts.t.Logf("Test telnet client connected from %s", conn.RemoteAddr())
	
	// Send initial game prompt to simulate TWX game server
	ts.sendResponse(conn, "Trade Wars 2002\r\nEnter your login name: ")
	
	// Start goroutine to handle dynamic data
	go func() {
		for data := range ts.dynamicData {
			ts.t.Logf("Sending dynamic data to client: %q", data)
			ts.sendResponse(conn, data)
		}
	}()
	
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		input := scanner.Text()
		
		ts.mutex.Lock()
		ts.inputs = append(ts.inputs, input)
		ts.t.Logf("Test telnet received input: %q", input)
		
		// Send next pre-configured response if available
		if ts.currentStep < len(ts.responses) {
			response := ts.responses[ts.currentStep]
			ts.currentStep++
			ts.mutex.Unlock()
			
			ts.sendResponse(conn, response)
		} else {
			ts.mutex.Unlock()
			// Send a generic prompt to keep the connection alive
			ts.sendResponse(conn, "Command [TL=00:00:00]: ")
		}
	}
	
	ts.t.Logf("Test telnet client disconnected")
}

// sendResponse sends a response to the client with a small delay to simulate network latency
func (ts *TestTelnetServer) sendResponse(conn net.Conn, response string) {
	time.Sleep(10 * time.Millisecond) // Simulate network latency
	
	ts.t.Logf("Test telnet sending response: %q", response)
	fmt.Fprint(conn, response)
}

// MockTuiAPI implements api.TuiAPI for testing
type MockTuiAPI struct {
	connectionStatus api.ConnectionStatus
	connectionAddress string
	receivedData [][]byte
	errors []error
	mutex sync.Mutex
	t *testing.T
}

// NewMockTuiAPI creates a new mock TUI API for testing
func NewMockTuiAPI(t *testing.T) *MockTuiAPI {
	return &MockTuiAPI{
		t: t,
		receivedData: make([][]byte, 0),
		errors: make([]error, 0),
	}
}

func (m *MockTuiAPI) OnConnectionStatusChanged(status api.ConnectionStatus, address string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.connectionStatus = status
	m.connectionAddress = address
	m.t.Logf("MockTuiAPI: Connection status changed to %v for %s", status, address)
}

func (m *MockTuiAPI) OnConnectionError(err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.errors = append(m.errors, err)
	m.t.Logf("MockTuiAPI: Connection error: %v", err)
}

func (m *MockTuiAPI) OnData(data []byte) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	m.receivedData = append(m.receivedData, dataCopy)
	// Only log short data to avoid spam
	if len(data) < 100 {
		m.t.Logf("MockTuiAPI: Received data: %q", string(data))
	}
}

func (m *MockTuiAPI) OnScriptStatusChanged(status api.ScriptStatusInfo) {
	m.t.Logf("MockTuiAPI: Script status changed: %+v", status)
}

func (m *MockTuiAPI) OnScriptError(scriptName string, err error) {
	m.t.Logf("MockTuiAPI: Script error in %s: %v", scriptName, err)
}

func (m *MockTuiAPI) OnDatabaseStateChanged(info api.DatabaseStateInfo) {
	m.t.Logf("MockTuiAPI: Database state changed: %+v", info)
}

func (m *MockTuiAPI) OnCurrentSectorChanged(sectorInfo api.SectorInfo) {
	m.t.Logf("MockTuiAPI: Current sector changed: %+v", sectorInfo)
}

func (m *MockTuiAPI) OnTraderDataUpdated(sectorNumber int, traders []api.TraderInfo) {
	m.t.Logf("MockTuiAPI: Trader data updated for sector %d: %+v", sectorNumber, traders)
}

func (m *MockTuiAPI) OnPlayerStatsUpdated(stats api.PlayerStatsInfo) {
	m.t.Logf("MockTuiAPI: Player stats updated: %+v", stats)
}

func (m *MockTuiAPI) GetReceivedData() [][]byte {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return append([][]byte(nil), m.receivedData...)
}