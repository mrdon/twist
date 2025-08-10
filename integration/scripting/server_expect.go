package scripting

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// ServerExpectEngine runs expect scripts on the server side of telnet connection
// This simulates a realistic game server that responds to player input
type ServerExpectEngine struct {
	t              *testing.T
	conn           net.Conn
	inputCapture   []string
	expectEngine   *SimpleExpectEngine
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
	serverEngine.expectEngine = NewExpectEngine(t, func(data string) {
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

// Enhanced TestTelnetServer with server-side expect support
type ExpectTelnetServer struct {
	*TestTelnetServer
	serverScript   string
	serverEngine   *ServerExpectEngine
	scriptComplete chan error
}

// NewExpectTelnetServer creates a telnet server with server-side expect support
func NewExpectTelnetServer(t *testing.T) *ExpectTelnetServer {
	return &ExpectTelnetServer{
		TestTelnetServer: NewTestTelnetServer(t),
		scriptComplete:   make(chan error, 1),
	}
}

// SetServerScript sets the server-side expect script
func (ets *ExpectTelnetServer) SetServerScript(script string) {
	ets.serverScript = script
	ets.t.Logf("SERVER SCRIPT SET:\n%s", script)
}

// Start starts the telnet server with expect script support
func (ets *ExpectTelnetServer) Start() (int, error) {
	// We need to manually implement the start logic to use our connection handler
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	
	ets.TestTelnetServer.listener = listener
	ets.TestTelnetServer.port = listener.Addr().(*net.TCPAddr).Port
	
	// Start our custom connection handler
	go ets.handleConnections()
	
	ets.t.Logf("Expect telnet server started on port %d", ets.TestTelnetServer.port)
	return ets.TestTelnetServer.port, nil
}

// handleConnections handles incoming connections with expect support
func (ets *ExpectTelnetServer) handleConnections() {
	for {
		conn, err := ets.TestTelnetServer.listener.Accept()
		if err != nil {
			return // Server closed
		}
		
		ets.TestTelnetServer.mutex.Lock()
		ets.TestTelnetServer.connections = append(ets.TestTelnetServer.connections, conn)
		ets.TestTelnetServer.mutex.Unlock()
		
		go ets.handleConnection(conn)
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

// handleConnection overrides the base class to use server expect engine  
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
	} else {
		// Fallback to original behavior - send initial prompt
		ets.sendResponse(conn, "Trade Wars 2002\r\nEnter your login name: ")
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

// sendResponse sends response to client (keep original method signature)
func (ets *ExpectTelnetServer) sendResponse(conn net.Conn, response string) {
	time.Sleep(10 * time.Millisecond)
	ets.t.Logf("Expect telnet sending response: %q", response)
	conn.Write([]byte(response))
}