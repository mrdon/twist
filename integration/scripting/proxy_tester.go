package scripting

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"twist/internal/api"
	"twist/internal/api/factory"
)

// ProxyResult contains the result of running a proxy test
type ProxyResult struct {
	Database     *sql.DB // Database instance for assertions
	ClientOutput string  // All output the client received
}

// Execute runs server and client scripts using the API for complete black-box testing
//
// Flow:
// 1. SERVER: Creates ExpectTelnetServer that runs serverScript (sends data to proxy)
// 2. PROXY: Created via api.Connect(), may load TWX script, processes data bidirectionally  
// 3. CLIENT: Creates SimpleExpectEngine that runs clientScript (expects data from proxy, sends user input)
//
func Execute(t *testing.T, serverScript, clientScript string, connectOpts *api.ConnectOptions) *ProxyResult {
	// 1. SERVER: Create and start telnet server with server script
	server := NewExpectTelnetServer(t)
	server.SetServerScript(serverScript)
	port, err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start telnet server: %v", err)
	}
	defer server.Stop()

	// If connectOpts contains a ScriptName that looks like script content, create a temp file
	if connectOpts != nil && connectOpts.ScriptName != "" && !strings.HasSuffix(connectOpts.ScriptName, ".ts") {
		// This looks like script content, not a filename - create temp file
		scriptPath := t.TempDir() + "/temp_script.ts"
		err := os.WriteFile(scriptPath, []byte(connectOpts.ScriptName), 0644)
		if err != nil {
			t.Fatalf("Failed to create temp script file: %v", err)
		}
		// Update connectOpts to use the file path
		connectOpts.ScriptName = scriptPath
	}

	// 2. PROXY: Create client expect engine and connect proxy
	clientExpectEngine := NewSimpleExpectEngine(t, nil, "\r")
	tuiAPI := &TestTuiAPI{expectEngine: clientExpectEngine}
	address := fmt.Sprintf("localhost:%d", port)
	proxyInstance := factory.Connect(address, tuiAPI, connectOpts)
	defer proxyInstance.Disconnect()

	// Set the input sender for client expect engine - this simulates user typing
	clientExpectEngine.inputSender = func(input string) {
		t.Logf("CLIENT EXPECT ENGINE SENDING USER INPUT: %q", input)
		// Send user input to the proxy, which forwards it to the server
		err := proxyInstance.SendData([]byte(input))
		if err != nil {
			t.Logf("Failed to send input: %v", err)
		}
	}

	// 3. CLIENT: Run client script (includes automatic sync token wait)
	err = clientExpectEngine.Run(clientScript)
	if err != nil {
		t.Fatalf("Client script failed: %v", err)
	}

	// Open database for return if path was specified
	var sqlDB *sql.DB
	if connectOpts != nil && connectOpts.DatabasePath != "" {
		sqlDB, err = sql.Open("sqlite3", connectOpts.DatabasePath)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		t.Logf("PROXY TESTER: Returning database at %s", connectOpts.DatabasePath)
	}

	return &ProxyResult{
		Database:     sqlDB,
		ClientOutput: clientExpectEngine.GetAllOutput(),
	}
}

// TestTuiAPI implements api.TuiAPI to capture proxy output for client expect engine
type TestTuiAPI struct {
	expectEngine *SimpleExpectEngine // This is the client expect engine
}

func (t *TestTuiAPI) OnConnectionStatusChanged(status api.ConnectionStatus, address string) {}
func (t *TestTuiAPI) OnConnectionError(err error)                                           {}
func (t *TestTuiAPI) OnData(data []byte) {
	if t.expectEngine != nil {
		t.expectEngine.AddOutput(string(data))
	}
}
func (t *TestTuiAPI) OnScriptStatusChanged(status api.ScriptStatusInfo)              {}
func (t *TestTuiAPI) OnScriptError(scriptName string, err error)                     {}
func (t *TestTuiAPI) OnDatabaseStateChanged(info api.DatabaseStateInfo)              {}
func (t *TestTuiAPI) OnCurrentSectorChanged(sectorInfo api.SectorInfo)               {}
func (t *TestTuiAPI) OnTraderDataUpdated(sectorNumber int, traders []api.TraderInfo) {}
func (t *TestTuiAPI) OnPlayerStatsUpdated(stats api.PlayerStatsInfo)                 {}

// ExpectTelnetServer - Telnet server with server-side expect script support for black-box testing
type ExpectTelnetServer struct {
	t              *testing.T
	listener       net.Listener
	connections    []net.Conn
	inputs         []string
	port           int
	mutex          sync.Mutex
	dynamicData    chan string
	serverScript   string
	serverEngine   *ServerExpectEngine
	scriptComplete chan error
}

// NewExpectTelnetServer creates a telnet server with server-side expect support
func NewExpectTelnetServer(t *testing.T) *ExpectTelnetServer {
	return &ExpectTelnetServer{
		t:              t,
		connections:    make([]net.Conn, 0),
		inputs:         make([]string, 0),
		dynamicData:    make(chan string, 100),
		scriptComplete: make(chan error, 1),
	}
}

// SetServerScript sets the server-side expect script
func (ets *ExpectTelnetServer) SetServerScript(script string) {
	ets.serverScript = script
	ets.t.Logf("SERVER SCRIPT SET:\n%s", script)
}

// Start starts the telnet server with expect script support
func (ets *ExpectTelnetServer) Start() (int, error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	ets.listener = listener
	ets.port = listener.Addr().(*net.TCPAddr).Port

	// Start connection handler
	go ets.handleConnections()

	ets.t.Logf("Expect telnet server started on port %d", ets.port)
	return ets.port, nil
}

// Stop stops the telnet server
func (ets *ExpectTelnetServer) Stop() {
	if ets.listener != nil {
		ets.listener.Close()
	}

	// Close dynamic data channel
	close(ets.dynamicData)

	ets.mutex.Lock()
	for _, conn := range ets.connections {
		conn.Close()
	}
	ets.connections = ets.connections[:0]
	ets.mutex.Unlock()
}

// GetInputs returns all inputs received from the client
func (ets *ExpectTelnetServer) GetInputs() []string {
	ets.mutex.Lock()
	defer ets.mutex.Unlock()
	return append([]string(nil), ets.inputs...)
}

// SendDynamicData sends data to all connected clients immediately
func (ets *ExpectTelnetServer) SendDynamicData(data string) {
	select {
	case ets.dynamicData <- data:
		ets.t.Logf("Queued dynamic data: %q", data)
	default:
		ets.t.Errorf("Dynamic data channel full, dropping: %q", data)
	}
}

// WaitForServerScript waits for the server script to complete
func (ets *ExpectTelnetServer) WaitForServerScript(timeout time.Duration) error {
	select {
	case err := <-ets.scriptComplete:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("server script timeout after %v", timeout)
	}
}

// handleConnections handles incoming connections with expect support
func (ets *ExpectTelnetServer) handleConnections() {
	for {
		conn, err := ets.listener.Accept()
		if err != nil {
			return // Server closed
		}

		ets.mutex.Lock()
		ets.connections = append(ets.connections, conn)
		ets.mutex.Unlock()

		go ets.handleConnection(conn)
	}
}

// handleConnection handles a single connection with server expect engine
func (ets *ExpectTelnetServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	ets.t.Logf("Expect telnet client connected from %s", conn.RemoteAddr())

	// Create server expect engine for this connection
	ets.serverEngine = NewServerExpectEngine(ets.t, conn)

	// Run server script in background if provided
	if ets.serverScript != "" {
		go func() {
			err := ets.serverEngine.RunServerScript(ets.serverScript)
			ets.scriptComplete <- err
		}()
	}

	// Handle dynamic data in background
	go func() {
		for data := range ets.dynamicData {
			ets.t.Logf("Sending dynamic data to client: %q", data)
			ets.sendResponse(conn, data)
		}
	}()

	// Read client input and feed to server expect engine
	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			ets.t.Logf("Client disconnected: %v", err)
			break
		}

		if n > 0 {
			input := string(buffer[:n])

			// Clean up telnet negotiation sequences
			cleanInput := ets.cleanTelnetInput(input)
			if cleanInput != "" {
				ets.mutex.Lock()
				ets.inputs = append(ets.inputs, cleanInput)
				ets.mutex.Unlock()

				ets.t.Logf("Expect telnet received input: %q", cleanInput)

				// Feed to server expect engine
				if ets.serverEngine != nil {
					ets.serverEngine.AddClientInput(cleanInput)
				}
			}
		}
	}

	ets.t.Logf("Expect telnet client disconnected")
}

// cleanTelnetInput removes telnet negotiation sequences and returns clean text
func (ets *ExpectTelnetServer) cleanTelnetInput(input string) string {
	// Remove common telnet negotiation sequences
	cleaned := input

	// Remove IAC sequences (0xFF followed by command bytes)
	var result strings.Builder
	i := 0
	for i < len(cleaned) {
		if i < len(cleaned) && cleaned[i] == '\xFF' {
			// Skip IAC sequence (usually 3 bytes: FF FB/FC/FD XX)
			if i+2 < len(cleaned) {
				i += 3
			} else {
				i = len(cleaned)
			}
		} else {
			result.WriteByte(cleaned[i])
			i++
		}
	}

	cleaned = result.String()

	// Remove control characters except printable ones
	result.Reset()
	for _, char := range cleaned {
		if char >= 32 && char < 127 || char == '\r' || char == '\n' {
			result.WriteRune(char)
		}
	}

	return strings.TrimSpace(result.String())
}

// sendResponse sends response to client
func (ets *ExpectTelnetServer) sendResponse(conn net.Conn, response string) {
	time.Sleep(10 * time.Millisecond)
	ets.t.Logf("Expect telnet sending response: %q", response)
	conn.Write([]byte(response))
}

// ServerExpectEngine runs expect scripts on the server side of telnet connection
type ServerExpectEngine struct {
	t            *testing.T
	conn         net.Conn
	inputCapture []string
	expectEngine *SimpleExpectEngine
}

// NewServerExpectEngine creates a server-side expect engine
func NewServerExpectEngine(t *testing.T, conn net.Conn) *ServerExpectEngine {
	serverEngine := &ServerExpectEngine{
		t:            t,
		conn:         conn,
		inputCapture: make([]string, 0),
	}

	// Create underlying expect engine with server-side input sender
	// Server sends "\r\n" for "*" since it's sending full protocol responses
	serverEngine.expectEngine = NewSimpleExpectEngine(t, func(data string) {
		serverEngine.sendToClient(data)
	}, "\r\n")

	return serverEngine
}

// sendToClient sends data to the connected client (proxy)
func (s *ServerExpectEngine) sendToClient(data string) {
	s.t.Logf("SERVER EXPECT SENDING TO CLIENT: %q", data)

	if s.conn != nil {
		// Add small delay to simulate network latency
		time.Sleep(10 * time.Millisecond)
		s.conn.Write([]byte(data))
	}
}

// AddClientInput adds input received from client to expect engine
func (s *ServerExpectEngine) AddClientInput(input string) {
	s.inputCapture = append(s.inputCapture, input)
	s.t.Logf("SERVER EXPECT RECEIVED FROM CLIENT: %q", input)

	if s.expectEngine != nil {
		s.expectEngine.AddOutput(input)
	}
}

// RunServerScript executes a server-side expect script
func (s *ServerExpectEngine) RunServerScript(script string) error {
	s.t.Logf("SERVER EXPECT RUNNING SCRIPT:\n%s", script)
	return s.expectEngine.Run(script)
}