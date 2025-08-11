package scripting

import (
	"bufio"
	"database/sql"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"twist/integration/setup"
	"twist/internal/api"
	"twist/internal/api/factory"
)

// ScriptLine represents a single line in the raw script format
type ScriptLine struct {
	Direction string // "<" for server, ">" for client
	Data      string // the raw data
}

// ProxyResult contains the result of running a proxy test
type ProxyResult struct {
	Database     *sql.DB          // Database instance for assertions
	Assert       *setup.DBAsserts // Database assertion helper
	ClientOutput string           // All output the client received
}

// ExecuteScriptFile runs a test script from a script file
func ExecuteScriptFile(t *testing.T, scriptFilePath string, connectOpts *api.ConnectOptions) *ProxyResult {
	scriptLines, err := LoadScriptFile(scriptFilePath)
	if err != nil {
		t.Fatalf("Failed to load test script from %s: %v", scriptFilePath, err)
	}

	serverScript, clientScript := ConvertToExpectScripts(scriptLines)
	return Execute(t, serverScript, clientScript, connectOpts)
}

// LoadScriptFile loads a test script from a script file
func LoadScriptFile(scriptFilePath string) ([]ScriptLine, error) {
	file, err := os.Open(scriptFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open script file: %w", err)
	}
	defer file.Close()

	var lines []ScriptLine
	scanner := bufio.NewScanner(file)

	// Regex to parse lines like: < data or > data
	lineRegex := regexp.MustCompile(`^([<>])\s+(.+)$`)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := lineRegex.FindStringSubmatch(line)
		if matches == nil {
			return nil, fmt.Errorf("invalid line format: %s", line)
		}

		direction := matches[1]
		data := matches[2]

		lines = append(lines, ScriptLine{
			Direction: direction,
			Data:      data,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading script file: %w", err)
	}

	return lines, nil
}

// ConvertToExpectScripts converts ScriptLines to server and client expect scripts with intelligent pattern generation
func ConvertToExpectScripts(scriptLines []ScriptLine) (serverScript, clientScript string) {
	var serverLines, clientLines []string

	for i, line := range scriptLines {
		if line.Direction == "<" {
			// Server sends data (keep literal string format)
			serverLines = append(serverLines, `send "`+line.Data+`"`)

			// Generate client expect only if next line is client data
			if i+1 < len(scriptLines) && scriptLines[i+1].Direction == ">" {
				// Generate expect pattern from last unique characters
				expectPattern := generateExpectPattern(line.Data)
				// Escape any actual ANSI sequences back to string literals for expect scripts
				escapedPattern := escapeANSIForExpect(expectPattern)
				clientLines = append(clientLines, `expect "`+escapedPattern+`"`)
			} else if i == len(scriptLines)-1 {
				// This is the last server message - always add sync mechanism
				// Generate client expect for the last server message to ensure processing completes
				expectPattern := generateExpectPattern(line.Data)
				escapedPattern := escapeANSIForExpect(expectPattern)
				clientLines = append(clientLines, `expect "`+escapedPattern+`"`)
				
				// Add sync token: server sends a unique marker, client expects it
				syncToken := "\\x1b[0m<SYNC_COMPLETE>\\x1b[0m"
				serverLines = append(serverLines, `send "`+syncToken+`"`)
				clientLines = append(clientLines, `expect "`+syncToken+`"`)
			}
		} else if line.Direction == ">" {
			// Client sends data (keep literal string format)
			clientLines = append(clientLines, `send "`+line.Data+`"`)

			// Check if next line is server data - if so, generate server expect
			if i+1 < len(scriptLines) && scriptLines[i+1].Direction == "<" {
				// Server expects exactly what client sends
				serverLines = append(serverLines, `expect "`+line.Data+`"`)
			}
		}
	}

	return strings.Join(serverLines, "\n"), strings.Join(clientLines, "\n")
}


// generateExpectPattern extracts a unique expect pattern from server data
// Takes last 10-20 characters and finds a good breaking point at ANSI sequences or control chars
func generateExpectPattern(serverData string) string {
	// Convert escaped sequences to actual characters but keep ANSI codes
	actualData, _ := strconv.Unquote("\"" + serverData + "\"")

	// If text is empty, return it as-is
	if len(actualData) == 0 {
		return actualData
	}

	// If text is short, return the whole string
	if len(actualData) <= 10 {
		return actualData
	}

	// Start from last 20 characters (or beginning if shorter) to give us room to find break points
	startSearchPos := len(actualData) - 20
	if startSearchPos < 0 {
		startSearchPos = 0
	}
	
	// Count visible characters from the end to find where we'd have 10 left
	visibleCount := 0
	tenCharsLeftPos := len(actualData)
	
	for i := len(actualData) - 1; i >= 0; i-- {
		// Skip ANSI sequences and control characters when counting
		if actualData[i] == '\x1b' {
			// We're at the end of an ANSI sequence, skip backwards to find start
			continue
		} else if actualData[i] == '\r' || actualData[i] == '\n' {
			// Skip control characters
			continue
		} else if i > 0 && actualData[i-1] == '\x1b' && actualData[i] == '[' {
			// We're at [ in \x1b[, skip
			continue
		} else if i > 1 && actualData[i-2] == '\x1b' && actualData[i-1] == '[' {
			// We're in the middle of an ANSI sequence
			continue
		}
		
		// This is a visible character
		visibleCount++
		if visibleCount == 10 {
			tenCharsLeftPos = i
			break
		}
	}

	// Now scan forward from startSearchPos to find the best break point
	// We prefer break points that give us more context, but only if they're significantly better
	bestBreakPoint := tenCharsLeftPos // Default to exactly 10 chars
	
	// Only look for break points in the first half of our search range
	// This ensures we don't choose ANSI sequences that are very close to the 10-char boundary
	midPoint := startSearchPos + (tenCharsLeftPos-startSearchPos)/2
	
	for i := startSearchPos; i < midPoint && i < len(actualData); i++ {
		// Look for good break points
		if actualData[i] == '\x1b' {
			// Start of ANSI sequence - this is a good break point if it gives us more context
			bestBreakPoint = i
			break
		} else if actualData[i] == '\r' || actualData[i] == '\n' {
			// Start of newline - this is a good break point
			bestBreakPoint = i  
			break
		}
	}

	return actualData[bestBreakPoint:]
}



// escapeANSIForExpect converts actual ANSI escape characters back to string literals
// This is needed when building expect scripts that contain ANSI sequences
func escapeANSIForExpect(text string) string {
	// Replace escape character and control characters with their string literals
	text = strings.ReplaceAll(text, "\x1b", "\\x1b")
	text = strings.ReplaceAll(text, "\r", "\\r")
	text = strings.ReplaceAll(text, "\n", "\\n")
	return text
}

// Execute runs server and client scripts using the API for complete black-box testing
//
// Flow:
// 1. SERVER: Creates ExpectTelnetServer that runs serverScript (sends data to proxy)
// 2. PROXY: Created via api.Connect(), may load TWX script, processes data bidirectionally
// 3. CLIENT: Creates SimpleExpectEngine that runs clientScript (expects data from proxy, sends user input)
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
	t.Logf("CLIENT SCRIPT SET:\n%s", clientScript)
	err = clientExpectEngine.Run(clientScript)
	if err != nil {
		t.Fatalf("Client script failed: %v", err)
	}

	// Open database for return if path was specified
	var sqlDB *sql.DB
	var dbAsserts *setup.DBAsserts
	if connectOpts != nil && connectOpts.DatabasePath != "" {
		sqlDB, err = sql.Open("sqlite3", connectOpts.DatabasePath)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		// Validate database was created and is accessible
		if sqlDB == nil {
			t.Fatal("Expected database instance to be created")
		}
		// Test database connection
		if err := sqlDB.Ping(); err != nil {
			t.Fatalf("Database connection failed: %v", err)
		}
		// Create database assertion helper
		dbAsserts = setup.NewDBAsserts(t, sqlDB)
	}

	return &ProxyResult{
		Database:     sqlDB,
		Assert:       dbAsserts,
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

	return result.String()
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
		// Process escape sequences to convert string literals to actual control characters
		processedData := processEscapeSequences(data)
		// Add small delay to simulate network latency
		time.Sleep(10 * time.Millisecond)
		s.conn.Write([]byte(processedData))
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
